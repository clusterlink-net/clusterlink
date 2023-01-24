package mbgControlplane

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/pkg/protocol"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
)

var hlog = logrus.WithField("component", "mbgControlPlane/Hello")

//Send hello to other mbg using HelloReq
func SendHello(mbgId string) string {
	//Update MBG state
	MbgArr := state.GetMbgArr()
	MyInfo := state.GetMyInfo()
	m, ok := MbgArr[mbgId]
	if ok {
		resp := HelloReq(m, MyInfo)
		hlog.Infof("Finish send Hello to MBG %v", mbgId)
		return resp
	} else {
		hlog.Infof("MBG %v is not exist in the MBG peers list", mbgId)
		return httpAux.RESPFAIL
	}
}

//Send hello to all mbg peers using HelloReq
func SendHello2All() string {
	MbgArr := state.GetMbgArr()
	MyInfo := state.GetMyInfo()
	for _, m := range MbgArr {
		hlog.Info(m)
		resp := HelloReq(m, MyInfo)
		if resp != httpAux.RESPOK {
			return resp
		}
	}
	hlog.Infof("Finish sending Hello to all Mbgs")
	return httpAux.RESPOK
}

//send hello request(http) to other mbg
func HelloReq(m, myInfo state.MbgInfo) string {
	address := state.GetAddrStart() + m.Ip + m.Cport.External + "/peer/" + myInfo.Id
	hlog.Infof("Start Hello message to MBG with address %v", address)

	j, err := json.Marshal(protocol.PeerRequest{Id: myInfo.Id, Ip: myInfo.Ip, Cport: myInfo.Cport.External})
	if err != nil {
		hlog.Error(err)
		return err.Error()
	}
	//Send hello
	resp := httpAux.HttpPost(address, j, state.GetHttpClient())

	if string(resp) == httpAux.RESPFAIL {
		return string(resp)
	} else {
		return httpAux.RESPOK
	}

}
