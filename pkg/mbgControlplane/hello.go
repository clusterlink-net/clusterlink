package mbgControlplane

import (
	"bytes"
	"encoding/json"

	log "github.com/sirupsen/logrus"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/pkg/protocol"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
)

func Hello(h protocol.HelloRequest) {
	//Update MBG state
	state.UpdateState()
	state.AddMbgNbr(h.Id, h.Ip, h.Cport)
}

//send hello request(http) to other mbg
func HelloReq(m, myInfo state.MbgInfo) {
	address := "http://" + m.Ip + ":" + m.Cport.External + "/hello"
	log.Infof("Start Hello message to MBG with address %v", address)

	j, err := json.Marshal(protocol.HelloRequest{Id: myInfo.Id, Ip: myInfo.Ip, Cport: myInfo.Cport.External})
	if err != nil {
		log.Fatal(err)
	}
	//Send hello
	resp := httpAux.HttpPost(address, j)

	var h protocol.HelloResponse
	err = json.NewDecoder(bytes.NewBuffer(resp)).Decode(&h)
	if err != nil {
		log.Infof("Unable to decode response %v", err)
	}
	log.Infof(`Response message for Hello:  %s`, h.Status)
}
