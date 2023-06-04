package controlplane

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
	apiObject "github.ibm.com/mbg-agent/pkg/controlplane/api/object"
	"github.ibm.com/mbg-agent/pkg/controlplane/store"
	"github.ibm.com/mbg-agent/pkg/utils/httputils"
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
		return httputils.RESPFAIL
	}
}

// Send hello to all mbg peers using HelloReq
func SendHello2All() string {
	MyInfo := store.GetMyInfo()
	for _, mbgId := range store.GetMbgList() {
		resp := HelloReq(mbgId, MyInfo)
		if resp != httputils.RESPOK {
			return resp
		}
	}
	return httputils.RESPOK
}

// send hello request(http) to other mbg
func HelloReq(m string, myInfo store.MbgInfo) string {
	address := store.GetAddrStart() + store.GetMbgTarget(m) + "/peer/" + myInfo.Id
	hlog.Infof("Sending Hello message to MBG at %v", address)

	j, err := json.Marshal(apiObject.PeerRequest{Id: myInfo.Id, Ip: myInfo.Ip, Cport: myInfo.Cport.External})
	if err != nil {
		hlog.Error(err)
		return err.Error()
	}
	//Send hello
	resp, _ := httputils.HttpPost(address, j, store.GetHttpClient())

	if string(resp) == httputils.RESPFAIL {
		return string(resp)
	} else {
		return httputils.RESPOK
	}

}
