package mbgControlplane

import (
	"encoding/json"
	"strconv"

	"github.com/sirupsen/logrus"

	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/pkg/deployment/kubernetes"
	"github.ibm.com/mbg-agent/pkg/eventManager"
	"github.ibm.com/mbg-agent/pkg/protocol"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
)

var mlog = logrus.WithField("component", "mbgControlPlane/Expose")

func Expose(e protocol.ExposeRequest) {
	//Update MBG state
	state.UpdateState()
	ExposeToMbg(e.Id)
}

func ExposeToMbg(serviceId string) {
	exposeResp, err := state.GetEventManager().RaiseExposeRequestEvent(eventManager.ExposeRequestAttr{Service: serviceId})
	if err != nil {
		mlog.Errorf("Unable to raise expose request event")
		return
	}
	mlog.Infof("Response = %+v", exposeResp)
	if exposeResp.Action == eventManager.Deny {
		mlog.Errorf("Denying Expose of service %s", serviceId)
		return
	}

	myIp := state.GetMyIp()
	svcExp := state.GetLocalService(serviceId)
	svcExp.Ip = myIp
	if exposeResp.Action == eventManager.AllowAll {
		for _, mbgId := range state.GetMbgList() {
			ExposeReq(svcExp, mbgId, "MBG")
		}
		return
	}
	for _, mbgId := range exposeResp.TargetMbgs {
		ExposeReq(svcExp, mbgId, "MBG")
	}
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
	resp, err := httpAux.HttpPost(address, j, state.GetHttpClient())
	mlog.Infof("Service(%s) Expose Response message:  %s", svcExp.Id, string(resp))
	if string(resp) != httpAux.RESPFAIL {
		state.AddPeerLocalService(svcExp.Id, mbgId)
	}
}

func CreateLocalServiceEndpoint(serviceId string, port int, namespace, mbgAppName string) error {
	sPort := state.GetConnectionArr()[serviceId].Local

	targetPort, err := strconv.Atoi(sPort[1:])
	if err != nil {
		return err
	}
	mlog.Infof("Creating service end point at %s:%d:%d", serviceId, port, targetPort)
	return kubernetes.Data.CreateServiceEndpoint(serviceId, port, targetPort, namespace, mbgAppName)
}

func DeleteLocalServiceEndpoint(serviceId string) error {
	mlog.Infof("Deleting service end point at %s", serviceId)
	return kubernetes.Data.DeleteServiceEndpoint(serviceId)
}
