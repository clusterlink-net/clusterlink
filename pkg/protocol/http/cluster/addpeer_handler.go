package handler

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"

	"github.ibm.com/mbg-agent/cmd/cluster/state"
	"github.ibm.com/mbg-agent/pkg/protocol"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
)

func AddPeerReq(peerId, peerIp, peerCport string) {
	mbgIP := state.GetMbgIP()
	address := "http://" + mbgIP + "/peer/" + peerId
	j, err := json.Marshal(protocol.PeerRequest{Id: peerId, Ip: peerIp, Cport: peerCport})
	if err != nil {
		log.Fatal(err)
	}
	//send expose
	resp := httpAux.HttpPost(address, j)
	log.Infof(`Response message for adding MBG peer %s command : %s`, peerId, string(resp))
}
