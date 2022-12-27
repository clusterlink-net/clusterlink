package handler

import (
	"encoding/json"
	"net/http"

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
	//mbgIP := strings.Split(r.RemoteAddr, ":")
	//log.Infof("Received connect to service %s from MBG: %s", c.Id, mbgIP[0])
	message, connectType, connectDest := mbgDataplane.Connect(c, nil)

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

func (m MbgHandler) connectConnect(w http.ResponseWriter, r *http.Request) {
	//Phrase struct from request
	var c protocol.ConnectRequest
	err := json.NewDecoder(r.Body).Decode(&c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//Connect control plane logic
	log.Infof("Received connect to service: %v", c.Id)
	//Check if we can hijack connection
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "server doesn't support hijacking", http.StatusInternalServerError)
		return
	}
	//Write response
	w.WriteHeader(http.StatusOK)
	//Hijack the connection
	conn, _, err := hj.Hijack()
	//connection logic
	message, connectType, connectDest := mbgDataplane.Connect(c, conn)

	log.Info("Result from connect handler:", message, connectType, connectDest)

}
