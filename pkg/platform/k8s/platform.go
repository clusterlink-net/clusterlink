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

// Platform represents a k8s platform.
type Platform struct {
	endpointReconciler *reconciler
	serviceReconciler  *reconciler
	client             client.Client
	namespace          string
	logger             *logrus.Entry
}

// CreateService creates a service.
func (d *Platform) CreateService(name, targetApp string, port, targetPort uint16) {
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
			Selector: map[string]string{"app": targetApp},
		},
	}
	d.logger.Infof("Creating K8s service at %s:%d.", name, port)
	go d.serviceReconciler.CreateResource(serviceSpec)
}

// DeleteService deletes a service.
func (d *Platform) DeleteService(name string) {
	serviceSpec := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: d.namespace}}

	d.logger.Infof("Deleting K8s service %s.", name)
	go d.serviceReconciler.DeleteResource(serviceSpec)
}

// UpdateService updates a service.
func (d *Platform) UpdateService(name, targetApp string, port, targetPort uint16) {
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
			Selector: map[string]string{"app": targetApp},
		},
	}

	d.logger.Infof("Updating K8s service at %s:%d.", name, port)
	go d.serviceReconciler.UpdateResource(serviceSpec)

}

// CreateEndpoint creates a K8s endpoint.
func (d *Platform) CreateEndpoint(name, targetIP string, targetPort uint16) {
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

	d.logger.Infof("Creating K8s endPoint at %s:%d that connected to external IP: %s:%d.", name, targetPort, targetIP, targetPort)
	go d.endpointReconciler.CreateResource(endpointSpec)

}

// UpdateEndpoint creates a K8s endpoint.
func (d *Platform) UpdateEndpoint(name, targetIP string, targetPort uint16) {
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

	d.logger.Infof("Updating K8s endPoint at %s:%d that connected to external IP: %s:%d.", name, targetPort, targetIP, targetPort)
	go d.endpointReconciler.UpdateResource(endpointSpec)

}

// DeleteEndpoint deletes a k8s endpoint.
func (d *Platform) DeleteEndpoint(name string) {
	endpointSpec := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: d.namespace}}

	d.logger.Infof("Deleting K8s endPoint %s.", name)
	go d.endpointReconciler.DeleteResource(endpointSpec)

}

// NewPlatform returns a new Kubernetes platform.
func NewPlatform() (*Platform, error) {
	logger := logrus.WithField("component", "k8s-platform")
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	cl, err := client.New(cfg, client.Options{})
	if err != nil {
		return nil, err
	}

	// Get namespace
	var podList corev1.PodList
	labelSelector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: map[string]string{"app": "cl-controlplane"}})
	if err != nil {
		return &Platform{}, err
	}

	listOptions := &client.ListOptions{LabelSelector: labelSelector}
	err = cl.List(context.Background(), &podList, listOptions)
	if err != nil {
		return &Platform{}, err
	}

	if len(podList.Items) == 0 {
		return &Platform{}, fmt.Errorf("pod not found.")
	}

	clNameSpace := podList.Items[0].Namespace
	return &Platform{
		client:             cl,
		serviceReconciler:  NewReconciler(cl),
		endpointReconciler: NewReconciler(cl),
		namespace:          clNameSpace,
		logger:             logger,
	}, nil
}
