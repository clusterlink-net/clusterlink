package eventManager

import (
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
)

var elog = logrus.WithField("component", "EventManager")

type MbgEventManager struct {
	PolicyDispatcherTarget string      //URL for now
	HttpClient             http.Client `json:"-"`
}

var HttpClient http.Client

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
		resp, err := httpAux.HttpPost(url, jsonReq, m.HttpClient)

		var r ConnectionRequestResp
		err = json.Unmarshal(resp, &r)
		if err != nil {
			elog.Errorf("Unable to unmarshal ConnectionRequestResp json %v", err)
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
		resp, err := httpAux.HttpPost(url, jsonReq, m.HttpClient)
		if err != nil {
			return NewRemoteServiceResp{Action: Allow}, err
		}
		var r NewRemoteServiceResp
		err = json.Unmarshal(resp, &r)
		if err != nil {
			elog.Errorf("Unable to unmarshal RaiseNewRemoteServiceEvent json %v", err)
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
		resp, err := httpAux.HttpPost(url, jsonReq, m.HttpClient)
		if err != nil {
			return ExposeRequestResp{Action: Allow}, err
		}
		var r ExposeRequestResp
		err = json.Unmarshal(resp, &r)
		if err != nil {
			elog.Errorf("Unable to unmarshal RaiseExposeRequestEvent json %v", err)
			return ExposeRequestResp{Action: Allow}, err
		}
		return r, nil
	} else {
		// No Policy Dispatcher assigned
		return ExposeRequestResp{Action: AllowAll}, nil
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
		resp, err := httpAux.HttpPost(url, jsonReq, m.HttpClient)
		if err != nil {
			elog.Errorf("Unable to unmarshal RaiseAddPeerEvent json %v", err)
			return AddPeerResp{Action: Allow}, err
		}
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

func (m *MbgEventManager) RaiseRemovePeerEvent(removePeerAttr RemovePeerAttr) error {
	elog.Infof("Remove Peer MBG Event %+v", removePeerAttr)
	url := m.PolicyDispatcherTarget + "/" + RemovePeerRequest
	// Send the event to PolicyDispatcher
	if m.PolicyDispatcherTarget != "" {
		elog.Infof("Sending to PolicyDispatcher : %s", m.PolicyDispatcherTarget)
		jsonReq, err := json.Marshal(removePeerAttr)
		if err != nil {
			elog.Errorf("Unable to marshal json %v", err)
			return err
		}
		_, err = httpAux.HttpPost(url, jsonReq, m.HttpClient)
		if err != nil {
			elog.Errorf("Unable to send to Policy dispatcher %s", url)
		}
		return nil
	} else {
		// No Policy Dispatcher assigned
		return nil
	}
}

func (m *MbgEventManager) RaiseRemoveRemoteServiceEvent(removeRemoteServiceAttr RemoveRemoteServiceAttr) error {
	elog.Infof("Remove Remote service Event %+v", removeRemoteServiceAttr)
	url := m.PolicyDispatcherTarget + "/" + RemoveRemoteService
	// Send the event to PolicyDispatcher
	if m.PolicyDispatcherTarget != "" {
		elog.Infof("Sending to PolicyDispatcher : %s", m.PolicyDispatcherTarget)
		jsonReq, err := json.Marshal(removeRemoteServiceAttr)
		if err != nil {
			elog.Errorf("Unable to marshal json %v", err)
			return err
		}
		resp, _ := httpAux.HttpPost(url, jsonReq, m.HttpClient)
		if string(resp) == httpAux.RESPFAIL {
			elog.Errorf("Unable to send to Policy dispatcher %s", url)
		}
		return nil
	} else {
		// No Policy Dispatcher assigned
		elog.Infof("No PolicyDispatcher ")
		return nil
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

func (m *MbgEventManager) AssignPolicyDispatcher(targetUrl string, httpClient http.Client) {
	m.PolicyDispatcherTarget = targetUrl
	m.HttpClient = httpClient
	elog.Infof("PolicyDispatcher Target = %+v, httpclient=%+v", m.PolicyDispatcherTarget, HttpClient)
}
