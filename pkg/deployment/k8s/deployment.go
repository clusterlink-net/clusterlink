package k8s

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	dataplaneAppName = "cl-dataplane"
)

// Deployment represents a k8s deployment.
type Deployment struct {
	client    client.Client
	namespace string
	logger    *logrus.Entry
}

// CreateService creates a service.
func (d *Deployment) CreateService(name string, port, targetPort uint16) error {
	serviceSpec := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: d.namespace},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       int32(port),
					TargetPort: intstr.FromInt(int(targetPort)),
				},
			},
			Type:     corev1.ServiceTypeClusterIP,
			Selector: map[string]string{"app": dataplaneAppName},
		},
	}

	err := d.client.Create(context.Background(), serviceSpec)
	if err != nil {
		d.logger.Errorf("error occurred while creating K8s service %v:", err)
		return err
	}

	d.logger.Infof("Creating K8s service at %s:%d.", name, port)
	return nil
}

// DeleteService deletes a service.
func (d *Deployment) DeleteService(name string) error {
	serviceSpec := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: d.namespace}}

	err := d.client.Delete(context.Background(), serviceSpec)
	if err != nil {
		d.logger.Errorf("error occurred while deleting K8s service %v:", err)
		return err
	}

	d.logger.Infof("Deleting K8s service %s.", name)
	return nil
}

// UpdateService updates a service.
func (d *Deployment) UpdateService(name string, port, targetPort uint16) error {
	serviceSpec := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: d.namespace},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       int32(port),
					TargetPort: intstr.FromInt(int(targetPort)),
				},
			},
			Type:     corev1.ServiceTypeClusterIP,
			Selector: map[string]string{"app": dataplaneAppName},
		},
	}

	err := d.client.Update(context.Background(), serviceSpec)
	if err != nil {
		d.logger.Errorf("error occurred while updating K8s service %v:", err)
		return err
	}

	d.logger.Infof("Updating K8s service at %s:%d.", name, port)
	return nil
}

// DeleteService deletes a service.
func (d *Deployment) DeleteService(name string) error {
	return d.clientset.CoreV1().Services(d.namespace).Delete(
		context.TODO(), name, metav1.DeleteOptions{})
}

// NewDeployment returns a new Kubernetes deployment.
func NewDeployment() (*Deployment, error) {
	logger := logrus.WithField("component", "k8s-deployment")
	cl, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		return &Deployment{}, err
	}

	// Get namespace
	var podList corev1.PodList
	labelSelector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{"app": "cl-controlplane"}})
	if err != nil {
		return &Deployment{}, err
	}

	listOptions := &client.ListOptions{LabelSelector: labelSelector}
	err = cl.List(context.Background(), &podList, listOptions)
	if err != nil {
		return &Deployment{}, err
	}

	if len(podList.Items) == 0 {
		return &Deployment{}, fmt.Errorf("pod not found")
	}
	clNameSpace := podList.Items[0].Namespace
	return &Deployment{
		client:    cl,
		namespace: clNameSpace,
		logger:    logger,
	}, nil
}
