// Copyright 2023 The ClusterLink Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof" //nolint:gosec // G108:  Profiling endpoint is automatically exposed on /debug/pprof
	"time"

	log "github.com/sirupsen/logrus"

	dp "github.com/clusterlink-net/clusterlink/pkg/dataplane"
	"github.com/clusterlink-net/clusterlink/pkg/dataplane/store"
	logutils "github.com/clusterlink-net/clusterlink/pkg/util/log"
)

const (
	logFileName = "/.gw/dataplane.log"
)

func main() {
	var id, port, ca, cert, key, logLevel, dataplane, controlplane string
	var profilePort int
	// Initialize the variable with the flag
	flag.StringVar(&id, "id", "", "Data plane gateway id")
	flag.StringVar(&port, "port", "443", "Default port data-plane start to listen")
	flag.StringVar(&ca, "certca", "", "Path to the Root Certificate Auth File (.pem)")
	flag.StringVar(&cert, "cert", "", "Path to the Certificate File (.pem)")
	flag.StringVar(&key, "key", "", "Path to the Key File (.pem)")
	flag.StringVar(&dataplane, "dataplane", "mtls", "tcp/mtls based data-plane proxies")
	flag.StringVar(&controlplane, "controlplane", "controlplane:443", "Target(ip:port) of the controlplane")
	flag.StringVar(&logLevel, "logLevel", "info", "Log level: debug, info, warning, error")
	flag.IntVar(&profilePort, "profilePort", 0, "Port to enable profiling")
	// Parse command-line flags
	flag.Parse()
	// Set log file
	f, err := logutils.SetLog(logLevel, logFileName)
	if err != nil {
		log.Error(err)
	}

	if f != nil {
		defer func() {
			if err := f.Close(); err != nil {
				log.Errorf("Cannot close log file: %v", err)
			}
		}()
	}

	log.Infof("Dataplane main started")

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

	// Set Dataplane
	dp := dp.NewDataplane(&store.Store{ID: id, CertAuthority: ca, Cert: cert, Key: key, Dataplane: dataplane}, controlplane)
	dp.StartServer(port)
	log.Infof("Dataplane main process is finished")
}
