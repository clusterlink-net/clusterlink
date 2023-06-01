package mbgControlplane

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"

	"github.ibm.com/mbg-agent/cmd/controlplane/state"
	"github.ibm.com/mbg-agent/pkg/controlplane/eventManager"
	kubernetes "github.ibm.com/mbg-agent/pkg/k8s/kubernetes"
	"github.ibm.com/mbg-agent/pkg/protocol"
	httpUtils "github.ibm.com/mbg-agent/pkg/utils/http"
)

var mlog = logrus.WithField("component", "mbgControlPlane/Expose")

func Expose(e protocol.ExposeRequest) error {
	//Update MBG state
	state.UpdateState()
	return ExposeToMbg(e.Id, e.MbgID)
}

func ExposeToMbg(serviceId, peerId string) error {
	exposeResp, err := state.GetEventManager().RaiseExposeRequestEvent(eventManager.ExposeRequestAttr{Service: serviceId})
	if err != nil {
		return fmt.Errorf("Unable to raise expose request event")
	}
	mlog.Infof("Response = %+v", exposeResp)
	if exposeResp.Action == eventManager.Deny {
		mlog.Errorf("Denying Expose of service %s", serviceId)
		return fmt.Errorf("Denying Expose of service %s", serviceId)
	}

	myIp := state.GetMyIp()
	svcExp := state.GetLocalService(serviceId)
	if svcExp.Ip == "" {
		return fmt.Errorf("Denying Expose of service %s - target is not set", serviceId)
	}
	svcExp.Ip = myIp
	if peerId == "" { //Expose to all
		if exposeResp.Action == eventManager.AllowAll {
			for _, mbgId := range state.GetMbgList() {
				ExposeReq(svcExp, mbgId, "MBG")
			}
			return nil
		}
		for _, mbgId := range exposeResp.TargetMbgs {
			ExposeReq(svcExp, mbgId, "MBG")
		}
	} else { //Expose to specific peer
		if slices.Contains(exposeResp.TargetMbgs, peerId) {
			ExposeReq(svcExp, peerId, "MBG")
		}
	}
	return nil
}

func ExposeReq(svcExp state.LocalService, mbgId, cType string) {
	destIp := state.GetMbgTarget(mbgId)
	mlog.Printf("Starting to expose service %v (%v)", svcExp.Id, destIp)
	address := state.GetAddrStart() + destIp + "/remoteservice"

	j, err := json.Marshal(protocol.ExposeRequest{Id: svcExp.Id, Ip: svcExp.Ip, Description: svcExp.Description, MbgID: state.GetMyId()})
	if err != nil {
		mlog.Error(err)
		return
	}
	//Send expose
	resp, err := httpUtils.HttpPost(address, j, state.GetHttpClient())
	mlog.Infof("Service(%s) Expose Response message:  %s", svcExp.Id, string(resp))
	if string(resp) != httpUtils.RESPFAIL {
		state.AddPeerLocalService(svcExp.Id, mbgId)
	}
}

func CreateLocalServiceEndpoint(serviceId string, port int, name, namespace, mbgAppName string) error {
	sPort := state.GetConnectionArr()[serviceId].Local

	targetPort, err := strconv.Atoi(sPort[1:])
	if err != nil {
		return err
	}
	mlog.Infof("Creating service end point at %s:%d:%d for service %s", name, port, targetPort, serviceId)
	return kubernetes.Data.CreateServiceEndpoint(name, port, targetPort, namespace, mbgAppName)
}

func DeleteLocalServiceEndpoint(serviceId string) error {
	mlog.Infof("Deleting service end point at %s", serviceId)
	return kubernetes.Data.DeleteServiceEndpoint(serviceId)
}
