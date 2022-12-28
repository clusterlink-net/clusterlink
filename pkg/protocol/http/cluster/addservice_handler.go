package handler

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"

	"github.ibm.com/mbg-agent/cmd/cluster/state"
	"github.ibm.com/mbg-agent/pkg/protocol"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
)

func AddServiceReq(serviceId string) {
	log.Printf("Cluster Start addService %v to ", serviceId)
	s := state.GetService(serviceId)
	mbgIP := state.GetMbgIP()
	svcExp := s.Service
	log.Printf("Service %v", s)

	address := "http://" + mbgIP + "/addservice"
	j, err := json.Marshal(protocol.AddServiceRequest{Id: svcExp.Id, Ip: svcExp.Ip, Domain: svcExp.Domain})
	if err != nil {
		log.Fatal(err)
	}
	//send expose
	resp := httpAux.HttpPost(address, j)
	log.Infof(`Response message for serive %s expose :  %s`, svcExp.Id, string(resp))
}
