package mbgControlplane

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/pkg/eventManager"
	"github.ibm.com/mbg-agent/pkg/protocol"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
)

var klog = logrus.WithField("component", "mbgControlPlane/HealthMonitor")

const (
	timeout  = 5 //seconds
	Interval = 1 * time.Second
)

// Send hello messages to peer MBGs every second
func SendHeartBeats() error {
	state.UpdateState()
	j, err := json.Marshal(protocol.HeartBeat{Id: state.GetMyId()})
	if err != nil {
		klog.Error(err)
		return fmt.Errorf("unable to marshal json for heartbeat")
	}
	monitorStarted := false
	for {
		MbgArr := state.GetMbgArr()
		for _, m := range MbgArr {
			url := state.GetAddrStart() + m.Ip + m.Cport.External + "/hello/hb"
			resp := httpAux.HttpPost(url, j, state.GetHttpClient())

			if string(resp) == httpAux.RESPFAIL {
				klog.Errorf("Unable to send heartbeat to %s", url)
				continue
			}
			state.UpdateLastSeen(m.Id)
		}
		// Start monitoring only atleast after a round of sending heartbeats
		if !monitorStarted {
			monitorStarted = true
			go MonitorHeartBeats()
		}
		time.Sleep(Interval)
	}
}

func RecvHeartbeat(mbgID string) {
	state.UpdateLastSeen(mbgID)
}

func MonitorHeartBeats() {
	for {
		state.UpdateState()
		MbgArr := state.GetMbgArr()
		for _, m := range MbgArr {
			t := time.Now()
			lastSeen := state.GetLastSeen(m.Id)
			diff := t.Sub(lastSeen)
			if diff.Seconds() > timeout {
				klog.Errorf("Heartbeat Timeout reached, Inactivating MBG %s(LastSeen:%v)", m.Id, lastSeen)
				err := state.GetEventManager().RaiseRemovePeerEvent(eventManager.RemovePeerAttr{PeerMbg: m.Id})
				if err != nil {
					plog.Errorf("Unable to raise remove peer event")
					return
				}
				state.MbgInactive(m)
			}
		}
		time.Sleep(Interval)
	}
}
