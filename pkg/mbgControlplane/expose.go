package mbgControlplane

import (
	"encoding/json"

	"github.com/sirupsen/logrus"

	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/pkg/protocol"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
	service "github.ibm.com/mbg-agent/pkg/serviceMap"
)

var mlog = logrus.WithField("component", "mbgControlPlane/Expose")

func Expose(e protocol.ExposeRequest) {
	//Update MBG state
	state.UpdateState()
	state.AddLocalService(e.Id, e.Ip)
	ExposeToMbg(e.Id)
}

func ExposeToMbg(serviceId string) {
	MbgArr := state.GetMbgArr()
	myIp := state.GetMyIp()

	s := state.GetLocalService(serviceId)
	svcExp := s.Service
	svcExp.Ip = myIp
	for _, m := range MbgArr {
		destIp := m.Ip + ":" + m.Cport.External
		ExposeReq(svcExp, destIp, "MBG")
	}
}

func ExposeReq(svcExp service.Service, destIp, cType string) {
	mlog.Printf("Start expose %v to %v with IP address %v", svcExp.Id, cType, destIp)
	address := "http://" + destIp + "/remoteservice"

	j, err := json.Marshal(protocol.ExposeRequest{Id: svcExp.Id, Ip: svcExp.Ip, MbgID: state.GetMyId()})
	if err != nil {
		mlog.Error(err)
		return
	}
	//Send expose
	resp := httpAux.HttpPost(address, j)
	mlog.Infof(`Response message for serive %s expose :  %s`, svcExp.Id, string(resp))
}
