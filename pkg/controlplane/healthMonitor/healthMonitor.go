package healthMonitor

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	apiObject "github.ibm.com/mbg-agent/pkg/controlplane/api/object"
	"github.ibm.com/mbg-agent/pkg/controlplane/eventManager"
	"github.ibm.com/mbg-agent/pkg/controlplane/store"
	httpUtils "github.ibm.com/mbg-agent/pkg/utils/http"
)

var klog = logrus.WithField("component", "controlPlane/HealthMonitor")

const (
	timeout  = 5 //seconds
	Interval = 1 * time.Second
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
		//}
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
	for {
		mList := store.GetMbgList()
		for _, m := range mList {
			url := store.GetAddrStart() + store.GetMbgTarget(m) + "/hb"
			_, err := httpUtils.HttpPost(url, j, store.GetHttpClient())

			if err != nil {
				klog.Errorf("Unable to send heartbeat to %s, Error: %v", url, err.Error())
				continue
			}
			updateLastSeen(m)
		}
		time.Sleep(Interval)
	}
}

func RecvHeartbeat(mbgID string) {
	updateLastSeen(mbgID)
	validateMBGs(mbgID)
}

func MonitorHeartBeats() {
	for {
		time.Sleep(Interval)
		store.UpdateState()
		mList := store.GetMbgList()
		for _, m := range mList {
			t := time.Now()
			lastSeen, ok := getLastSeen(m)
			if !ok {
				continue
			}
			diff := t.Sub(lastSeen)
			if diff.Seconds() > timeout {
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
