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
	mbgIP := strings.Split(r.RemoteAddr, ":")[0]
	log.Infof("Received connect to service %s from MBG: %s", c.Id, mbgIP)
	connect, connectType, connectDest := mbgDataplane.Connect(c, mbgIP, nil)

	log.Errorf("Got {%+v, %+v, %+v} from connect ", connect, connectType, connectDest)
	//Set Connect response
	respJson, err := json.Marshal(protocol.ConnectReply{Connect: connect, ConnectType: connectType, ConnectDest: connectDest})
	if err != nil {
		log.Errorf("Unable to marshal json:%+v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, err = w.Write(respJson)
	if err != nil {
		log.Println(err)
	}
}

func (m MbgHandler) handleConnect(w http.ResponseWriter, r *http.Request) {
	//Phrase struct from request
	var c protocol.ConnectRequest
	err := json.NewDecoder(r.Body).Decode(&c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//Connect control plane logic
	log.Infof("Received connect to service: %v", c.Id)

	//connection logic
	mbgIP := strings.Split(r.RemoteAddr, ":")[0]
	log.Infof("Received connect to service %s from MBG: %s", c.Id, mbgIP)
	allow, _, _ := mbgDataplane.Connect(c, mbgIP, w)

	//Write response for error
	if allow != true {
		w.WriteHeader(http.StatusForbidden)
	}
}
