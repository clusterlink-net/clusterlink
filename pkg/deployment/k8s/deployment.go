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

// CreateEndpoint creates a K8s endpoint.
func (d *Deployment) CreateEndpoint(name, targetIP string, targetPort uint16) error {
	endpointSpec := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: d.namespace,
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: targetIP, // Replace with the desired IP address of the endpoint.
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Port: int32(targetPort),
					},
				},
			},
		},
	}

	err := d.client.Create(context.Background(), endpointSpec)
	if err != nil {
		d.logger.Errorf("error occurred while creating K8s endpoint %v:", err)
		return err
	}

	d.logger.Infof("Creating K8s endPoint at %s:%d that connected to external IP: %s:%d.", name, targetPort, targetIP, targetPort)
	return nil
}

// UpdateEndpoint creates a K8s endpoint.
func (d *Deployment) UpdateEndpoint(name, targetIP string, targetPort uint16) error {
	endpointSpec := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: d.namespace,
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: targetIP, // Replace with the desired IP address of the endpoint.
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Port: int32(targetPort),
					},
				},
			},
		},
	}

	err := d.client.Update(context.Background(), endpointSpec)
	if err != nil {
		d.logger.Errorf("error occurred while updating K8s endpoint %v:", err)
		return err
	}

	d.logger.Infof("Updating K8s endPoint at %s:%d that connected to external IP: %s:%d.", name, targetPort, targetIP, targetPort)
	return nil
}

// DeleteEndpoint deletes a k8s endpoint.
func (d *Deployment) DeleteEndpoint(name string) error {
	endpointSpec := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: d.namespace}}

	err := d.client.Delete(context.Background(), endpointSpec)
	if err != nil {
		d.logger.Errorf("error occurred while deleting K8s endpoint %v:", err)
		return err
	}

	d.logger.Infof("Deleting K8s endPoint %s.", name)
	return nil
}

// GetPodLabelsByIP returns all the labels that match the pod IP.
func (d *Deployment) GetPodLabelsByIP(podIP string) (map[string]string, error) {
	var podList corev1.PodList
	err := d.client.List(context.Background(), &podList) // Adjust the label selector as needed
	if err != nil {
		d.logger.Errorf("GetPodLabelsByIP %v.", err.Error())
		return nil, err
	}

	for _, pod := range podList.Items {
		if pod.Status.PodIP == podIP {
			// Found the pod matching the IP address, so get its labels.
			return pod.Labels, nil
		}
	}

	return nil, fmt.Errorf("pod with IP %s not found.", podIP)
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
		return &Deployment{}, fmt.Errorf("pod not found.")
	}
	clNameSpace := podList.Items[0].Namespace
	return &Deployment{
		client:    cl,
		namespace: clNameSpace,
		logger:    logger,
	}, nil
}
