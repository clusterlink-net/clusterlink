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

package platform

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"text/template"

	cpapp "github.com/clusterlink-net/clusterlink/cmd/cl-controlplane/app"
	dpapp "github.com/clusterlink-net/clusterlink/cmd/cl-dataplane/app"
	apis "github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	cpapi "github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
	dpapi "github.com/clusterlink-net/clusterlink/pkg/dataplane/api"
	corev1 "k8s.io/api/core/v1"
)

const (
	nsTemplate = `---
  apiVersion: v1
  kind: Namespace
  metadata:
    name: {{.namespace}}
`
	certsTemplate = `---
apiVersion: v1
kind: Secret
metadata:
  name: cl-ca
  namespace: {{.namespace}}
data:
  ca: {{.ca}}
---
apiVersion: v1
kind: Secret
metadata:
  name: {{.controlplaneName}}
  namespace: {{.namespace}}
data:
  cert: {{.controlplaneCert}}
  key: {{.controlplaneKey}}
---
apiVersion: v1
kind: Secret
metadata:
  name: {{.dataplaneName}}
  namespace: {{.namespace}}
data:
  cert: {{.dataplaneCert}}
  key: {{.dataplaneKey}}
---
apiVersion: v1
kind: Secret
metadata:
  name: cl-peer
  namespace: {{.namespace}}
data:
  {{.peerCertificateFile}}: {{.peerCert}}
  {{.peerKeyFile}}: {{.peerKey}}
  {{.fabricCertFile}}: {{.fabricCert}}
`
	k8sTemplate = `---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.controlplaneName}}
  namespace: {{.namespace}}
  labels:
    app: {{.controlplaneName}}
spec:
  replicas: {{.controlplanes}}
  selector:
    matchLabels:
      app: {{.controlplaneName}}
  template:
    metadata:
      labels:
        app: {{.controlplaneName}}
    spec:
      volumes:
        - name: ca
          secret:
            secretName: cl-ca
        - name: tls
          secret:
            secretName: {{.controlplaneName}}
        - name: peer-tls
          secret:
            secretName: cl-peer
      containers:
        - name: {{.controlplaneName}}
          image: {{.containerRegistry}}{{.controlplaneName}}:{{.tag}}
          args: ["--log-level", "{{.logLevel}}"]
          imagePullPolicy: IfNotPresent
          readinessProbe:
            httpGet:
              port:  {{.controlplaneReadinessPort}}
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
            - name: peer-tls
              mountPath: {{.peerTLSMountPath}}
              readOnly: true
          env:
            - name: {{ .namespaceEnvVariable }}
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.dataplaneName}}
  namespace: {{.namespace}}
  labels:
    app: {{ .dataplaneName }}
spec:
  replicas: {{.dataplanes}}
  selector:
    matchLabels:
      app: {{ .dataplaneName }}
  template:
    metadata:
      labels:
        app: {{ .dataplaneName }}
    spec:
      volumes:
        - name: ca
          secret:
            secretName: cl-ca
        - name: tls
          secret:
            secretName: {{.dataplaneName}}
      containers:
        - name: dataplane
          image: {{.containerRegistry}}{{
          if (eq .dataplaneType .dataplaneTypeEnvoy) }}{{.dataplaneName}}{{
          else }}{{.goDataplaneImageName}}{{ end }}:{{.tag}}
          args: ["--log-level", "{{.logLevel}}", "--controlplane-host", "{{.controlplaneName}}"]
          imagePullPolicy: IfNotPresent
          ports:
            - containerPort: {{.dataplanePort}}
          readinessProbe:
            httpGet:
              port: {{.dataplaneReadinessPort}}
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
kind: Service
metadata:
  name: {{.controlplaneName}}
  namespace: {{.namespace}}
spec:
  selector:
    app: {{.controlplaneName}}
  ports:
    - name: controlplane
      port: {{.controlplanePort}}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: {{.controlplaneName}}
rules:
- apiGroups: [""]
  resources: ["events"]
  verbs: ["create", "update"]
- apiGroups: [""]
  resources: ["services"]
  verbs: ["get", "list", "watch", "create", "delete", "update"]
- apiGroups: ["coordination.k8s.io"]
  resources: ["leases"]
  verbs: ["get", "create", "update"]
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "list", "watch", "create", "update"]
- apiGroups: ["discovery.k8s.io"]
  resources: ["endpointslices"]
  verbs: ["get", "list", "watch", "create", "delete", "update"]
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["clusterlink.net"]
  resources: ["exports", "peers", "accesspolicies", "privilegedaccesspolicies"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["clusterlink.net"]
  resources: ["imports"]
  verbs: ["get", "list", "watch", "update"]
- apiGroups: ["clusterlink.net"]
  resources: ["imports/status", "exports/status", "peers/status"]
  verbs: ["update"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: {{.controlplaneName}}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: {{.controlplaneName}}
subjects:
- kind: ServiceAccount
  name: default
  namespace: {{.namespace}}
`
	ClusterLinkInstanceTemplate = `apiVersion: clusterlink.net/v1alpha1
kind: Instance
metadata:
  labels:
    app.kubernetes.io/name: instance
    app.kubernetes.io/instance: {{.name}}
    app.kubernetes.io/part-of: clusterlink
    app.kubernetes.io/created-by: clusterlink
  name: {{.name}}
  namespace: clusterlink-operator
spec:
  dataplane:
    type: {{.dataplaneType}}
    replicas: {{.dataplanes}}
  ingress:
    type: {{.ingressType}}
{{ if .ingressPort }}
    port: {{.ingressPort }}
{{ end }}
    annotations: {{.ingressAnnotations}}
  logLevel: {{.logLevel}}
  containerRegistry: {{.containerRegistry}}
  namespace: {{.namespace}}
  tag: {{.tag}}
`
	ingressTemplate = `---
apiVersion: v1
kind: Service
metadata:
  name: {{.dataplaneService}}
  namespace: {{.namespace}}
spec:
  type: {{.ingressType}}
  ports:
    - name: dataplane
      port: {{.ingressPort }}
      targetPort: {{.dataplanePort}}
{{ if .ingressNodePort }}
      nodePort: {{.ingressNodePort }}
{{ end }}
  selector:
    app:  {{.dataplaneName}}
  `
)

