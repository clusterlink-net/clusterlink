package eventManager

import (
	"github.com/sirupsen/logrus"
)

var elog = logrus.WithField("component", "EventManager")

type MbgEventManager struct {
	PolicyDispatcherTarget string //URL for now
}

func (m *MbgEventManager) RaiseNewConnectionRequestEvent(connectionAttr ConnectionRequestAttr) (ConnectionRequestResp, error) {
	if connectionAttr.Direction == Incoming {
		elog.Infof("New Incoming Connection Request Event %+v", connectionAttr)
	} else {
		elog.Infof("New Outgoing Connection Request Event %+v", connectionAttr)
	}
	// Send the event to PolicyDispatcher
	elog.Infof("Sending to %s", m.PolicyDispatcherTarget)
	return ConnectionRequestResp{Action: Allow, TargetMbg: "", BitRate: 0}, nil
}

func (m *MbgEventManager) RaiseNewRemoteServiceEvent(remoteServiceAttr NewRemoteServiceAttr) (NewRemoteServiceResp, error) {
	elog.Infof("New Remote Service Event %+v", remoteServiceAttr)
	return NewRemoteServiceResp{Action: Allow}, nil
}

func (m *MbgEventManager) RaiseExposeRequestEvent(exposeRequestAttr ExposeRequestAttr) (ExposeRequestResp, error) {
	elog.Infof("New Remote Service Event %+v", exposeRequestAttr)
	return ExposeRequestResp{Action: AllowAll, TargetMbgs: nil}, nil
}

func (m *MbgEventManager) RaiseAddPeerEvent(AddPeerAttr AddPeerAttr) (AddPeerResp, error) {
	elog.Infof("Add Peer MBG Event %+v", AddPeerAttr)
	return AddPeerResp{Action: Allow}, nil
}

func (m *MbgEventManager) RaiseServiceListRequestEvent(serviceListRequestAttr ServiceListRequestAttr) (ServiceListRequestResp, error) {
	elog.Infof("Service List Event %+v", serviceListRequestAttr)
	return ServiceListRequestResp{Action: Allow, Services: nil}, nil
}

func (m *MbgEventManager) RaiseServiceRequestEvent(serviceRequestAttr ServiceRequestAttr) (ServiceRequestResp, error) {
	elog.Infof("Service List Event %+v", serviceRequestAttr)
	return ServiceRequestResp{Action: Allow}, nil
}

func (m *MbgEventManager) AssignPolicyDispatcher(targetUrl string) {
	m.PolicyDispatcherTarget = targetUrl
}
