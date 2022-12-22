package mbgControlplane

import (
	"bytes"
	"encoding/json"
	"os"

	log "github.com/sirupsen/logrus"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/pkg/protocol"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
)

func Hello(h protocol.HelloRequest) {
	//Update MBG state
	state.UpdateState()
	certFile := "cert_" + h.Id + ".pem"
	keyFile := "key_" + h.Id + ".pem"
	err := os.WriteFile(certFile, h.CertData, 0644)
	if err != nil {
		log.Infof("Failed to write cert : %+v", err)
	}
	err = os.WriteFile(keyFile, h.KeyData, 0644)
	if err != nil {
		log.Infof("Failed to write key : %+v", err)
	}
	state.AddMbgNbr(h.Id, h.Ip, h.Cport, certFile, keyFile)

}

//send hello request(http) to other mbg
func HelloReq(m, myInfo state.MbgInfo) {
	address := "http://" + m.Ip + ":" + m.Cport.External + "/hello"
	log.Infof("Start Hello message to MBG with address %v", address)

	j, err := json.Marshal(protocol.HelloRequest{Id: myInfo.Id, Ip: myInfo.Ip, Cport: myInfo.Cport.External, CertData: myInfo.CertData, KeyData: myInfo.KeyData})
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
	certFile := "cert_" + m.Id + ".pem"
	keyFile := "key_" + m.Id + ".pem"
	err = os.WriteFile(certFile, h.CertData, 0644)
	if err != nil {
		log.Infof("Failed to write cert : %+v", err)
	}
	err = os.WriteFile(keyFile, h.KeyData, 0644)
	if err != nil {
		log.Infof("Failed to write key : %+v", err)
	}
	state.UpdateMbgCerts(m.Id, certFile, keyFile)
}
