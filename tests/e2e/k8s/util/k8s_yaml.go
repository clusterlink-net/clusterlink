// Copyright (c) The ClusterLink Authors.
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

package util

import (
	"fmt"
	"os"
	"strings"

	"github.com/clusterlink-net/clusterlink/pkg/bootstrap/platform"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
)

const (
	ClusterNameLabel = "cluster"
	PeerIPLabel      = "ip"
)

// replaceOnce replaces <search> exactly once.
func replaceOnce(s, search, replace string) (string, error) {
	searchCount := strings.Count(s, search)
	if searchCount != 1 {
		return "", fmt.Errorf("found %d (!=1) occurrences of '%s'", searchCount, search)
	}

	return strings.ReplaceAll(s, search, replace), nil
}

// remove removes a substring starting with <from> until <to> (excluding).
func remove(s, from, to string) (string, error) {
	searchCount := strings.Count(s, from)
	if searchCount != 1 {
		return "", fmt.Errorf("found %d (!=1) occurrences of '%s'", searchCount, from)
	}

	startPos := strings.Index(s, from)
	tmpPos := strings.Index(s[startPos+len(from):], to)
	if tmpPos == -1 {
		return "", fmt.Errorf("cannot found termination for '%s'", from)
	}
	endPos := startPos + len(from) + tmpPos

	return s[:startPos] + s[endPos:], nil
}

func (f *Fabric) generateK8SYAML(p *peer, cfg *PeerConfig) (string, error) {
	logLevel := "info"
	if os.Getenv("DEBUG") == "1" {
		logLevel = "debug"
	}

	k8sYAMLBytes, err := platform.K8SConfig(&platform.Config{
		Peer:                    p.cluster.Name(),
		FabricCertificate:       f.cert,
		PeerCertificate:         p.peerCert,
		CACertificate:           p.caCert,
		Controlplanes:           cfg.Controlplanes,
		ControlplaneCertificate: p.controlplaneCert,
		DataplaneCertificate:    p.dataplaneCert,
		Dataplanes:              cfg.Dataplanes,
		DataplaneType:           cfg.DataplaneType,
		LogLevel:                logLevel,
		ContainerRegistry:       "",
		Namespace:               f.namespace,
		Tag:                     "latest",
		PeerLabels: map[string]string{
			"cluster": p.cluster.name,
			"ip":      p.cluster.ip,
		},
	})
	if err != nil {
		return "", err
	}

	k8sYAML := string(k8sYAMLBytes)

	k8sYAML, err = switchDataplaneServiceToNodeport(k8sYAML)
	if err != nil {
		return "", fmt.Errorf("cannot switch dataplane type to nodeport: %w", err)
	}

	k8sYAML, err = switchClusterRoleName(k8sYAML, f.namespace)
	if err != nil {
		return "", fmt.Errorf("cannot switch ClusterRole name: %w", err)
	}

	k8sYAML, err = switchClusterRoleBindingName(k8sYAML, f.namespace)
	if err != nil {
		return "", fmt.Errorf("cannot switch ClusterRoleBinding name: %w", err)
	}

	if (os.Getenv("DEBUG")) == "1" {
		dpLogLevel := "trace" // More informative than the debug level.
		if cfg.ExpectLargeDataplaneTraffic && os.Getenv("CICD") == "1" {
			dpLogLevel = "info"
		}

		k8sYAML, err = changeDataplaneDebugLevel(k8sYAML, dpLogLevel)
		if err != nil {
			return "", fmt.Errorf("cannot set dataplane debug level: %w", err)
		}
	}

	return k8sYAML, nil
}

func switchDataplaneServiceToNodeport(yaml string) (string, error) {
	search := `
  ports:
    - name: dataplane`
	replace := `
  type: NodePort
  ports:
    - name: dataplane`
	return replaceOnce(yaml, search, replace)
}

func switchClusterRoleName(yaml, name string) (string, error) {
	var err error
	search := `
kind: ClusterRole
metadata:
  name: ` + api.Name
	replace := `
kind: ClusterRole
metadata:
  name: %s`
	replace = fmt.Sprintf(replace, name)
	yaml, err = replaceOnce(yaml, search, replace)
	if err != nil {
		return "", err
	}

	search = `
  kind: ClusterRole
  name: ` + api.Name
	replace = `
  kind: ClusterRole
  name: %s`
	replace = fmt.Sprintf(replace, name)
	return replaceOnce(yaml, search, replace)
}

func switchClusterRoleBindingName(yaml, name string) (string, error) {
	search := `
kind: ClusterRoleBinding
metadata:
  name: ` + api.Name
	replace := `
kind: ClusterRoleBinding
metadata:
  name: %s`
	replace = fmt.Sprintf(replace, name)
	return replaceOnce(yaml, search, replace)
}

func changeDataplaneDebugLevel(yaml, logLevel string) (string, error) {
	search := `args: ["--log-level", "debug", "--controlplane-host"`
	replace := `args: ["--log-level", "` + logLevel + `", "--controlplane-host"`
	return replaceOnce(yaml, search, replace)
}

// generateClusterlinkSecrets ClusterLink secretes.
func (f *Fabric) generateClusterlinkSecrets(p *peer) (string, error) {
	certConfig, err := platform.K8SCertificateConfig(&platform.Config{
		Peer:                    p.cluster.Name(),
		FabricCertificate:       f.cert,
		PeerCertificate:         p.peerCert,
		CACertificate:           p.caCert,
		ControlplaneCertificate: p.controlplaneCert,
		DataplaneCertificate:    p.dataplaneCert,
		Namespace:               f.namespace,
	})
	if err != nil {
		return "", err
	}
	return string(certConfig), nil
}

// generateClusterlinkInstance generates ClusterLink instance yaml.
func (f *Fabric) generateClusterlinkInstance(name string, p *peer, cfg *PeerConfig) (string, error) {
	logLevel := "info"
	if os.Getenv("DEBUG") == "1" {
		logLevel = "debug"
	}

	instance, err := platform.K8SClusterLinkInstanceConfig(&platform.Config{
		Peer:              p.cluster.Name(),
		Controlplanes:     cfg.Controlplanes,
		Dataplanes:        cfg.Dataplanes,
		DataplaneType:     cfg.DataplaneType,
		LogLevel:          logLevel,
		ContainerRegistry: "docker.io/library", // Tell kind to use local image.
		Tag:               "latest",
		Namespace:         f.namespace,
		IngressType:       "NodePort",
	}, name)
	if err != nil {
		return "", err
	}

	return string(instance), nil
}
