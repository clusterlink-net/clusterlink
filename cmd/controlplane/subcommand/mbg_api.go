package subcommand

import (
	"time"

	log "github.com/sirupsen/logrus"
	cp "github.ibm.com/mbg-agent/pkg/controlplane"
	event "github.ibm.com/mbg-agent/pkg/controlplane/eventManager"
	"github.ibm.com/mbg-agent/pkg/controlplane/health"
	"github.ibm.com/mbg-agent/pkg/controlplane/store"
	"github.ibm.com/mbg-agent/pkg/k8s/kubernetes"
	"github.ibm.com/mbg-agent/pkg/policyEngine"

	"github.ibm.com/mbg-agent/pkg/utils/logutils"
	"github.ibm.com/mbg-agent/pkg/utils/netutils"
)

const (
	LogFileName = "gw.log"
)

type Mbg struct {
	Id string
}

func (m *Mbg) AddPolicyEngine(policyEngineTarget string, start bool, zeroTrust bool) {
	store.GetEventManager().AssignPolicyDispatcher(store.GetAddrStart()+policyEngineTarget+"/policy", store.GetHttpClient())
	// TODO : Handle different MBG IDs
	store.SaveState()
	defaultRule := event.AllowAll
	if zeroTrust {
		defaultRule = event.Deny
	}
	if start {
		policyEngine.StartPolicyDispatcher(store.GetChiRouter(), defaultRule)
	}
}

func (m *Mbg) StartMbg() {
	go health.SendHeartBeats()
	err := kubernetes.InitializeKubeDeployment("")
	if err != nil {
		log.Errorf("Failed to initialize kube deployment: %+v", err)
	}
	health.MonitorHeartBeats()
}

func CreateMbg(id, ip, cportLocal, cportExtern, localDataPortRange, externalDataPortRange, dataplane,
	caFile, certificateFile, keyFile, logLevel string, logFile, restore bool) (Mbg, error) {

	logutils.SetLog(logLevel, logFile, LogFileName)
	store.SetState(id, ip, cportLocal, cportExtern, localDataPortRange, externalDataPortRange, caFile, certificateFile, keyFile, dataplane)

	//Set chi router
	r := store.GetChiRouter()
	r.Mount("/", cp.MbgHandler{}.Routes())

	if dataplane == "mtls" {
		go netutils.StartMTLSServer(":"+cportLocal, caFile, certificateFile, keyFile, r)
	} else {
		go netutils.StartHTTPServer(":"+cportLocal, r)
	}

	return Mbg{id}, nil
}

func RestoreMbg(id string, policyEngineTarget, logLevel string, logFile, startPolicyEngine bool, zeroTrust bool) (Mbg, error) {

	store.UpdateState()
	logutils.SetLog(logLevel, logFile, LogFileName)
	m := Mbg{store.GetMyId()}
	if startPolicyEngine {
		go m.AddPolicyEngine("localhost"+store.GetMyCport().Local, true, zeroTrust)
	}

	//Set chi router
	r := store.GetChiRouter()
	r.Mount("/", cp.MbgHandler{}.Routes())
	if store.GetDataplane() == "mtls" {
		go netutils.StartMTLSServer(store.GetMyCport().Local, store.GetMyInfo().CaFile, store.GetMyInfo().CertificateFile, store.GetMyInfo().KeyFile, r)
	} else {
		go netutils.StartHTTPServer(store.GetMyCport().Local, r)
	}

	time.Sleep(health.Interval)
	store.RestoreMbg()
	cp.RestoreRemoteServices()

	return Mbg{store.GetMyId()}, nil
}
