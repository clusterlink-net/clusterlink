package k8s

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	dataplaneAppName = "cl-dataplane"
)

// Deployment represents a k8s deployment.
type Deployment struct {
	clientset *kubernetes.Clientset
	namespace string
}

// CreateService creates a service.
func (d *Deployment) CreateService(name string, port, targetPort uint16) error {
	serviceSpec := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Protocol:   v1.ProtocolTCP,
					Port:       int32(port),
					TargetPort: intstr.FromInt(int(targetPort)),
				},
			},
			Type:     v1.ServiceTypeClusterIP,
			Selector: map[string]string{"app": dataplaneAppName},
		},
	}

	_, err := d.clientset.CoreV1().Services(d.namespace).Create(
		context.TODO(), serviceSpec, metav1.CreateOptions{})
	return err
}

// UpdateService updates a service.
func (d *Deployment) UpdateService(name string, port, targetPort uint16) error {
	serviceSpec := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Protocol:   v1.ProtocolTCP,
					Port:       int32(port),
					TargetPort: intstr.FromInt(int(targetPort)),
				},
			},
			Type:     v1.ServiceTypeClusterIP,
			Selector: map[string]string{"app": dataplaneAppName},
		},
	}

	_, err := d.clientset.CoreV1().Services(d.namespace).Update(
		context.TODO(), serviceSpec, metav1.UpdateOptions{})
	return err
}

// DeleteService deletes a service.
func (d *Deployment) DeleteService(name string) error {
	return d.clientset.CoreV1().Services(d.namespace).Delete(
		context.TODO(), name, metav1.DeleteOptions{})
}

// NewDeployment returns a new Kubernetes deployment.
func NewDeployment() (*Deployment, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// TODO: support non-default namespace
	return &Deployment{
		clientset: clientset,
		namespace: v1.NamespaceDefault,
	}, nil
}
