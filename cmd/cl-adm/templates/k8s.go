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
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/clusterlink-net/clusterlink/cmd/cl-adm/config"
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
          imagePullPolicy: IfNotPresent
          args: ["--log-level", "info", "--platform", "k8s"]
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
          imagePullPolicy: IfNotPresent
          args: ["--controlplane-host", "cl-controlplane", "--log-level", "info"]
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
        secretName: cl-fabric
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

// CreateK8SConfig creates a kubernetes deployment file.
func CreateK8SConfig(args map[string]interface{}, outDir string) error {
	var k8sConfig bytes.Buffer
	t := template.Must(template.New("").Parse(k8sTemplate))
	if err := t.Execute(&k8sConfig, args); err != nil {
		return fmt.Errorf("cannot create k8s configuration off template: %v", err)
	}

	outPath := filepath.Join(outDir, config.K8SYamlFile)
	return os.WriteFile(outPath, k8sConfig.Bytes(), 0600)
}
