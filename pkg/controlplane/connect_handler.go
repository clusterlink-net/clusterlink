package controlplane

import (
	"encoding/json"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"

	apiObject "github.ibm.com/mbg-agent/pkg/controlplane/api/object"
	dp "github.ibm.com/mbg-agent/pkg/dataplane"
)

func ConnectPostHandler(w http.ResponseWriter, r *http.Request) {

	//Phrase struct from request
	var c apiObject.ConnectRequest
	err := json.NewDecoder(r.Body).Decode(&c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//Connect data plane logic
	mbgIP := strings.Split(r.RemoteAddr, ":")[0]
	log.Infof("Received connect to service %s from MBG: %s", c.Id, mbgIP)
	connect, connectType, connectDest := dp.Connect(c, mbgIP, nil)

	log.Infof("Got {%+v, %+v, %+v} from connect \n", connect, connectType, connectDest)
	//Set Connect response
	respJson, err := json.Marshal(apiObject.ConnectReply{Connect: connect, ConnectType: connectType, ConnectDest: connectDest})
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

func HandleConnectHandler(w http.ResponseWriter, r *http.Request) {
	//Phrase struct from request
	var c apiObject.ConnectRequest
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
	allow, _, _ := dp.Connect(c, mbgIP, w)

	//Write response for error
	if allow != true {
		w.WriteHeader(http.StatusForbidden)
	}
}
