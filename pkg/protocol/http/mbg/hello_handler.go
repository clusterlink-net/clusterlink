package handler

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/pkg/mbgControlplane"
	"github.ibm.com/mbg-agent/pkg/protocol"
)

func (m MbgHandler) helloGet(w http.ResponseWriter, r *http.Request) {

	//Marshal or convert MyInfo object back to json and write to response
	myInfo := state.GetMyInfo()
	userJson, err := json.Marshal(protocol.HelloRequest{Id: myInfo.Id, Ip: myInfo.Ip, Cport: myInfo.Cport.External})
	if err != nil {
		panic(err)
	}

	//Set Content-Type header so that clients will know how to read response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	//Write json response back to response
	_, err = w.Write(userJson)
	if err != nil {
		log.Println(err)
	}
}

func (m MbgHandler) helloPost(w http.ResponseWriter, r *http.Request) {

	//phrase hello struct from request
	var h protocol.HelloRequest
	err := json.NewDecoder(r.Body).Decode(&h)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	//Hello control plane logic
	log.Infof("Received Hello from MBG ip: %v", h.Ip)
	mbgControlplane.Hello(h)

	//Response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Hello succeed"))
	if err != nil {
		log.Println(err)
	}
}
