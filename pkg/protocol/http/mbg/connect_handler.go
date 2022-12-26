package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.ibm.com/mbg-agent/pkg/mbgDataplane"
	"github.ibm.com/mbg-agent/pkg/protocol"
)

func (m MbgHandler) connectPost(w http.ResponseWriter, r *http.Request) {

	//Phrase struct from request
	var c protocol.ConnectRequest
	err := json.NewDecoder(r.Body).Decode(&c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//Connect data plane logic
	mbgIP := strings.Split(r.RemoteAddr, ":")
	log.Infof("Received connect to service %s from MBG: %s", c.Id, mbgIP[0])
	message, connectType, connectDest := mbgDataplane.Connect(c, mbgIP[0])

	//Set Connect response
	respJson, err := json.Marshal(protocol.ConnectReply{Message: message, ConnectType: connectType, ConnectDest: connectDest})
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, err = w.Write(respJson)
	if err != nil {
		log.Println(err)
	}
}
