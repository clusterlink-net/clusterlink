package mbgControlplane

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
	"github.ibm.com/mbg-agent/cmd/controlplane/state"
	"github.ibm.com/mbg-agent/pkg/eventManager"
	"github.ibm.com/mbg-agent/pkg/protocol"
	httpUtils "github.ibm.com/mbg-agent/pkg/utils/http"
)

var plog = logrus.WithField("component", "mbgControlPlane/Peer")

func AddPeer(p protocol.PeerRequest) {
	//Update MBG state
	state.UpdateState()

	peerResp, err := state.GetEventManager().RaiseAddPeerEvent(eventManager.AddPeerAttr{PeerMbg: p.Id})
	if err != nil {
		plog.Errorf("Unable to raise connection request event")
		return
	}
	if peerResp.Action == eventManager.Deny {
		plog.Infof("Denying add peer(%s) due to policy", p.Id)
		return
	}
	state.AddMbgNbr(p.Id, p.Ip, p.Cport)
}

func GetAllPeers() map[string]protocol.PeerRequest {
	//Update MBG state
	state.UpdateState()
	pArr := make(map[string]protocol.PeerRequest)

	for _, s := range state.GetMbgList() {
		ip, port := state.GetMbgTargetPair(s)
		pArr[s] = protocol.PeerRequest{Id: s, Ip: ip, Cport: port}
	}
	return pArr

}

func GetPeer(peerID string) protocol.PeerRequest {
	//Update MBG state
	state.UpdateState()
	ok := state.IsMbgPeer(peerID)
	if ok {
		ip, port := state.GetMbgTargetPair(peerID)
		return protocol.PeerRequest{Id: peerID, Ip: ip, Cport: port}
	} else {
		plog.Infof("MBG %s does not exist in the peers list ", peerID)
		return protocol.PeerRequest{}
	}

}

func RemovePeer(p protocol.PeerRemoveRequest) {
	//Update MBG state
	state.UpdateState()

	err := state.GetEventManager().RaiseRemovePeerEvent(eventManager.RemovePeerAttr{PeerMbg: p.Id})
	if err != nil {
		plog.Errorf("Unable to raise connection request event")
		return
	}
	if p.Propagate {
		// Remove this MBG from the remove MBG's peer list
		peerIP := state.GetMbgTarget(p.Id)
		address := state.GetAddrStart() + peerIP + "/peer/" + state.GetMyId()
		j, err := json.Marshal(protocol.PeerRemoveRequest{Id: state.GetMyId(), Propagate: false})
		if err != nil {
			return
		}
		httpUtils.HttpDelete(address, j, state.GetHttpClient())
	}

	// Remove remote MBG from current MBG's peer
	state.RemoveMbg(p.Id)
	RemoveLastSeen(p.Id)
}
