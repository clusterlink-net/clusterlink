package controlplane

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"

	apiObject "github.ibm.com/mbg-agent/pkg/controlplane/api/object"
	"github.ibm.com/mbg-agent/pkg/controlplane/store"
	"github.ibm.com/mbg-agent/pkg/utils/httputils"
)

var hlog = logrus.WithField("component", "mbgControlPlane/Hello")

// Send hello to peer - HTTP handler
func SendHelloHandler(w http.ResponseWriter, r *http.Request) {

	// Parse hello struct from request
	peerID := chi.URLParam(r, "peerID")

	// Hello control plane logic
	hlog.Infof("Send Hello to peer id: %v", peerID)
	resp := sendHello(peerID)

	// Set response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(apiObject.HelloResponse{Status: resp}); err != nil {
		hlog.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}
}

// Send hello to peer -control plane logic
func sendHello(mbgId string) string {
	MyInfo := store.GetMyInfo()
	ok := store.IsMbgPeer(mbgId)
	if ok {
		resp := helloReq(mbgId, MyInfo)
		return resp
	} else {
		hlog.Errorf("Unable to find peer %v in the peers list", mbgId)
		return httputils.RESPFAIL
	}
}

// Send hello to all peers - HTTP handler
func SendHelloToAllHandler(w http.ResponseWriter, r *http.Request) {

	// Hello control plane logic
	hlog.Infof("Send hello to peers")
	resp := sendHelloToAll()

	// Set response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(apiObject.HelloResponse{Status: resp}); err != nil {
		hlog.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}
}

// Send hello to all peers -control plane logic
func sendHelloToAll() string {
	MyInfo := store.GetMyInfo()
	for _, mbgId := range store.GetMbgList() {
		resp := helloReq(mbgId, MyInfo)
		if resp != httputils.RESPOK {
			return resp
		}
	}
	return httputils.RESPOK
}

// Send hello request(HTTP) to other peers
func helloReq(m string, myInfo store.MbgInfo) string {
	address := store.GetAddrStart() + store.GetMbgTarget(m) + "/peer/"

	hlog.Infof("Sending Hello message to peer at %v", address)

	j, err := json.Marshal(apiObject.PeerRequest{Id: myInfo.Id, Ip: myInfo.Ip, Cport: myInfo.Cport.External})
	if err != nil {
		hlog.Error(err)
		return err.Error()
	}
	// Send hello
	resp, _ := httputils.HttpPost(address, j, store.GetHttpClient())

	if string(resp) == httputils.RESPFAIL {
		return string(resp)
	} else {
		return httputils.RESPOK
	}

}
