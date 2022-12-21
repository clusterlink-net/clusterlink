package mbgControlplane

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"

	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/pkg/protocol"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
	service "github.ibm.com/mbg-agent/pkg/serviceMap"
)

func Expose(e protocol.ExposeRequest) {
	//Update MBG state
	state.UpdateState()
	if e.Domain == "Internal" {
		state.AddLocalService(e.Id, e.Ip, e.Domain)
		ExposeToMbg(e.Id)
	} else { //Got the service from MBG so expose to local Cluster
		state.AddRemoteService(e.Id, e.Ip, e.Domain, e.MbgID)
		ExposeToCluster(e.Id)
		// myServicePort, err := state.GetFreeLocalPort(e.Id)
		//md.StartClusterService()
	}

}

func ExposeToMbg(serviceId string) {
	MbgArr := state.GetMbgArr()
	myIp := state.GetMyIp()

	s := state.GetLocalService(serviceId)
	svcExp := s.Service
	svcExp.Ip = myIp
	svcExp.Domain = "Remote"
	for _, m := range MbgArr {
		destIp := m.Ip + ":" + m.Cport.External
		ExposeReq(svcExp, destIp, "MBG")
	}
}

func ExposeToCluster(serviceId string) {
	clusterArr := state.GetLocalClusterArr()
	myIp := state.GetMyIp()
	s := state.GetRemoteService(serviceId)
	svcExp := s.Service
	svcExp.Ip = myIp
	svcExp.Domain = "Remote"

	for _, g := range clusterArr {
		destIp := g.Ip
		ExposeReq(svcExp, destIp, "Gateway")
	}
}

func ExposeReq(svcExp service.Service, destIp, cType string) {
	log.Printf("Start expose %v to %v with IP address %v", svcExp.Id, cType, destIp)
	address := "http://" + destIp + "/expose"

	j, err := json.Marshal(protocol.ExposeRequest{Id: svcExp.Id, Ip: svcExp.Ip, Domain: svcExp.Domain, MbgID: state.GetMyId()})
	if err != nil {
		log.Fatal(err)
	}
	//Send expose
	resp := httpAux.HttpPost(address, j)
	log.Infof(`Response message for serive %s expose :  %s`, svcExp.Id, string(resp))
}
