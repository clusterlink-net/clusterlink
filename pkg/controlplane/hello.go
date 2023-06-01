package mbgControlplane

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
	"github.ibm.com/mbg-agent/pkg/controlplane/store"
	"github.ibm.com/mbg-agent/pkg/protocol"
	httpUtils "github.ibm.com/mbg-agent/pkg/utils/http"
)

var hlog = logrus.WithField("component", "mbgControlPlane/Hello")

// Send hello to other mbg using HelloReq
func SendHello(mbgId string) string {
	//Update MBG state
	MyInfo := store.GetMyInfo()
	ok := store.IsMbgPeer(mbgId)
	if ok {
		resp := HelloReq(mbgId, MyInfo)
		return resp
	} else {
		hlog.Errorf("Unable to find MBG %v in the peers list", mbgId)
		return httpUtils.RESPFAIL
	}
}

// Send hello to all mbg peers using HelloReq
func SendHello2All() string {
	MyInfo := store.GetMyInfo()
	for _, mbgId := range store.GetMbgList() {
		resp := HelloReq(mbgId, MyInfo)
		if resp != httpUtils.RESPOK {
			return resp
		}
	}
	return httpUtils.RESPOK
}

// send hello request(http) to other mbg
func HelloReq(m string, myInfo store.MbgInfo) string {
	address := store.GetAddrStart() + store.GetMbgTarget(m) + "/peer/" + myInfo.Id
	hlog.Infof("Sending Hello message to MBG at %v", address)

	j, err := json.Marshal(protocol.PeerRequest{Id: myInfo.Id, Ip: myInfo.Ip, Cport: myInfo.Cport.External})
	if err != nil {
		hlog.Error(err)
		return err.Error()
	}
	//Send hello
	resp, _ := httpUtils.HttpPost(address, j, store.GetHttpClient())

	if string(resp) == httpUtils.RESPFAIL {
		return string(resp)
	} else {
		return httpUtils.RESPOK
	}

}
