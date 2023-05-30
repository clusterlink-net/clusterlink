package mbgControlplane

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.ibm.com/mbg-agent/cmd/controlplane/state"
	"github.ibm.com/mbg-agent/pkg/eventManager"
	"github.ibm.com/mbg-agent/pkg/protocol"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
)

var klog = logrus.WithField("component", "mbgControlPlane/HealthMonitor")

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
	ok := state.IsMbgPeer(mbgId)
	if !ok {
		// klog.Infof("Update state before activating MBG %s", mbgId)
		// state.UpdateState()
		// ok = state.IsMbgPeer(mbgId)
		// if !ok {
		// Activate MBG only if its present in inactive list
		if state.IsMbgInactivePeer(mbgId) {
			state.ActivateMbg(mbgId)
		}
		//}
	}
}

// Send hello messages to peer MBGs every second
func SendHeartBeats() error {
	mbgLastSeen = make(map[string]time.Time)
	state.UpdateState()
	j, err := json.Marshal(protocol.HeartBeat{Id: state.GetMyId()})
	if err != nil {
		klog.Error(err)
		return fmt.Errorf("unable to marshal json for heartbeat")
	}
	for {
		mList := state.GetMbgList()
		for _, m := range mList {
			url := state.GetAddrStart() + state.GetMbgTarget(m) + "/hb"
			_, err := httpAux.HttpPost(url, j, state.GetHttpClient())

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
		state.UpdateState()
		mList := state.GetMbgList()
		for _, m := range mList {
			t := time.Now()
			lastSeen, ok := getLastSeen(m)
			if !ok {
				continue
			}
			diff := t.Sub(lastSeen)
			if diff.Seconds() > timeout {
				klog.Errorf("Heartbeat Timeout reached, Inactivating MBG %s(LastSeen:%v)", m, lastSeen)
				err := state.GetEventManager().RaiseRemovePeerEvent(eventManager.RemovePeerAttr{PeerMbg: m})
				if err != nil {
					plog.Errorf("Unable to raise remove peer event")
					return
				}
				state.InactivateMbg(m)
			}
		}
	}
}
