package handler

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"

	"github.ibm.com/mbg-agent/pkg/protocol"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
)

func DisconnectReq(svcId, svcIdDest, mbgIP string) {
	log.Printf("Start disconnect Request to MBG %v for service %v:%v", mbgIP, svcId, svcIdDest)
	address := "http://" + mbgIP + "/connect"

	j, err := json.Marshal(protocol.DisconnectRequest{Id: svcId, IdDest: svcIdDest})
	if err != nil {
		log.Fatal(err)
	}
	//send expose
	resp := httpAux.HttpDelete(address, j)
	log.Infof(`Service %s disconnect for message: %s`, svcId, string(resp))
}

/************************** Http server function *************************************/
