// Copyright 2023 The ClusterLink Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package health

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	apiObject "github.com/clusterlink-net/clusterlink/pkg/controlplane/api/object"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/eventmanager"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/store"
	"github.com/clusterlink-net/clusterlink/pkg/utils/httputils"
)

var klog = logrus.WithField("component", "controlPlane/health")

const (
	interval = 1 * time.Second // Interval for sending liveness/health checks
	timeout  = 5 * time.Second
)

var mbgLastSeenMutex sync.RWMutex
var mbgLastSeen map[string]time.Time

func updateLastSeen(mbgID string) {
	mbgLastSeenMutex.Lock()
	mbgLastSeen[mbgID] = time.Now()
	mbgLastSeenMutex.Unlock()
}

func RemoveLastSeen(mbgID string) {
	mbgLastSeenMutex.Lock()
	delete(mbgLastSeen, mbgID)
	mbgLastSeenMutex.Unlock()
}

func getLastSeen(mbgID string) (time.Time, bool) {
	mbgLastSeenMutex.RLock()
	lastSeen, ok := mbgLastSeen[mbgID]
	mbgLastSeenMutex.RUnlock()
	return lastSeen, ok
}

func validateMBGs(mbgID string) {
	ok := store.IsMbgPeer(mbgID)
	if !ok {
		// klog.Infof("Update state before activating MBG %s", mbgID)
		// store.UpdateState()
		// ok = store.IsMbgPeer(mbgID)
		// if !ok {
		// Activate MBG only if its present in inactive list
		if store.IsMbgInactivePeer(mbgID) {
			store.ActivateMbg(mbgID)
		}
		// }
	}
}

// Send hello messages to peer MBGs every second
func SendHeartBeats() error {
	mbgLastSeen = make(map[string]time.Time)
	store.UpdateState()
	j, err := json.Marshal(apiObject.HeartBeat{ID: store.GetMyID()})
	if err != nil {
		klog.Error(err)
		return fmt.Errorf("unable to marshal json for heartbeat")
	}
	head := store.GetAddrStart()
	httpclient := store.GetHTTPClient()
	for {
		mList := store.GetMbgList()
		for _, m := range mList {
			url := head + store.GetMbgTarget(m) + "/hb"
			_, err := httputils.Post(url, j, httpclient)
			if err != nil {
				klog.Errorf("Unable to send heartbeat to %s, Error: %v", url, err.Error())
				continue
			}
			updateLastSeen(m)
		}
		time.Sleep(interval)
	}
}

// Send HB to peer http handler
func HandleHB(w http.ResponseWriter, r *http.Request) {
	var h apiObject.HeartBeat
	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&h)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	RecvHeartbeat(h.ID)

	// Response
	w.WriteHeader(http.StatusOK)
}

func RecvHeartbeat(mbgID string) {
	updateLastSeen(mbgID)
	validateMBGs(mbgID)
}

func MonitorHeartBeats() {
	for {
		time.Sleep(timeout)
		store.UpdateState()
		mList := store.GetMbgList()
		for _, m := range mList {
			t := time.Now()
			lastSeen, ok := getLastSeen(m)
			if !ok {
				continue
			}
			elapsed := t.Sub(lastSeen)
			if elapsed > timeout {
				klog.Errorf("Heartbeat Timeout reached, Inactivating MBG %s(LastSeen:%v)", m, lastSeen)
				err := store.GetEventManager().RaiseRemovePeerEvent(eventmanager.RemovePeerAttr{PeerMbg: m})
				if err != nil {
					klog.Errorf("Unable to raise remove peer event")
					return
				}
				store.InactivateMbg(m)
			}
		}
	}
}
