package handler

import (
	log "github.com/sirupsen/logrus"

	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
)

func HelloReq(mbgIP, peerID string) {
	log.Printf("Start hello from to MBG peer %v", peerID)

	address := "http://" + mbgIP + "/hello/" + peerID

	//send hello
	j := []byte{}
	resp := httpAux.HttpPost(address, j)
	log.Infof(`Response message hello to MBG peer(%s) :  %s`, peerID, string(resp))
}

func Hello2AllReq(mbgIP string) {
	log.Printf("Start hello to all MBG peers")

	address := "http://" + mbgIP + "/hello/"

	//send hello
	j := []byte{}
	resp := httpAux.HttpPost(address, j)
	log.Infof(`Response message hello to all MBG peers: %s`, string(resp))
}
