package mbgControlplane

import (
	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/pkg/protocol"
)

//var mlog = logrus.WithField("component", "mbgControlPlane/AddService")

func AddService(s protocol.ServiceRequest) {
	state.UpdateState()
	state.AddLocalService(s.Id, s.Ip, s.Domain)
}

func GetService(svcId string) protocol.ServiceRequest {
	state.UpdateState()
	s := state.GetRemoteService(svcId).Service
	sPort := state.GetConnectionArr()[s.Id].External
	s.Ip = state.GetMyIp() + sPort
	return protocol.ServiceRequest{Id: s.Id, Ip: s.Ip, Domain: s.Domain}
}

func GetAllServices() map[string]protocol.ServiceRequest {
	state.UpdateState()
	sArr := make(map[string]protocol.ServiceRequest)

	for _, s := range state.GetRemoteServicesArr() {
		sPort := state.GetConnectionArr()[s.Service.Id].External
		sIp := state.GetMyIp() + sPort
		sArr[s.Service.Id] = protocol.ServiceRequest{Id: s.Service.Id, Ip: sIp, Domain: s.Service.Domain}
	}

	return sArr
}
