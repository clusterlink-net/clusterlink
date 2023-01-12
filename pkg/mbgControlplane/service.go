package mbgControlplane

import (
	"github.ibm.com/mbg-agent/cmd/mbg/state"
	md "github.ibm.com/mbg-agent/pkg/mbgDataplane"
	"github.ibm.com/mbg-agent/pkg/protocol"
)

//var mlog = logrus.WithField("component", "mbgControlPlane/AddService")
/******************* Local Service ****************************************/
func AddLocalService(s protocol.ServiceRequest) {
	state.UpdateState()
	state.AddLocalService(s.Id, s.Ip)
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
		sArr[s.Service.Id] = protocol.ServiceRequest{Id: s.Service.Id, Ip: sIp}
	}

	return sArr
}

/******************* Local Service ****************************************/
func AddRemoteService(e protocol.ExposeRequest) {
	state.AddRemoteService(e.Id, e.Ip, e.MbgID)
	myServicePort, err := state.GetFreePorts(e.Id)
	if err != nil {
		mlog.Infof("")
	}
	targetMbgIP := state.GetMbgIP(e.MbgID)
	rootCA, certFile, keyFile := state.GetMyMbgCerts()
	mtlsPort := (state.GetMyMtlsPort()).External
	mbgTarget := targetMbgIP + mtlsPort
	mlog.Infof("Starting a local Service for remote service %s at %s->%s with certs(%s,%s,%s)", e.Id, myServicePort.Local, mbgTarget, rootCA, certFile, keyFile)
	go md.StartLocalService(e.Id, myServicePort.Local, mbgTarget, rootCA, certFile, keyFile)
}

func GetRemoteService(svcId string) protocol.ServiceRequest {
	state.UpdateState()
	s := state.GetRemoteService(svcId).Service
	sPort := state.GetConnectionArr()[s.Id].External
	s.Ip = state.GetMyIp() + sPort
	return protocol.ServiceRequest{Id: s.Id, Ip: s.Ip}
}

func GetAllRemoteServices() map[string]protocol.ServiceRequest {
	state.UpdateState()
	sArr := make(map[string]protocol.ServiceRequest)

	for _, s := range state.GetRemoteServicesArr() {
		sPort := state.GetConnectionArr()[s.Service.Id].External
		sIp := state.GetMyIp() + sPort
		sArr[s.Service.Id] = protocol.ServiceRequest{Id: s.Service.Id, Ip: sIp}
	}

	return sArr
}
