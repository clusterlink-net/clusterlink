package handler

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"

	"github.ibm.com/mbg-agent/cmd/cluster/state"
	"github.ibm.com/mbg-agent/pkg/protocol"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
)

func ExposeReq(serviceId, mbgIP string) {
	log.Printf("Cluster Start expose %v to MBG with IP address %v", serviceId, mbgIP)
	s := state.GetService(serviceId)
	svcExp := s.Service
	log.Printf("Service %v", s)

	address := "http://" + mbgIP + "/expose"
	j, err := json.Marshal(protocol.ExposeRequest{Id: svcExp.Id, Ip: svcExp.Ip, Domain: svcExp.Domain, MbgID: ""})
	if err != nil {
		log.Fatal(err)
	}
	//send expose
	resp := httpAux.HttpPost(address, j)
	log.Infof(`Response message for serive %s expose :  %s`, svcExp.Id, string(resp))
}
