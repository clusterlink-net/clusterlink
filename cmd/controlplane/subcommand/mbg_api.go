package cmd

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	cp "github.ibm.com/mbg-agent/pkg/controlplane"
	handler "github.ibm.com/mbg-agent/pkg/controlplane/api"
	event "github.ibm.com/mbg-agent/pkg/controlplane/eventManager"
	"github.ibm.com/mbg-agent/pkg/controlplane/healthMonitor"
	"github.ibm.com/mbg-agent/pkg/controlplane/store"
	"github.ibm.com/mbg-agent/pkg/k8s/kubernetes"
	"github.ibm.com/mbg-agent/pkg/policyEngine"
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
	go healthMonitor.SendHeartBeats()
	err := kubernetes.InitializeKubeDeployment("")
	if err != nil {
		log.Errorf("Failed to initialize kube deployment: %+v", err)
	}
	healthMonitor.MonitorHeartBeats()
}

func startHttpServer(ip string) {
	//Set chi router
	r := store.GetChiRouter()
	r.Mount("/", handler.MbgHandler{}.Routes())

	//Use router to start the server
	log.Fatal(http.ListenAndServe(ip, r))
}

func startMtlsServer(ip, rootCA, certificate, key string) {
	// Create the TLS Config with the CA pool and enable Client certificate validation
	caCert, err := ioutil.ReadFile(rootCA)
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		ClientCAs:  caCertPool,
		ClientAuth: tls.RequireAndVerifyClientCert,
	}
	//Set chi router
	r := store.GetChiRouter()
	r.Mount("/", handler.MbgHandler{}.Routes())

	// Create a Server instance to listen on port 8443 with the TLS config
	server := &http.Server{
		Addr:      ip,
		TLSConfig: tlsConfig,
		Handler:   r,
	}
	// Listen to HTTPS connections with the server certificate and wait
	log.Fatal(server.ListenAndServeTLS(certificate, key))
}

func Close() {

}

func initLogger(logLevel string, op *os.File) {
	ll, err := log.ParseLevel(logLevel)
	if err != nil {
		ll = log.ErrorLevel
	}
	log.SetLevel(ll)
	log.SetOutput(op)
	log.SetFormatter(
		&log.TextFormatter{
			DisableColors:   false,
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			PadLevelText:    true,
			DisableQuote:    true,
		})
}

func CreateMbg(id, ip, cportLocal, cportExtern, localDataPortRange, externalDataPortRange, dataplane,
	caFile, certificateFile, keyFile, logLevel string, logFile, restore bool) (Mbg, error) {

	store.SetLog(logLevel, logFile)
	store.SetState(id, ip, cportLocal, cportExtern, localDataPortRange, externalDataPortRange, caFile, certificateFile, keyFile, dataplane)

	if dataplane == "mtls" {
		go startMtlsServer(":"+cportLocal, caFile, certificateFile, keyFile)
	} else {
		go startHttpServer(":" + cportLocal)
	}

	return Mbg{id}, nil
}

func RestoreMbg(id string, policyEngineTarget, logLevel string, logFile, startPolicyEngine bool, zeroTrust bool) (Mbg, error) {

	store.UpdateState()
	store.SetLog(logLevel, logFile)
	m := Mbg{store.GetMyId()}
	if startPolicyEngine {
		go m.AddPolicyEngine("localhost"+store.GetMyCport().Local, true, zeroTrust)
	}

	if store.GetDataplane() == "mtls" {
		go startMtlsServer(store.GetMyCport().Local, store.GetMyInfo().CaFile, store.GetMyInfo().CertificateFile, store.GetMyInfo().KeyFile)
	} else {
		go startHttpServer(store.GetMyCport().Local)
	}

	time.Sleep(healthMonitor.Interval)
	store.RestoreMbg()
	cp.RestoreRemoteServices()

	return Mbg{store.GetMyId()}, nil
}
