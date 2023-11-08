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

package platform

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"text/template"

	cpapp "github.com/clusterlink-net/clusterlink/cmd/cl-controlplane/app"
	dpapp "github.com/clusterlink-net/clusterlink/cmd/cl-dataplane/app"
	cpapi "github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
	dpapi "github.com/clusterlink-net/clusterlink/pkg/dataplane/api"
)

const (
	k8sTemplate = `---
apiVersion: v1
kind: Secret
metadata:
  name: cl-fabric
data:
  ca: {{.fabricCA}}
---
apiVersion: v1
kind: Secret
metadata:
  name: cl-peer
data:
  ca: {{.peerCA}}
---
apiVersion: v1
kind: Secret
metadata:
  name: cl-controlplane
data:
  cert: {{.controlplaneCert}}
  key: {{.controlplaneKey}}
---
apiVersion: v1
kind: Secret
metadata:
  name: cl-dataplane
data:
  cert: {{.dataplaneCert}}
  key: {{.dataplaneKey}}
---
apiVersion: v1
kind: Secret
metadata:
  name: gwctl
data:
  cert: {{.gwctlCert}}
  key: {{.gwctlKey}}
---
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
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cl-controlplane
  labels:
    app: cl-controlplane
spec:
  replicas: 1
  selector:
    matchLabels:
      app: cl-controlplane
  template:
    metadata:
      labels:
        app: cl-controlplane
    spec:
      volumes:
        - name: ca
          secret:
            secretName: cl-fabric
        - name: tls
          secret:
            secretName: cl-controlplane
        - name: cl-controlplane
          persistentVolumeClaim:
            claimName: cl-controlplane
      containers:
        - name: cl-controlplane
          image: cl-controlplane
          args: ["--log-level", "{{.logLevel}}", "--platform", "k8s"]
          imagePullPolicy: IfNotPresent
          ports:
            - containerPort: {{.controlplanePort}}
          volumeMounts:
            - name: ca
              mountPath: {{.controlplaneCAMountPath}}
              subPath: "ca"
              readOnly: true
            - name: tls
              mountPath: {{.controlplaneCertMountPath}}
              subPath: "cert"
              readOnly: true
            - name: tls
              mountPath: {{.controlplaneKeyMountPath}}
              subPath: "key"
              readOnly: true
            - name: cl-controlplane
              mountPath: {{.persistencyDirectoryMountPath}}
          env:
            - name: CL-NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cl-dataplane
  labels:
    app: cl-dataplane
spec:
  replicas: {{.dataplanes}}
  selector:
    matchLabels:
      app: cl-dataplane
  template:
    metadata:
      labels:
        app: cl-dataplane
    spec:
      volumes:
        - name: ca
          secret:
            secretName: cl-fabric
        - name: tls
          secret:
            secretName: cl-dataplane
      containers:
        - name: dataplane
          {{ if (eq .dataplaneType .dataplaneTypeEnvoy) }}image: cl-dataplane{{ else }}image: cl-go-dataplane{{ end }}
          args: ["--log-level", "{{.logLevel}}", "--controlplane-host", "cl-controlplane"]
          imagePullPolicy: IfNotPresent
          ports:
            - containerPort: {{.dataplanePort}}
          volumeMounts:
            - name: ca
              mountPath: {{.dataplaneCAMountPath}}
              subPath: "ca"
              readOnly: true
            - name: tls
              mountPath: {{.dataplaneCertMountPath}}
              subPath: "cert"
              readOnly: true
            - name: tls
              mountPath: {{.dataplaneKeyMountPath}}
              subPath: "key"
              readOnly: true
---
apiVersion: v1
kind: Pod
metadata:
  name: gwctl
  labels:
    app: gwctl
spec:
  volumes:
    - name: ca
      secret:
        secretName: cl-peer
    - name: tls
      secret:
        secretName: gwctl
  containers:
    - name: gwctl
      image: gwctl
      imagePullPolicy: IfNotPresent
      command: ["/bin/sh"]
      args:
        - -c
        - >-
            gwctl init --id {{.peer}} \
                       --gwIP cl-dataplane \
                       --gwPort {{.dataplanePort}} \
                       --certca /root/ca.pem \
                       --cert /root/cert.pem \
                       --key /root/key.pem &&
            gwctl config use-context --myid {{.peer}} &&
            sleep infinity
      volumeMounts:
        - name: ca
          mountPath: /root/ca.pem
          subPath: "ca"
          readOnly: true
        - name: tls
          mountPath: /root/cert.pem
          subPath: "cert"
          readOnly: true
        - name: tls
          mountPath: /root/key.pem
          subPath: "key"
          readOnly: true
---
apiVersion: v1
kind: Service
metadata:
  name: cl-controlplane
spec:
  selector:
    app: cl-controlplane
  ports:
    - name: controlplane
      port: {{.controlplanePort}}
---
apiVersion: v1
kind: Service
metadata:
  name: cl-dataplane
spec:
  selector:
    app: cl-dataplane
  ports:
    - name: dataplane
      port: {{.dataplanePort}}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: cl-controlplane
rules:
- apiGroups: [""]
  resources: ["services", "endpoints"]
  verbs: ["create", "delete"]
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: cl-controlplane
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cl-controlplane
subjects:
- kind: ServiceAccount
  name: default
  namespace: default`
)

// K8SConfig returns a kubernetes deployment file.
func K8SConfig(config *Config) ([]byte, error) {
	args := map[string]interface{}{
		"peer":          config.Peer,
		"dataplanes":    config.Dataplanes,
		"dataplaneType": config.DataplaneType,
		"logLevel":      config.LogLevel,

		"dataplaneTypeEnvoy": DataplaneTypeEnvoy,

		"fabricCA":         base64.StdEncoding.EncodeToString(config.FabricCertificate.RawCert()),
		"peerCA":           base64.StdEncoding.EncodeToString(config.PeerCertificate.RawCert()),
		"controlplaneCert": base64.StdEncoding.EncodeToString(config.ControlplaneCertificate.RawCert()),
		"controlplaneKey":  base64.StdEncoding.EncodeToString(config.ControlplaneCertificate.RawKey()),
		"dataplaneCert":    base64.StdEncoding.EncodeToString(config.DataplaneCertificate.RawCert()),
		"dataplaneKey":     base64.StdEncoding.EncodeToString(config.DataplaneCertificate.RawKey()),
		"gwctlCert":        base64.StdEncoding.EncodeToString(config.GWCTLCertificate.RawCert()),
		"gwctlKey":         base64.StdEncoding.EncodeToString(config.GWCTLCertificate.RawKey()),

		"persistencyDirectoryMountPath": filepath.Dir(cpapp.StoreFile),

		"controlplaneCAMountPath":   cpapp.CAFile,
		"controlplaneCertMountPath": cpapp.CertificateFile,
		"controlplaneKeyMountPath":  cpapp.KeyFile,

		"dataplaneCAMountPath":   dpapp.CAFile,
		"dataplaneCertMountPath": dpapp.CertificateFile,
		"dataplaneKeyMountPath":  dpapp.KeyFile,

		"controlplanePort": cpapi.ListenPort,
		"dataplanePort":    dpapi.ListenPort,
	}

	var k8sConfig bytes.Buffer
	t := template.Must(template.New("").Parse(k8sTemplate))
	if err := t.Execute(&k8sConfig, args); err != nil {
		return nil, fmt.Errorf("cannot create k8s configuration off template: %w", err)
	}

	return k8sConfig.Bytes(), nil
}
