package mbgControlplane

import (
	log "github.com/sirupsen/logrus"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/pkg/eventManager"
	"github.ibm.com/mbg-agent/pkg/protocol"
)

func AddPeer(p protocol.PeerRequest) {
	//Update MBG state
	state.UpdateState()

	peerResp, err := state.GetEventManager().RaiseAddPeerEvent(eventManager.AddPeerAttr{PeerMbg: p.Id})
	if err != nil {
		log.Errorf("[MBG %v] Unable to raise connection request event", state.GetMyId())
		return
	}
	if peerResp.Action == eventManager.Deny {
		log.Infof("Denying add peer(%s) due to policy", p.Id)
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
		log.Infof("MBG %s is not exist in the MBG peers list ", peerID)
		return protocol.PeerRequest{}
	}

}
