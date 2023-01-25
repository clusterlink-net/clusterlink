package eventManager

import (
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
)

var elog = logrus.WithField("component", "EventManager")

type MbgEventManager struct {
	PolicyDispatcherTarget string //URL for now
	httpClient             http.Client
}

func (m *MbgEventManager) RaiseNewConnectionRequestEvent(connectionAttr ConnectionRequestAttr) (ConnectionRequestResp, error) {
	// Send the event to PolicyDispatcher
	url := m.PolicyDispatcherTarget + "/" + NewConnectionRequest
	if m.PolicyDispatcherTarget != "" {
		elog.Infof("Sending to PolicyDispatcher : %s", m.PolicyDispatcherTarget)
		jsonReq, err := json.Marshal(connectionAttr)
		if err != nil {
			elog.Errorf("Unable to marshal json %v", err)
			return ConnectionRequestResp{Action: Allow, TargetMbg: "", BitRate: 0}, err
		}
		resp := httpAux.HttpPost(url, jsonReq, m.httpClient)
		var r ConnectionRequestResp
		err = json.Unmarshal(resp, &r)
		if err != nil {
			elog.Errorf("Unable to unmarshal json %v", err)
			return ConnectionRequestResp{Action: Allow, TargetMbg: "", BitRate: 0}, err
		}
		return r, nil
	} else {
		// No Policy Dispatcher assigned
		return ConnectionRequestResp{Action: Allow, TargetMbg: "", BitRate: 0}, nil
	}
}

func (m *MbgEventManager) RaiseNewRemoteServiceEvent(remoteServiceAttr NewRemoteServiceAttr) (NewRemoteServiceResp, error) {
	elog.Infof("New Remote Service Event %+v", remoteServiceAttr)
	url := m.PolicyDispatcherTarget + "/" + NewRemoteService
	if m.PolicyDispatcherTarget != "" {
		elog.Infof("Sending to PolicyDispatcher : %s", m.PolicyDispatcherTarget)
		jsonReq, err := json.Marshal(remoteServiceAttr)
		if err != nil {
			elog.Errorf("Unable to marshal json %v", err)
			return NewRemoteServiceResp{Action: Allow}, err
		}
		resp := httpAux.HttpPost(url, jsonReq, m.httpClient)
		var r NewRemoteServiceResp
		err = json.Unmarshal(resp, &r)
		if err != nil {
			elog.Errorf("Unable to unmarshal json %v", err)
			return NewRemoteServiceResp{Action: Allow}, err
		}
		return r, nil
	} else {
		// No Policy Dispatcher assigned
		return NewRemoteServiceResp{Action: Allow}, nil
	}
}

func (m *MbgEventManager) RaiseExposeRequestEvent(exposeRequestAttr ExposeRequestAttr) (ExposeRequestResp, error) {
	elog.Infof("New Expose Event %+v", exposeRequestAttr)
	url := m.PolicyDispatcherTarget + "/" + ExposeRequest
	// Send the event to PolicyDispatcher
	if m.PolicyDispatcherTarget != "" {
		elog.Infof("Sending to PolicyDispatcher : %s", m.PolicyDispatcherTarget)
		jsonReq, err := json.Marshal(exposeRequestAttr)
		if err != nil {
			elog.Errorf("Unable to marshal json %v", err)
			return ExposeRequestResp{Action: Allow}, err
		}
		resp := httpAux.HttpPost(url, jsonReq, m.httpClient)
		var r ExposeRequestResp
		err = json.Unmarshal(resp, &r)
		if err != nil {
			elog.Errorf("Unable to unmarshal json %v", err)
			return ExposeRequestResp{Action: Allow}, err
		}
		return r, nil
	} else {
		// No Policy Dispatcher assigned
		return ExposeRequestResp{Action: Allow}, nil
	}
}

func (m *MbgEventManager) RaiseAddPeerEvent(addPeerAttr AddPeerAttr) (AddPeerResp, error) {
	elog.Infof("Add Peer MBG Event %+v", addPeerAttr)
	url := m.PolicyDispatcherTarget + "/" + AddPeerRequest
	// Send the event to PolicyDispatcher
	if m.PolicyDispatcherTarget != "" {
		elog.Infof("Sending to PolicyDispatcher : %s", m.PolicyDispatcherTarget)
		jsonReq, err := json.Marshal(addPeerAttr)
		if err != nil {
			elog.Errorf("Unable to marshal json %v", err)
			return AddPeerResp{Action: Allow}, err
		}
		resp := httpAux.HttpPost(url, jsonReq, m.httpClient)
		var r AddPeerResp
		err = json.Unmarshal(resp, &r)
		if err != nil {
			elog.Errorf("Unable to unmarshal json %v", err)
			return AddPeerResp{Action: Allow}, err
		}
		return r, nil
	} else {
		// No Policy Dispatcher assigned
		return AddPeerResp{Action: Allow}, nil
	}
}

func (m *MbgEventManager) RaiseServiceListRequestEvent(serviceListRequestAttr ServiceListRequestAttr) (ServiceListRequestResp, error) {
	elog.Infof("Service List Event %+v", serviceListRequestAttr)
	return ServiceListRequestResp{Action: Allow, Services: nil}, nil
}

func (m *MbgEventManager) RaiseServiceRequestEvent(serviceRequestAttr ServiceRequestAttr) (ServiceRequestResp, error) {
	elog.Infof("Service Request Event %+v", serviceRequestAttr)
	return ServiceRequestResp{Action: Allow}, nil
}

func (m *MbgEventManager) AssignPolicyDispatcher(targetUrl string) {
	m.PolicyDispatcherTarget = targetUrl
	m.httpClient = http.Client{}
}
