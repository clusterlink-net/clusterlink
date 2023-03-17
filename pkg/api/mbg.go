package api

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/pkg/mbgControlplane"
	"github.ibm.com/mbg-agent/pkg/policyEngine"
	handler "github.ibm.com/mbg-agent/pkg/protocol/http/mbg"
)

type Mbg struct {
	Id string
}

func (m *Mbg) AddPolicyEngine(policyEngineTarget string, start bool) {
	state.GetEventManager().AssignPolicyDispatcher("http://" + policyEngineTarget + "/policy")
	// TODO : Handle different MBG IDs
	state.SaveState()
	serverPolicyIp := policyEngineTarget
	if strings.Contains(policyEngineTarget, "localhost") {
		serverPolicyIp = ":" + strings.Split(policyEngineTarget, ":")[1]
	}
	if start {
		policyEngine.StartPolicyDispatcher(state.GetChiRouter(), serverPolicyIp)
	}
}

func (m *Mbg) StartMbg() {
	go mbgControlplane.SendHeartBeats()
	mbgControlplane.MonitorHeartBeats()
}

func startHttpServer(ip string) {
	//Set chi router
	r := state.GetChiRouter()
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
	r := state.GetChiRouter()
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

	state.SetLog(logLevel, logFile)
	state.SetState(id, ip, cportLocal, cportExtern, localDataPortRange, externalDataPortRange, caFile, certificateFile, keyFile, dataplane)

	if dataplane == "mtls" {
		go startMtlsServer(":"+cportLocal, caFile, certificateFile, keyFile)
	} else {
		go startHttpServer(":" + cportLocal)
	}

	return Mbg{id}, nil
}

func RestoreMbg(id string, policyEngineTarget, logLevel string, logFile, startPolicyEngine bool) (Mbg, error) {

	state.UpdateState()
	state.SetLog(logLevel, logFile)
	m := Mbg{state.GetMyId()}
	if startPolicyEngine {
		go m.AddPolicyEngine(policyEngineTarget, true)
	}

	if state.GetDataplane() == "mtls" {
		go startMtlsServer(state.GetMyCport().Local, state.GetMyInfo().CaFile, state.GetMyInfo().CertificateFile, state.GetMyInfo().KeyFile)
	} else {
		go startHttpServer(state.GetMyCport().Local)
	}

	time.Sleep(mbgControlplane.Interval)
	state.RestoreMbg()
	mbgControlplane.RestoreRemoteServices()

	return Mbg{state.GetMyId()}, nil
}
