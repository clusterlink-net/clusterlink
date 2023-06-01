package controlplane

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
	apiObject "github.ibm.com/mbg-agent/pkg/controlplane/api/object"
	"github.ibm.com/mbg-agent/pkg/controlplane/eventManager"
	"github.ibm.com/mbg-agent/pkg/controlplane/healthMonitor"
	"github.ibm.com/mbg-agent/pkg/controlplane/store"
	httpUtils "github.ibm.com/mbg-agent/pkg/utils/http"
)

var plog = logrus.WithField("component", "mbgControlPlane/Peer")

func AddPeer(p apiObject.PeerRequest) {
	//Update MBG state
	store.UpdateState()

	peerResp, err := store.GetEventManager().RaiseAddPeerEvent(eventManager.AddPeerAttr{PeerMbg: p.Id})
	if err != nil {
		plog.Errorf("Unable to raise connection request event")
		return
	}
	if peerResp.Action == eventManager.Deny {
		plog.Infof("Denying add peer(%s) due to policy", p.Id)
		return
	}
	store.AddMbgNbr(p.Id, p.Ip, p.Cport)
}

func GetAllPeers() map[string]apiObject.PeerRequest {
	//Update MBG state
	store.UpdateState()
	pArr := make(map[string]apiObject.PeerRequest)

	for _, s := range store.GetMbgList() {
		ip, port := store.GetMbgTargetPair(s)
		pArr[s] = apiObject.PeerRequest{Id: s, Ip: ip, Cport: port}
	}
	return pArr

}

func GetPeer(peerID string) apiObject.PeerRequest {
	//Update MBG state
	store.UpdateState()
	ok := store.IsMbgPeer(peerID)
	if ok {
		ip, port := store.GetMbgTargetPair(peerID)
		return apiObject.PeerRequest{Id: peerID, Ip: ip, Cport: port}
	} else {
		plog.Infof("MBG %s does not exist in the peers list ", peerID)
		return apiObject.PeerRequest{}
	}

}

func RemovePeer(p apiObject.PeerRemoveRequest) {
	//Update MBG state
	store.UpdateState()

	err := store.GetEventManager().RaiseRemovePeerEvent(eventManager.RemovePeerAttr{PeerMbg: p.Id})
	if err != nil {
		plog.Errorf("Unable to raise connection request event")
		return
	}
	if p.Propagate {
		// Remove this MBG from the remove MBG's peer list
		peerIP := store.GetMbgTarget(p.Id)
		address := store.GetAddrStart() + peerIP + "/peer/" + store.GetMyId()
		j, err := json.Marshal(apiObject.PeerRemoveRequest{Id: store.GetMyId(), Propagate: false})
		if err != nil {
			return
		}
		httpUtils.HttpDelete(address, j, store.GetHttpClient())
	}

	// Remove remote MBG from current MBG's peer
	store.RemoveMbg(p.Id)
	healthMonitor.RemoveLastSeen(p.Id)
}
