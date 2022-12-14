package handler

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.ibm.com/mbg-agent/cmd/cluster/state"
	"github.ibm.com/mbg-agent/pkg/protocol"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
)

func ExposeReq(serviceId, mbgIP string) {
	log.Printf("Clster Start expose %v to MBG with IP address %v", serviceId, mbgIP)
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

/************************** Http server function *************************************/
func (m ClusterHandler) exposePost(w http.ResponseWriter, r *http.Request) {
	//phrase hello struct from request
	var e protocol.ExposeRequest
	err := json.NewDecoder(r.Body).Decode(&e)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Infof("Received expose to service: %v", e.Id)

	//Update MBG state
	state.UpdateState()
	state.AddService(e.Id, e.Ip, e.Domain)

	//Response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Expose succeed"))
	if err != nil {
		log.Println(err)
	}
}
