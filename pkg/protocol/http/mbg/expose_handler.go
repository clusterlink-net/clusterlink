package handler

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/pkg/protocol"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
	service "github.ibm.com/mbg-agent/pkg/serviceMap"
)

/***************************************************/
//Expose functions
func (m MbgHandler) exposePost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		//phrase hello struct from request
		var e protocol.ExposeRequest
		err := json.NewDecoder(r.Body).Decode(&e)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return

		}
		log.Infof("Received expose to service: %v", e.Id)

		//Update MBG satae
		state.UpdateState()
		if e.Domain == "Internal" {
			state.AddLocalService(e.Id, e.Ip, e.Domain)
			ExposeToMbg(e.Id)
		} else { //Got the service from MBG so expose to local Cluster
			state.AddRemoteService(e.Id, e.Ip, e.Domain, e.MbgID)
			ExposeToCluster(e.Id)
		}

		//Response
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("Expose succeed"))
		if err != nil {
			log.Println(err)
		}
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
		expose(svcExp, destIp, "MBG")
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
		expose(svcExp, destIp, "Gateway")
	}
}

func expose(svcExp service.Service, destIp, cType string) {
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
