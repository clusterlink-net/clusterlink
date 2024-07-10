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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StatusConditionType represents the status conditions type for ClusterLink components.
type StatusConditionType string

const (
	// DeploymentReady means the component deployment is ready to use.
	DeploymentReady StatusConditionType = "DeploymentReady"
	// ServiceReady means the component service is ready to use.
	ServiceReady StatusConditionType = "ServiceReady"
)

// IngressType represents the ingress type of the deployed ClusterLink.
type IngressType string

const (
	// IngressTypeNone indicates that the deployment instance did not create any external ingress.
	IngressTypeNone IngressType = "none"
	// IngressTypeNodePort indicates that the deployment instance created an external ingress of type NodePort.
	IngressTypeNodePort IngressType = "NodePort"
	// IngressTypeLoadBalancer indicates that the deployment instance created an external ingress of type LoadBalancer.
	IngressTypeLoadBalancer IngressType = "LoadBalancer"
)

// DataplaneType represents the dataplane type of the deployed ClusterLink.
type DataplaneType string

const (
	// DataplaneTypeGo indicates that the dataplane type is Go-dataplane.
	DataplaneTypeGo DataplaneType = "go"
	// DataplaneTypeEnvoy indicates that the dataplane type is Envoy.
	DataplaneTypeEnvoy DataplaneType = "envoy"
)

const (
	// DefaultExternalPort represents the default value for the external ingress service.
	DefaultExternalPort = 443
)

// ComponentStatus defines the status of a component in ClusterLink.
type ComponentStatus struct {
	// Conditions contain the status conditions.
	Conditions map[string]metav1.Condition `json:"conditions,omitempty"`
}

// IngressStatus defines the status of ingress in ClusterLink.
type IngressStatus struct {
	// IP represents the external ingress service's IP.
	IP string `json:"ip,omitempty"`
	// Port represents the external ingress service's Port.
	Port int32 `json:"port,omitempty"`
	// Conditions contain the status conditions.
	Conditions map[string]metav1.Condition `json:"conditions,omitempty"`
}

// InstanceStatus defines the observed state of a ClusterlLink Instance.
type InstanceStatus struct {
	Controlplane ComponentStatus `json:"controlplane,omitempty"`
	Dataplane    ComponentStatus `json:"dataplane,omitempty"`
	Ingress      IngressStatus   `json:"ingress,omitempty"`
}

// DataPlaneSpec defines the desired state of the dataplane components in ClusterLink.
type DataPlaneSpec struct {
	// +kubebuilder:validation:Enum=envoy;go
	// +kubebuilder:default=envoy
	// Type represents the dataplane type. Supports values "go" and "envoy".
	Type DataplaneType `json:"type,omitempty"`
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	// +kubebuilder:default=1
	// Replicas represents the number of dataplane replicas.
	Replicas int `json:"replicas,omitempty"`
}

// IngressSpec defines the type of the ingress component in ClusterLink.
type IngressSpec struct {
	// +kubebuilder:validation:Enum=none;LoadBalancer;NodePort
	// +kubebuilder:default=none
	// Type represents the type of service used to expose the ClusterLink deployment.
	// Supported values: "LoadBalancer","NodePort", "none".
	// The service name will be "clusterlink".
	Type IngressType `json:"type,omitempty"`

	// Port represents the port number of the external service.
	// If not set, the default values will be 443 for all types,
	// except for NodePort, where the port number will be allocated by Kubernetes.
	Port int32 `json:"port,omitempty"`

	// Annotations represents the annotations that will add to ingress service.
	Annotations map[string]string `json:"annotations,omitempty"`
}

// InstanceSpec defines the desired state of a ClusterLink instance.
type InstanceSpec struct {
	DataPlane DataPlaneSpec `json:"dataplane,omitempty"`
	Ingress   IngressSpec   `json:"ingress,omitempty"`

	// +kubebuilder:validation:Enum=trace;debug;info;warning;error;fatal
	// +kubebuilder:default=info
	// LogLevel define the ClusterLink components log level.
	LogLevel string `json:"logLevel,omitempty"`
	// ContainerRegistry is the container registry to pull the ClusterLink project images.
	ContainerRegistry string `json:"containerRegistry,omitempty"`
	// +kubebuilder:default="latest"
	// Tag represents the tag of the ClusterLink project images.
	Tag string `json:"tag,omitempty"`
	// +kubebuilder:default="clusterlink-system"
	// Namespace represents the namespace where the ClusterLink project components are deployed.
	Namespace string `json:"namespace,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Instance is the Schema for the ClusterLink instance API used for deploying a ClusterLink instance.
type Instance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstanceSpec   `json:"spec,omitempty"`
	Status InstanceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// InstanceList represents a list of Instance objects.
type InstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Instance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Instance{}, &InstanceList{})
}
