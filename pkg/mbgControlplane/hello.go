package mbgControlplane

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/pkg/protocol"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
)

var hlog = logrus.WithField("component", "mbgControlPlane/Hello")

// Send hello to other mbg using HelloReq
func SendHello(mbgId string) string {
	//Update MBG state
	MyInfo := state.GetMyInfo()
	ok := state.IsMbgPeer(mbgId)
	if ok {
		resp := HelloReq(mbgId, MyInfo)
		return resp
	} else {
		hlog.Errorf("Unable to find MBG %v in the peers list", mbgId)
		return httpAux.RESPFAIL
	}
}

// Send hello to all mbg peers using HelloReq
func SendHello2All() string {
	MyInfo := state.GetMyInfo()
	for _, mbgId := range state.GetMbgList() {
		resp := HelloReq(mbgId, MyInfo)
		if resp != httpAux.RESPOK {
			return resp
		}
	}
	return httpAux.RESPOK
}

// send hello request(http) to other mbg
func HelloReq(m string, myInfo state.MbgInfo) string {
	address := state.GetAddrStart() + state.GetMbgTarget(m) + "/peer/" + myInfo.Id
	hlog.Infof("Sending Hello message to MBG at %v", address)

	j, err := json.Marshal(protocol.PeerRequest{Id: myInfo.Id, Ip: myInfo.Ip, Cport: myInfo.Cport.External})
	if err != nil {
		hlog.Error(err)
		return err.Error()
	}
	//Send hello
	resp, _ := httpAux.HttpPost(address, j, state.GetHttpClient())

	if string(resp) == httpAux.RESPFAIL {
		return string(resp)
	} else {
		return httpAux.RESPOK
	}

}
