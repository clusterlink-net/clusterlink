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

package util

import (
	"fmt"
	"os"
	"strings"

	"github.com/clusterlink-net/clusterlink/pkg/bootstrap/platform"
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
		ControlplaneCertificate: p.controlplaneCert,
		DataplaneCertificate:    p.dataplaneCert,
		GWCTLCertificate:        p.gwctlCert,
		Dataplanes:              cfg.Dataplanes,
		DataplaneType:           cfg.DataplaneType,
		LogLevel:                logLevel,
		ContainerRegistry:       "",
	})

	if err != nil {
		return "", err
	}

	k8sYAML := string(k8sYAMLBytes)

	k8sYAML, err = switchDataplaneServiceToNodeport(k8sYAML)
	if err != nil {
		return "", fmt.Errorf("cannot switch dataplane type to nodeport: %w", err)
	}

	k8sYAML, err = switchNamespace(k8sYAML, f.namespace)
	if err != nil {
		return "", fmt.Errorf("cannot switch namespace: %w", err)
	}

	k8sYAML, err = switchClusterRoleName(k8sYAML, f.namespace)
	if err != nil {
		return "", fmt.Errorf("cannot switch ClusterRole name: %w", err)
	}

	k8sYAML, err = switchClusterRoleBindingName(k8sYAML, f.namespace)
	if err != nil {
		return "", fmt.Errorf("cannot switch ClusterRoleBinding name: %w", err)
	}

	k8sYAML, err = removeGWCTLPod(k8sYAML)
	if err != nil {
		return "", fmt.Errorf("cannot remove gwctl pod: %w", err)
	}

	k8sYAML, err = removeGWCTLSecret(k8sYAML)
	if err != nil {
		return "", fmt.Errorf("cannot remove gwctl secret: %w", err)
	}

	k8sYAML, err = removePeerSecret(k8sYAML)
	if err != nil {
		return "", fmt.Errorf("cannot remove peer secret: %w", err)
	}

	if !cfg.ControlplanePersistency {
		k8sYAML, err = removeControlplanePVC(k8sYAML)
		if err != nil {
			return "", fmt.Errorf("cannot remove controlplane PVC: %w", err)
		}
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

func switchNamespace(yaml, namespace string) (string, error) {
	return replaceOnce(yaml, "namespace: default", "namespace: "+namespace)
}

func switchClusterRoleName(yaml, name string) (string, error) {
	var err error
	search := `
kind: ClusterRole
metadata:
  name: cl-controlplane`
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
  name: cl-controlplane`
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
  name: cl-controlplane`
	replace := `
kind: ClusterRoleBinding
metadata:
  name: %s`
	replace = fmt.Sprintf(replace, name)
	return replaceOnce(yaml, search, replace)
}

func removeGWCTLPod(yaml string) (string, error) {
	search := `
---
apiVersion: v1
kind: Pod
metadata:
  name: gwctl`
	return remove(yaml, search, "\n---")
}

func removeGWCTLSecret(yaml string) (string, error) {
	search := `
---
apiVersion: v1
kind: Secret
metadata:
  name: gwctl`
	return remove(yaml, search, "\n---")
}

func removePeerSecret(yaml string) (string, error) {
	search := `
---
apiVersion: v1
kind: Secret
metadata:
  name: cl-peer`
	return remove(yaml, search, "\n---")
}

func removeControlplanePVC(yaml string) (string, error) {
	var err error
	search := `
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: cl-controlplane
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 100Mi
---`
	yaml, err = replaceOnce(yaml, search, "")
	if err != nil {
		return "", err
	}

	search = `
        - name: cl-controlplane
          persistentVolumeClaim:
            claimName: cl-controlplane`
	yaml, err = replaceOnce(yaml, search, "")
	if err != nil {
		return "", err
	}

	search = `
            - name: cl-controlplane
              mountPath: /var/lib/clink`
	return replaceOnce(yaml, search, "")
}

func changeDataplaneDebugLevel(yaml, logLevel string) (string, error) {
	search := `dataplane
          args: ["--log-level", "debug"`
	replace := `dataplane
          args: ["--log-level", "` + logLevel + `"`
	return replaceOnce(yaml, search, replace)
}
