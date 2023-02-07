package mbgControlplane

import (
	"github.com/sirupsen/logrus"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/pkg/eventManager"
	"github.ibm.com/mbg-agent/pkg/protocol"
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

	for _, s := range state.GetMbgArr() {
		pArr[s.Id] = protocol.PeerRequest{Id: s.Id, Ip: s.Ip, Cport: s.Cport.External}
	}
	return pArr

}

func GetPeer(peerID string) protocol.PeerRequest {
	//Update MBG state
	state.UpdateState()
	MbgArr := state.GetMbgArr()
	m, ok := MbgArr[peerID]
	if ok {
		return protocol.PeerRequest{Id: m.Id, Ip: m.Ip, Cport: m.Cport.External}
	} else {
		plog.Infof("MBG %s does not exist in the peers list ", peerID)
		return protocol.PeerRequest{}
	}

}
