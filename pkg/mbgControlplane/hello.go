package mbgControlplane

import (
	"bytes"
	"encoding/json"

	"github.com/sirupsen/logrus"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/pkg/protocol"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
)

var hlog = logrus.WithField("component", "mbgControlPlane/Hello")

//Send hello to other mbg using HelloReq
func SendHello(mbgId string) {
	//Update MBG state
	MbgArr := state.GetMbgArr()
	MyInfo := state.GetMyInfo()
	m, ok := MbgArr[mbgId]
	if ok {
		HelloReq(m, MyInfo)
		hlog.Infof("Finish send Hello to MBG %v", mbgId)
	} else {
		hlog.Infof("MBG %v is not exist in the MBG peers list", mbgId)
	}
}

//Send hello to all mbg peers using HelloReq
func SendHello2All() {
	MbgArr := state.GetMbgArr()
	MyInfo := state.GetMyInfo()
	for _, m := range MbgArr {
		hlog.Info(m)
		HelloReq(m, MyInfo)
	}
	hlog.Infof("Finish sending Hello to all Mbgs")
}

//send hello request(http) to other mbg
func HelloReq(m, myInfo state.MbgInfo) {
	address := "http://" + m.Ip + ":" + m.Cport.External + "/peer/" + myInfo.Id
	hlog.Infof("Start Hello message to MBG with address %v", address)

	j, err := json.Marshal(protocol.PeerRequest{Id: myInfo.Id, Ip: myInfo.Ip, Cport: myInfo.Cport.External})
	if err != nil {
		hlog.Error(err)
		return
	}
	//Send hello
	resp := httpAux.HttpPost(address, j)

	var h protocol.HelloResponse
	err = json.NewDecoder(bytes.NewBuffer(resp)).Decode(&h)
	if err != nil {
		hlog.Infof("Unable to decode response %v", err)
	}
	hlog.Infof(`Response message for Hello:  %s`, h.Status)
}
