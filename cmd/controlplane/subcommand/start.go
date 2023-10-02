package subcommand

import (
	"fmt"
	"net/http"
	_ "net/http/pprof" //nolint:gosec // G198: Subprocess launched by package local calls only
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	cp "github.com/clusterlink-org/clusterlink/pkg/controlplane"
	"github.com/clusterlink-org/clusterlink/pkg/controlplane/health"
	"github.com/clusterlink-org/clusterlink/pkg/controlplane/store"
	"github.com/clusterlink-org/clusterlink/pkg/k8s/kubernetes"
	metrics "github.com/clusterlink-org/clusterlink/pkg/metrics"
	"github.com/clusterlink-org/clusterlink/pkg/policyengine"
	"github.com/clusterlink-org/clusterlink/pkg/utils/logutils"
	"github.com/clusterlink-org/clusterlink/pkg/utils/netutils"
)

const (
	logFileName = "gw.log"
)

// StartCmd represents the start command of control plane
func StartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "A start command set all parameter state of the Multi-cloud Border Gateway",
		Long: `A start command set all parameter state of the MBg-
			The  id, IP cport(Cntrol port for grpc) and localDataPortRange,externalDataPortRange
			TBD now is done manually need to call some external `,
		Run: func(cmd *cobra.Command, args []string) {
			ip, _ := cmd.Flags().GetString("ip")
			id, _ := cmd.Flags().GetString("id")
			cportLocal, _ := cmd.Flags().GetString("cportLocal")
			cport, _ := cmd.Flags().GetString("cport")
			localDataPortRange, _ := cmd.Flags().GetString("localDataPortRange")
			externalDataPortRange, _ := cmd.Flags().GetString("externalDataPortRange")
			caFile, _ := cmd.Flags().GetString("certca")
			certificateFile, _ := cmd.Flags().GetString("cert")
			keyFile, _ := cmd.Flags().GetString("key")
			dataplane, _ := cmd.Flags().GetString("dataplane")
			startPolicyEngine, _ := cmd.Flags().GetBool("startPolicyEngine")
			observe, _ := cmd.Flags().GetBool("observe")
			policyengineTarget, _ := cmd.Flags().GetString("policyengineIp")
			restore, _ := cmd.Flags().GetBool("restore")
			logFile, _ := cmd.Flags().GetBool("logFile")
			logLevel, _ := cmd.Flags().GetString("logLevel")
			rtenv, _ := cmd.Flags().GetString("rtenv")
			profilePort, _ := cmd.Flags().GetInt("profilePort")

			if ip == "" || id == "" || cport == "" {
				fmt.Println("Error: please insert all flag arguments for Mbg start command")
				os.Exit(1)
			}

			if profilePort != 0 {
				go func() {
					log.Info("Starting PProf HTTP listener at ", profilePort)
					server := &http.Server{
						Addr:              fmt.Sprintf("localhost:%d", profilePort),
						ReadHeaderTimeout: 3 * time.Second,
					}
					log.WithError(server.ListenAndServe()).Error("PProf HTTP listener stopped working")
				}()
			}
			if restore {
				if !startPolicyEngine && policyengineTarget == "" {
					fmt.Println("Error: Please specify policyengineTarget")
					os.Exit(1)
				}
				restoreMbg(logLevel, logFile, startPolicyEngine)
				log.Infof("Restoring MBG")
				store.PrintState()
				initializeRuntimeEnv(rtenv)
				startHealthMonitor()
				return
			}

			createMbg(id, ip, cportLocal, cport, localDataPortRange, externalDataPortRange, dataplane,
				caFile, certificateFile, keyFile, logLevel, logFile)
			if startPolicyEngine {
				addPolicyEngine("localhost:"+cportLocal+"/policy", true)
			}
			if observe {
				addMetricsManager("localhost:"+cportLocal+"/metrics", true)
			}
			store.PrintState()
			initializeRuntimeEnv(rtenv)
			startHealthMonitor()
		},
	}
	addStartFlags(cmd)
	return cmd
}

