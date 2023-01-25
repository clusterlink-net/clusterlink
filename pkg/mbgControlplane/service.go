package mbgControlplane

import (
	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/pkg/eventManager"
	md "github.ibm.com/mbg-agent/pkg/mbgDataplane"
	"github.ibm.com/mbg-agent/pkg/protocol"
)

//var mlog = logrus.WithField("component", "mbgControlPlane/AddService")
/******************* Local Service ****************************************/
func AddLocalService(s protocol.ServiceRequest) {
	state.UpdateState()
	state.AddLocalService(s.Id, s.Ip, s.Description)
}

func GetLocalService(svcId string) protocol.ServiceRequest {
	state.UpdateState()
	s := state.GetLocalService(svcId).Service
	return protocol.ServiceRequest{Id: s.Id, Ip: s.Ip}
}

func GetAllLocalServices() map[string]protocol.ServiceRequest {
	state.UpdateState()
	sArr := make(map[string]protocol.ServiceRequest)

	for _, s := range state.GetLocalServicesArr() {
		sPort := state.GetConnectionArr()[s.Service.Id].External
		sIp := state.GetMyIp() + sPort
		sArr[s.Service.Id] = protocol.ServiceRequest{Id: s.Service.Id, Ip: sIp, Description: s.Service.Description}
	}

	return sArr
}

/******************* Remote Service ****************************************/
func AddRemoteService(e protocol.ExposeRequest) {
	state.AddRemoteService(e.Id, e.Ip, e.Description, e.MbgID)

	policyResp, err := state.GetEventManager().RaiseNewRemoteServiceEvent(eventManager.NewRemoteServiceAttr{Service: e.Id, Mbg: e.MbgID})
	if err != nil {
		mlog.Errorf("[MBG %v] Unable to raise connection request event", state.GetMyId())
		return
	}
	if policyResp.Action == eventManager.Deny {
		return
	}

	myServicePort, err := state.GetFreePorts(e.Id)
	if err != nil {
		mlog.Errorf("Unable to get free port")
		return
	}
	mbgTarget := state.GetMbgTarget(e.MbgID)
	rootCA, certFile, keyFile := state.GetMyMbgCerts()
	mlog.Infof("Starting a local Service for remote service %s at %s->%s with certs(%s,%s,%s)", e.Id, myServicePort.Local, mbgTarget, rootCA, certFile, keyFile)
	go md.StartProxyRemoteService(e.Id, myServicePort.Local, mbgTarget, rootCA, certFile, keyFile)
}

func GetRemoteService(svcId string) protocol.ServiceRequest {
	state.UpdateState()
	s := state.GetRemoteService(svcId)
	sPort := state.GetConnectionArr()[s.Service.Id].External
	sIp := state.GetMyIp() + sPort
	return protocol.ServiceRequest{Id: s.Service.Id, Ip: sIp, MbgID: s.MbgId, Description: s.Service.Description}
}

func GetAllRemoteServices() map[string]protocol.ServiceRequest {
	state.UpdateState()
	sArr := make(map[string]protocol.ServiceRequest)

	for _, s := range state.GetRemoteServicesArr() {
		sPort := state.GetConnectionArr()[s.Service.Id].External
		sIp := state.GetMyIp() + sPort
		sArr[s.Service.Id] = protocol.ServiceRequest{Id: s.Service.Id, Ip: sIp, MbgID: s.MbgId, Description: s.Service.Description}
	}

	return sArr
}
