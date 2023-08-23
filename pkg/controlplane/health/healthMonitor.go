package health

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	apiObject "github.ibm.com/mbg-agent/pkg/controlplane/api/object"
	"github.ibm.com/mbg-agent/pkg/controlplane/eventManager"
	"github.ibm.com/mbg-agent/pkg/controlplane/store"
	"github.ibm.com/mbg-agent/pkg/utils/httputils"
)

var klog = logrus.WithField("component", "controlPlane/health")

const (
	interval = 1 * time.Second // Interval for sending liveness/health checks
	timeout  = 5 * time.Second
)

var mbgLastSeenMutex sync.RWMutex
var mbgLastSeen map[string]time.Time

func updateLastSeen(mbgId string) {
	mbgLastSeenMutex.Lock()
	mbgLastSeen[mbgId] = time.Now()
	mbgLastSeenMutex.Unlock()
}

func RemoveLastSeen(mbgId string) {
	mbgLastSeenMutex.Lock()
	delete(mbgLastSeen, mbgId)
	mbgLastSeenMutex.Unlock()
}

func getLastSeen(mbgId string) (time.Time, bool) {
	mbgLastSeenMutex.RLock()
	lastSeen, ok := mbgLastSeen[mbgId]
	mbgLastSeenMutex.RUnlock()
	return lastSeen, ok
}

func validateMBGs(mbgId string) {
	ok := store.IsMbgPeer(mbgId)
	if !ok {
		// klog.Infof("Update state before activating MBG %s", mbgId)
		// store.UpdateState()
		// ok = store.IsMbgPeer(mbgId)
		// if !ok {
		// Activate MBG only if its present in inactive list
		if store.IsMbgInactivePeer(mbgId) {
			store.ActivateMbg(mbgId)
		}
		// }
	}
}

// Send hello messages to peer MBGs every second
func SendHeartBeats() error {
	mbgLastSeen = make(map[string]time.Time)
	store.UpdateState()
	j, err := json.Marshal(apiObject.HeartBeat{Id: store.GetMyId()})
	if err != nil {
		klog.Error(err)
		return fmt.Errorf("unable to marshal json for heartbeat")
	}
	head := store.GetAddrStart()
	httpclient := store.GetHttpClient()
	for {
		mList := store.GetMbgList()
		for _, m := range mList {
			url := head + store.GetMbgTarget(m) + "/hb"
			_, err := httputils.HttpPost(url, j, httpclient)
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

	RecvHeartbeat(h.Id)

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
				err := store.GetEventManager().RaiseRemovePeerEvent(eventManager.RemovePeerAttr{PeerMbg: m})
				if err != nil {
					klog.Errorf("Unable to raise remove peer event")
					return
				}
				store.InactivateMbg(m)
			}
		}
	}
}