func addStartFlags(cmd *cobra.Command) {
	cmd.Flags().String("id", "", "Multi-cloud Border Gateway id")
	cmd.Flags().String("ip", "", "Multi-cloud Border Gateway ip")
	cmd.Flags().String("cportLocal", "443", "Multi-cloud Border Gateway control local port inside the MBG")
	cmd.Flags().String("cport", "443", "Multi-cloud Border Gateway control external port for the MBG neighbors ")
	cmd.Flags().String("localDataPortRange", "5000", "Set the port range for data connection in the MBG")
	cmd.Flags().String("externalDataPortRange", "30000", "Set the port range for exposing data connection (each expose port connect to localDataPort")
	cmd.Flags().String("certca", "", "Path to the Root Certificate Auth File (.pem)")
	cmd.Flags().String("cert", "", "Path to the Certificate File (.pem)")
	cmd.Flags().String("key", "", "Path to the Key File (.pem)")
	cmd.Flags().String("dataplane", "mtls", "tcp/mtls based data-plane proxies")
	cmd.Flags().Bool("startPolicyEngine", true, "Start policy engine in port")
	cmd.Flags().Bool("observe", true, "Start metrics manager in port")
	cmd.Flags().String("policyengineIp", "", "Set the policy engine ip")
	cmd.Flags().Bool("restore", false, "Restore existing stored MBG states")
	cmd.Flags().Bool("logFile", true, "Save the outputs to file")
	cmd.Flags().String("logLevel", "info", "Log level: debug, info, warning, error")
	cmd.Flags().String("rtenv", "k8s", "Runtime environment of the gateway: k8s, vm")
}

// startKubeInformer start kube informer for k8s cluster
func initializeRuntimeEnv(rtenv string) {
	cp.MyRunTimeEnv.SetRuntimeEnv(rtenv)

	if cp.MyRunTimeEnv.IsRuntimeEnvK8s() {
		err := kubernetes.InitializeKubeDeployment("")
		if err != nil {
			log.Errorf("Failed to initialize kube deployment: %+v", err)
		}
	}
}

// startHealthMonitor starts health monitor bit
func startHealthMonitor() {
	go func() {
		if err := health.SendHeartBeats(); err != nil {
			log.Errorf("unable to start sending heartbeats: %+v", err)
		}
	}()

	health.MonitorHeartBeats()
}

// addPolicyEngine add policy engine server
func addPolicyEngine(policyengineTarget string, start bool) {
	store.GetEventManager().AssignPolicyDispatcher(store.GetAddrStart()+policyengineTarget, store.GetHTTPClient())
	// TODO : Handle different MBG IDs
	store.SaveState()
	if start {
		policyengine.StartPolicyDispatcher(store.GetChiRouter())
	}
}

// createMbg create mbg control plane process
func createMbg(id, ip, cportLocal, cportExtern, localDataPortRange, externalDataPortRange, dataplane,
	caFile, certificateFile, keyFile, logLevel string, logFile bool) {

	logutils.SetLog(logLevel, logFile, logFileName)
	store.SetState(id, ip, cportLocal, cportExtern, localDataPortRange, externalDataPortRange, caFile, certificateFile, keyFile, dataplane)

	// Set chi router
	r := store.GetChiRouter()
	r.Mount("/", cp.MbgHandler{}.Routes())

	if dataplane == "mtls" {
		go netutils.StartMTLSServer(":"+cportLocal, caFile, certificateFile, keyFile, r)
	} else {
		go netutils.StartHTTPServer(":"+cportLocal, r)
	}
}

// restoreMbg restore the mbg after a failure in the control plane
func restoreMbg(logLevel string, logFile, startPolicyEngine bool) {
	store.UpdateState()
	logutils.SetLog(logLevel, logFile, logFileName)
	if startPolicyEngine {
		go addPolicyEngine("localhost"+store.GetMyCport().Local, true)
	}

	// Set chi router
	r := store.GetChiRouter()
	r.Mount("/", cp.MbgHandler{}.Routes())
	if store.GetDataplane() == "mtls" {
		go netutils.StartMTLSServer(store.GetMyCport().Local, store.GetMyInfo().CaFile, store.GetMyInfo().CertificateFile, store.GetMyInfo().KeyFile, r)
	} else {
		go netutils.StartHTTPServer(store.GetMyCport().Local, r)
	}

	time.Sleep(1 * time.Second)
	store.RestoreMbg()
	cp.RestoreImportServices()
}

func addMetricsManager(metricsManagerTarget string, start bool) {
	store.GetEventManager().AssignMetricsManager(store.GetAddrStart()+metricsManagerTarget, store.GetHTTPClient())
	store.SaveState()
	if start {
		metrics.StartMetricsManager(store.GetChiRouter())
	}
}