// K8SConfig returns a kubernetes deployment file.
func K8SConfig(config *Config) ([]byte, error) {
	containerRegistry := config.ContainerRegistry
	if containerRegistry != "" {
		containerRegistry = config.ContainerRegistry + "/"
	}

	controlplanes := config.Controlplanes
	if controlplanes == 0 {
		controlplanes = 1
	}

	dataplanes := config.Dataplanes
	if dataplanes == 0 {
		dataplanes = 1
	}

	dataplaneType := config.DataplaneType
	if dataplaneType == "" {
		dataplaneType = DataplaneTypeEnvoy
	}

	args := map[string]interface{}{
		"peer":              config.Peer,
		"namespace":         config.Namespace,
		"controlplanes":     controlplanes,
		"dataplanes":        dataplanes,
		"dataplaneType":     dataplaneType,
		"logLevel":          config.LogLevel,
		"containerRegistry": containerRegistry,
		"tag":               config.Tag,

		"dataplaneTypeEnvoy":   DataplaneTypeEnvoy,
		"namespaceEnvVariable": cpapp.NamespaceEnvVariable,

		"controlplaneName":     cpapi.Name,
		"dataplaneName":        dpapi.Name,
		"goDataplaneImageName": dpapi.GoDataplaneName,

		"controlplaneCAMountPath":   cpapp.CAFile,
		"controlplaneCertMountPath": cpapp.CertificateFile,
		"controlplaneKeyMountPath":  cpapp.KeyFile,

		"peerTLSMountPath": cpapp.PeerTLSDirectory,

		"dataplaneCAMountPath":   dpapp.CAFile,
		"dataplaneCertMountPath": dpapp.CertificateFile,
		"dataplaneKeyMountPath":  dpapp.KeyFile,

		"controlplanePort":          cpapi.ListenPort,
		"controlplaneReadinessPort": cpapi.ReadinessListenPort,
		"dataplanePort":             dpapi.ListenPort,
		"dataplaneReadinessPort":    dpapi.ReadinessListenPort,
	}

	var k8sConfig, nsConfig bytes.Buffer
	// ClusterLink namespace
	t := template.Must(template.New("").Parse(nsTemplate))
	if err := t.Execute(&nsConfig, args); err != nil {
		return nil, fmt.Errorf("cannot create K8s namespace from template: %w", err)
	}

	// ClusterLink components
	t = template.Must(template.New("").Parse(k8sTemplate))
	if err := t.Execute(&k8sConfig, args); err != nil {
		return nil, fmt.Errorf("cannot create k8s configuration from template: %w", err)
	}

	// ClusterLink certificates
	certConfig, err := K8SCertificateConfig(config)
	if err != nil {
		return nil, err
	}

	// ClusterLink ingress service
	ingressConfig, err := k8SIngressConfig(config)
	if err != nil {
		return nil, err
	}

	k8sBytes := nsConfig.Bytes()
	k8sBytes = append(k8sBytes, certConfig...)
	k8sBytes = append(k8sBytes, k8sConfig.Bytes()...)
	k8sBytes = append(k8sBytes, ingressConfig...)

	return k8sBytes, nil
}

