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

func (m *MbgEventManager) AssignPolicyDispatcher(targetUrl string) {
	m.PolicyDispatcherTarget = targetUrl
}
