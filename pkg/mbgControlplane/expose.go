package mbgControlplane

import (
	"encoding/json"

	"github.com/sirupsen/logrus"

	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/pkg/eventManager"
	"github.ibm.com/mbg-agent/pkg/protocol"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
	service "github.ibm.com/mbg-agent/pkg/serviceMap"
)

var mlog = logrus.WithField("component", "mbgControlPlane/Expose")

func Expose(e protocol.ExposeRequest) {
	//Update MBG state
	state.UpdateState()
	ExposeToMbg(e.Id)
}

func ExposeToMbg(serviceId string) {
	exposeResp, err := state.GetEventManager().RaiseExposeRequestEvent(eventManager.ExposeRequestAttr{Service: serviceId})
	if err != nil {
		mlog.Errorf("Unable to raise expose request event")
		return
	}
	mlog.Infof("Response = %+v", exposeResp)
	if exposeResp.Action == eventManager.Deny {
		mlog.Errorf("Denying Expose of service %s", serviceId)
		return
	}

	myIp := state.GetMyIp()
	s := state.GetLocalService(serviceId)
	svcExp := s.Service
	svcExp.Ip = myIp
	if exposeResp.Action == eventManager.AllowAll {
		MbgArr := state.GetMbgArr()
		for _, m := range MbgArr {
			destIp := m.Ip + m.Cport.External
			ExposeReq(svcExp, destIp, "MBG")
		}
	} else {
		for _, mbgId := range exposeResp.TargetMbgs {
			mbgAddr := state.GetMbgControlTarget(mbgId)
			ExposeReq(svcExp, mbgAddr, "MBG")
		}
	}
}

func ExposeReq(svcExp service.Service, destIp, cType string) {
	mlog.Printf("Starting to expose service %v (%v)", svcExp.Id, destIp)
	address := state.GetAddrStart() + destIp + "/remoteservice"

	j, err := json.Marshal(protocol.ExposeRequest{Id: svcExp.Id, Ip: svcExp.Ip, Description: svcExp.Description, MbgID: state.GetMyId()})
	if err != nil {
		mlog.Error(err)
		return
	}
	//Send expose
	resp := httpAux.HttpPost(address, j, state.GetHttpClient())
	mlog.Infof("Service(%s) Expose Response message:  %s", svcExp.Id, string(resp))
}
