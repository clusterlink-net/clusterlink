package mbgControlplane

import (
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"

	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/pkg/eventManager"
	"github.ibm.com/mbg-agent/pkg/protocol"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
)

var mlog = logrus.WithField("component", "mbgControlPlane/Expose")

func Expose(e protocol.ExposeRequest) error {
	//Update MBG state
	state.UpdateState()
	return ExposeToMbg(e.Id, e.MbgID)
}

func ExposeToMbg(serviceId, peerId string) error {
	exposeResp, err := state.GetEventManager().RaiseExposeRequestEvent(eventManager.ExposeRequestAttr{Service: serviceId})
	if err != nil {
		return fmt.Errorf("Unable to raise expose request event")
	}
	mlog.Infof("Response = %+v", exposeResp)
	if exposeResp.Action == eventManager.Deny {
		mlog.Errorf("Denying Expose of service %s", serviceId)
		return fmt.Errorf("Denying Expose of service %s", serviceId)
	}

	myIp := state.GetMyIp()
	svcExp := state.GetLocalService(serviceId)
	if svcExp.Ip == "" {
		return fmt.Errorf("Denying Expose of service %s - target is not set", serviceId)
	}
	svcExp.Ip = myIp
	if peerId == "" { //Expose to all
		if exposeResp.Action == eventManager.AllowAll {
			for _, mbgId := range state.GetMbgList() {
				ExposeReq(svcExp, mbgId, "MBG")
			}
			return nil
		}
		for _, mbgId := range exposeResp.TargetMbgs {
			ExposeReq(svcExp, mbgId, "MBG")
		}
	} else { //Expose to specific peer
		if slices.Contains(exposeResp.TargetMbgs, peerId) {
			ExposeReq(svcExp, peerId, "MBG")
		}
	}
	return nil
}

func ExposeReq(svcExp state.LocalService, mbgId, cType string) {
	destIp := state.GetMbgTarget(mbgId)
	mlog.Printf("Starting to expose service %v (%v)", svcExp.Id, destIp)
	address := state.GetAddrStart() + destIp + "/remoteservice"

	j, err := json.Marshal(protocol.ExposeRequest{Id: svcExp.Id, Ip: svcExp.Ip, Description: svcExp.Description, MbgID: state.GetMyId()})
	if err != nil {
		mlog.Error(err)
		return
	}
	//Send expose
	resp, err := httpAux.HttpPost(address, j, state.GetHttpClient())
	mlog.Infof("Service(%s) Expose Response message:  %s", svcExp.Id, string(resp))
	if string(resp) != httpAux.RESPFAIL {
		state.AddPeerLocalService(svcExp.Id, mbgId)
	}
}
