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

package templates

import (
	"encoding/base64"
	"os"
	"path/filepath"

	"github.com/clusterlink-net/clusterlink/cmd/cl-adm/config"
	cpapp "github.com/clusterlink-net/clusterlink/cmd/cl-controlplane/app"
	dpapp "github.com/clusterlink-net/clusterlink/cmd/cl-dataplane/app"
	cpapi "github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
	dpapi "github.com/clusterlink-net/clusterlink/pkg/dataplane/api"
)

// Config holds a configuration to instantiate a template.
type Config struct {
	// Peer is the peer name.
	Peer string

	// Dataplanes is the number of dataplane servers to run.
	Dataplanes uint16

	// DataplaneType is the type of dataplane to create (envoy or go-based)
	DataplaneType string
}

const (
	// DataplaneTypeEnvoy represents an envoy-type dataplane.
	DataplaneTypeEnvoy = "envoy"
	// DataplaneTypeGo represents a go-type dataplane.
	DataplaneTypeGo = "go"
)

// TemplateArgs returns arguments for instantiating a text/template
func (c Config) TemplateArgs() (map[string]interface{}, error) {
	fabricCA, err := os.ReadFile(filepath.Join(config.FabricDirectory(), config.CertificateFileName))
	if err != nil {
		return nil, err
	}

	peerCA, err := os.ReadFile(filepath.Join(config.PeerDirectory(c.Peer), config.CertificateFileName))
	if err != nil {
		return nil, err
	}

	controlplaneCert, err := os.ReadFile(filepath.Join(config.ControlplaneDirectory(c.Peer), config.CertificateFileName))
	if err != nil {
		return nil, err
	}

	controlplaneKey, err := os.ReadFile(filepath.Join(config.ControlplaneDirectory(c.Peer), config.PrivateKeyFileName))
	if err != nil {
		return nil, err
	}

	dataplaneCert, err := os.ReadFile(filepath.Join(config.DataplaneDirectory(c.Peer), config.CertificateFileName))
	if err != nil {
		return nil, err
	}

	dataplaneKey, err := os.ReadFile(filepath.Join(config.DataplaneDirectory(c.Peer), config.PrivateKeyFileName))
	if err != nil {
		return nil, err
	}

	gwctlCert, err := os.ReadFile(filepath.Join(config.GWCTLDirectory(c.Peer), config.CertificateFileName))
	if err != nil {
		return nil, err
	}

	gwctlKey, err := os.ReadFile(filepath.Join(config.GWCTLDirectory(c.Peer), config.PrivateKeyFileName))
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"peer":          c.Peer,
		"dataplanes":    c.Dataplanes,
		"dataplaneType": c.DataplaneType,

		"dataplaneTypeEnvoy": DataplaneTypeEnvoy,

		"fabricCA":         base64.StdEncoding.EncodeToString(fabricCA),
		"peerCA":           base64.StdEncoding.EncodeToString(peerCA),
		"controlplaneCert": base64.StdEncoding.EncodeToString(controlplaneCert),
		"controlplaneKey":  base64.StdEncoding.EncodeToString(controlplaneKey),
		"dataplaneCert":    base64.StdEncoding.EncodeToString(dataplaneCert),
		"dataplaneKey":     base64.StdEncoding.EncodeToString(dataplaneKey),
		"gwctlCert":        base64.StdEncoding.EncodeToString(gwctlCert),
		"gwctlKey":         base64.StdEncoding.EncodeToString(gwctlKey),

		"fabricCAPath":         filepath.Join(config.FabricDirectory(), config.CertificateFileName),
		"peerCAPath":           filepath.Join(config.PeerDirectory(c.Peer), config.CertificateFileName),
		"controlplaneCertPath": filepath.Join(config.ControlplaneDirectory(c.Peer), config.CertificateFileName),
		"controlplaneKeyPath":  filepath.Join(config.ControlplaneDirectory(c.Peer), config.PrivateKeyFileName),
		"dataplaneCertPath":    filepath.Join(config.DataplaneDirectory(c.Peer), config.CertificateFileName),
		"dataplaneKeyPath":     filepath.Join(config.DataplaneDirectory(c.Peer), config.PrivateKeyFileName),
		"gwctlCertPath":        filepath.Join(config.GWCTLDirectory(c.Peer), config.CertificateFileName),
		"gwctlKeyPath":         filepath.Join(config.GWCTLDirectory(c.Peer), config.PrivateKeyFileName),

		"controlplanePersistencyDirectory": filepath.Join(config.ControlplaneDirectory(c.Peer), config.PersistencyDirectoryName),
		"dataplanePersistencyDirectory":    filepath.Join(config.DataplaneDirectory(c.Peer), config.PersistencyDirectoryName),

		"persistencyDirectoryMountPath": filepath.Dir(cpapp.StoreFile),

		"controlplaneCAMountPath":   cpapp.CAFile,
		"controlplaneCertMountPath": cpapp.CertificateFile,
		"controlplaneKeyMountPath":  cpapp.KeyFile,

		"dataplaneCAMountPath":   dpapp.CAFile,
		"dataplaneCertMountPath": dpapp.CertificateFile,
		"dataplaneKeyMountPath":  dpapp.KeyFile,

		"controlplanePort": cpapi.ListenPort,
		"dataplanePort":    dpapi.ListenPort,
	}, nil
}