// K8SCertificateConfig returns a kubernetes secrets that contains all the certificates.
func K8SCertificateConfig(config *Config) ([]byte, error) {
	args := map[string]interface{}{
		"ca":                  base64.StdEncoding.EncodeToString(config.CACertificate.RawCert()),
		"controlplaneCert":    base64.StdEncoding.EncodeToString(config.ControlplaneCertificate.RawCert()),
		"controlplaneKey":     base64.StdEncoding.EncodeToString(config.ControlplaneCertificate.RawKey()),
		"dataplaneCert":       base64.StdEncoding.EncodeToString(config.DataplaneCertificate.RawCert()),
		"dataplaneKey":        base64.StdEncoding.EncodeToString(config.DataplaneCertificate.RawKey()),
		"controlplaneName":    cpapi.Name,
		"dataplaneName":       dpapi.Name,
		"peerCertificateFile": cpapp.PeerCertificateFile,
		"peerKeyFile":         cpapp.PeerKeyFile,
		"fabricCertFile":      cpapp.FabricCertificateFile,
		"peerCert":            base64.StdEncoding.EncodeToString(config.PeerCertificate.RawCert()),
		"peerKey":             base64.StdEncoding.EncodeToString(config.PeerCertificate.RawKey()),
		"fabricCert":          base64.StdEncoding.EncodeToString(config.FabricCertificate.RawCert()),
		"namespace":           config.Namespace,
	}

	var certConfig bytes.Buffer
	t := template.Must(template.New("").Parse(certsTemplate))
	if err := t.Execute(&certConfig, args); err != nil {
		return nil, fmt.Errorf("cannot create k8s certificate configuration from template: %w", err)
	}

	return certConfig.Bytes(), nil
}

// K8SClusterLinkInstanceConfig returns a YAML file for the ClusterLink instance.
func K8SClusterLinkInstanceConfig(config *Config, name string) ([]byte, error) {
	containerRegistry := config.ContainerRegistry
	if containerRegistry != "" {
		containerRegistry = config.ContainerRegistry + "/"
	}

	// Convert ingress annotations map to string.
	ingressAnnotationsStr := "\n"
	for key, value := range config.IngressAnnotations {
		ingressAnnotationsStr += fmt.Sprintf("      %s: %s\n", key, value)
	}

	args := map[string]interface{}{
		"name":               name,
		"controlplanes":      config.Controlplanes,
		"dataplanes":         config.Dataplanes,
		"dataplaneType":      config.DataplaneType,
		"logLevel":           config.LogLevel,
		"containerRegistry":  containerRegistry,
		"namespace":          config.Namespace,
		"ingressType":        config.IngressType,
		"ingressAnnotations": ingressAnnotationsStr,
		"tag":                config.Tag,
	}

	if config.IngressPort != 0 {
		if config.IngressType == string(apis.IngressTypeNodePort) && (config.IngressPort < 30000) || (config.IngressPort > 32767) {
			return nil, fmt.Errorf("nodeport number %v is not in the valid range (30000:32767)", config.IngressPort)
		}
		args["ingressPort"] = config.IngressPort
	}

	var clConfig bytes.Buffer
	t := template.Must(template.New("").Parse(ClusterLinkInstanceTemplate))
	if err := t.Execute(&clConfig, args); err != nil {
		return nil, fmt.Errorf("cannot create clusterlink instance configuration from template: %w", err)
	}

	return clConfig.Bytes(), nil
}

// K8SEmptyCertificateConfig returns Kubernetes empty secrets for the control plane and data plane,
// used for deleting the secrets.
func K8SEmptyCertificateConfig(config *Config) ([]byte, error) {
	args := map[string]interface{}{
		"Namespace":        config.Namespace,
		"controlplaneName": cpapi.Name,
		"dataplaneName":    dpapi.Name,
		"ca":               "",
		"controlplaneCert": "",
		"controlplaneKey":  "",
		"dataplaneCert":    "",
		"dataplaneKey":     "",
		"peerCert":         "",
		"peerKey":          "",
		"fabricCert":       "",
	}

	var certConfig bytes.Buffer
	t := template.Must(template.New("").Parse(certsTemplate))
	if err := t.Execute(&certConfig, args); err != nil {
		return nil, fmt.Errorf("cannot create k8s certificate configuration from template: %w", err)
	}

	return certConfig.Bytes(), nil
}

// k8SIngressConfig returns a kubernetes ingress service.
func k8SIngressConfig(config *Config) ([]byte, error) {
	var ingressConfig bytes.Buffer

	ingressType := string(corev1.ServiceTypeClusterIP)
	if config.IngressType == string(apis.IngressTypeNodePort) || config.IngressType == string(apis.IngressTypeLoadBalancer) {
		ingressType = config.IngressType
	}

	args := map[string]interface{}{
		"namespace":   config.Namespace,
		"ingressPort": apis.DefaultExternalPort,
		"ingressType": ingressType,

		"dataplaneService": dpapp.IngressSvcName,
		"dataplaneName":    dpapi.Name,
		"dataplanePort":    dpapi.ListenPort,
	}

	if config.IngressPort != 0 {
		if config.IngressType == string(apis.IngressTypeNodePort) {
			args["ingressNodePort"] = config.IngressPort
			if (config.IngressPort < 30000) || (config.IngressPort > 32767) {
				return nil, fmt.Errorf("nodeport number %v is not in the valid range (30000:32767)", config.IngressPort)
			}
		} else {
			args["ingressPort"] = config.IngressPort
		}
	}

	t := template.Must(template.New("").Parse(ingressTemplate))
	if err := t.Execute(&ingressConfig, args); err != nil {
		return nil, fmt.Errorf("cannot create K8s namespace from template: %w", err)
	}

	return ingressConfig.Bytes(), nil
}
