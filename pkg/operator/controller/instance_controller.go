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

package controller

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	cpapp "github.com/clusterlink-net/clusterlink/cmd/cl-controlplane/app"
	dpapp "github.com/clusterlink-net/clusterlink/cmd/cl-dataplane/app"
	clusterlink "github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	cpapi "github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
	dpapi "github.com/clusterlink-net/clusterlink/pkg/dataplane/api"
	"github.com/sirupsen/logrus"
)

const (
	ControlPlaneName  = "cl-controlplane"
	DataPlaneName     = "cl-dataplane"
	GoDataPlaneName   = "cl-go-dataplane"
	IngressName       = "clusterlink"
	OperatorNamespace = "clusterlink-operator"
	InstanceNamespace = "clusterlink-system"
	FinalizerName     = "instance.clusterlink.net/finalizer"

	StatusModeNotExist    = "NotExist"
	StatusModeProgressing = "ProgressingMode"
	StatusModeReady       = "Ready"
)

// InstanceReconciler reconciles a ClusterLink instance object.
type InstanceReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Logger    *logrus.Entry
	Instances map[string]string
}

// +kubebuilder:rbac:groups=clusterlink.net,resources=instances,verbs=list;get;watch;update;patch
// +kubebuilder:rbac:groups=clusterlink.net,resources=instances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=clusterlink.net,resources=instances/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=list;get;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services;serviceaccounts,verbs=list;get;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=nodes,verbs=list;get;watch
// +kubebuilder:rbac:groups="",resources=pods,verbs=list;get;watch
// +kubebuilder:rbac:groups=clusterlink.net,resources=exports;peers;accesspolicies,verbs=list;get;watch
// +kubebuilder:rbac:groups=clusterlink.net,resources=imports,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=clusterlink.net,resources=peers/status,verbs=update
// +kubebuilder:rbac:groups="apps",resources=deployments,verbs=list;get;watch;create;update;patch;delete
//nolint:lll // Ignore long line warning for Kubebuilder command.
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterroles;clusterrolebindings,verbs=list;get;watch;create;update;patch;delete

// TODO- should review the operator RABCs.

// SetupWithManager sets up the controller with the Manager.
func (r *InstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterlink.Instance{}).
		Watches(
			&appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: ControlPlaneName,
				},
			},
			&handler.EnqueueRequestForObject{},
		).
		Watches(
			&appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name: DataPlaneName,
				},
			},
			&handler.EnqueueRequestForObject{},
		).
		Complete(r)
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// The reconcile get instance YAML and creates the ClusterLink components.
func (r *InstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Fetch the Clusterlink instance
	instance := &clusterlink.Instance{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		// Requeue the request on error
		return ctrl.Result{}, err
	}

	// Check one clusterlink per namespace
	if name, exist := r.Instances[instance.Spec.Namespace]; exist {
		if instance.Name != name {
			r.Logger.Errorf(
				"Can't create instance %s in namespace %s, instance %s already exists and only one instance is permitted in a Namespace",
				instance.Name, instance.Spec.Namespace, name)
			err := r.updateFailureStatus(ctx, instance)
			return ctrl.Result{}, err
		}
	} else {
		r.Instances[instance.Spec.Namespace] = instance.Name
	}
	// Examine DeletionTimestamp to determine if object is under deletion
	if !instance.DeletionTimestamp.IsZero() {
		if err := r.deleteClusterLink(ctx, instance.Spec.Namespace); err != nil {
			return ctrl.Result{}, err
		}

		if err := r.deleteFinalizer(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}

		delete(r.Instances, instance.Spec.Namespace)
		r.Logger.Infof("Delete instance: %s Namespace: %s", instance.Name, instance.Namespace)
		if err := r.triggerAnotherInstance(ctx, instance.Name, instance.Spec.Namespace); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// CRD details
	r.Logger.Infof("Enter instance Reconcile - (Namespace: %s, Name: %s)", instance.Namespace, instance.Name)
	r.Logger.Info("InstanceSpec- ",
		" DataPlane.Type: ", instance.Spec.DataPlane.Type,
		", DataPlane.Replicas: ", instance.Spec.DataPlane.Replicas,
		", Ingress.Type: ", instance.Spec.Ingress.Type,
		", Ingress.Port: ", instance.Spec.Ingress.Port,
		", Namespace: ", instance.Spec.Namespace,
		", LogLevel: ", instance.Spec.LogLevel,
		", ContainerRegistry: ", instance.Spec.ContainerRegistry,
		", ImageTag: ", instance.Spec.ImageTag,
	)

	// Set Finalizer if needed
	if err := r.createFinalizer(ctx, instance); err != nil {
		return ctrl.Result{}, fmt.Errorf("can't create Finalizer %w", err)
	}

	// Check ClusterLink components status
	if err := r.checkStatus(ctx, instance); err != nil {
		return ctrl.Result{}, fmt.Errorf("can't check components status %w", err)
	}

	// Apply ClusterLink components if needed
	if err := r.applyClusterLink(ctx, instance); err != nil {
		return ctrl.Result{}, fmt.Errorf("can't apply clusterlink components %w", err)
	}

	// Wait until status is ready or error
	if instance.Status.Controlplane.Conditions[string(clusterlink.DeploymentReady)].Reason == StatusModeProgressing ||
		instance.Status.Dataplane.Conditions[string(clusterlink.DeploymentReady)].Reason == StatusModeProgressing {
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 3}, err
	}
	return ctrl.Result{}, nil
}

// applyClusterLink sets up all the components for the ClusterLink project.
func (r *InstanceReconciler) applyClusterLink(ctx context.Context, instance *clusterlink.Instance) error {
	if instance.Spec.ContainerRegistry != "" && instance.Spec.ContainerRegistry[len(instance.Spec.ContainerRegistry)-1:] != "/" {
		instance.Spec.ContainerRegistry += "/"
	}
	// Create controlplane components
	if err := r.createPVC(ctx, ControlPlaneName, instance.Spec.Namespace); err != nil {
		return err
	}

	if err := r.createAccessControl(ctx, ControlPlaneName, instance.Spec.Namespace); err != nil {
		return err
	}

	if err := r.createService(ctx, ControlPlaneName, instance.Spec.Namespace, cpapi.ListenPort); err != nil {
		return err
	}

	if err := r.applyControlplane(ctx, instance); err != nil {
		return err
	}

	// Create datapalne components
	if err := r.createService(ctx, DataPlaneName, instance.Spec.Namespace, dpapi.ListenPort); err != nil {
		return err
	}

	if err := r.applyDataplane(ctx, instance); err != nil {
		return err
	}

	// create external ingress service
	return r.createExternalService(ctx, instance)
}

// applyControlplane sets up the controlplane deployment.
func (r *InstanceReconciler) applyControlplane(ctx context.Context, instance *clusterlink.Instance) error {
	cpDeployment := r.setDeployment(ControlPlaneName, instance.Spec.Namespace, 1)
	cpDeployment.Spec.Template.Spec = corev1.PodSpec{
		ServiceAccountName: ControlPlaneName,
		Volumes: []corev1.Volume{
			{
				Name: "ca",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: "cl-fabric",
					},
				},
			},
			{
				Name: "tls",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: ControlPlaneName,
					},
				},
			},
			{
				Name: ControlPlaneName,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: ControlPlaneName,
					},
				},
			},
		},
		Containers: []corev1.Container{
			{
				Name:            ControlPlaneName,
				Image:           instance.Spec.ContainerRegistry + ControlPlaneName + ":" + instance.Spec.ImageTag,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Args:            []string{"--log-level", instance.Spec.LogLevel},
				Ports: []corev1.ContainerPort{
					{
						ContainerPort: cpapi.ListenPort,
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "ca",
						MountPath: cpapp.CAFile,
						SubPath:   "ca",
						ReadOnly:  true,
					},
					{
						Name:      "tls",
						MountPath: cpapp.CertificateFile,
						SubPath:   "cert",
						ReadOnly:  true,
					},
					{
						Name:      "tls",
						MountPath: cpapp.KeyFile,
						SubPath:   "key",
						ReadOnly:  true,
					},
					{
						Name:      ControlPlaneName,
						MountPath: filepath.Dir(cpapp.StoreFile),
					},
				},
				Env: []corev1.EnvVar{
					{
						Name: cpapp.NamespaceEnvVariable,
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "metadata.namespace",
							},
						},
					},
				},
			},
		},
	}
	return r.createOrUpdateResource(ctx, &cpDeployment)
}

// applyDataplane sets up the dataplane deployment.
func (r *InstanceReconciler) applyDataplane(ctx context.Context, instance *clusterlink.Instance) error {
	DataplaneImage := DataPlaneName
	if instance.Spec.DataPlane.Type == clusterlink.DataplaneTypeGo {
		DataplaneImage = GoDataPlaneName
	}

	dpDeployment := r.setDeployment(DataPlaneName, instance.Spec.Namespace, int32(instance.Spec.DataPlane.Replicas))
	dpDeployment.Spec.Template.Spec = corev1.PodSpec{
		Volumes: []corev1.Volume{
			{
				Name: "ca",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: "cl-fabric",
					},
				},
			},
			{
				Name: "tls",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: DataPlaneName,
					},
				},
			},
		},
		Containers: []corev1.Container{
			{
				Name:  "dataplane",
				Image: instance.Spec.ContainerRegistry + DataplaneImage + ":" + instance.Spec.ImageTag,
				Args: []string{
					"--log-level", instance.Spec.LogLevel,
					"--controlplane-host", ControlPlaneName,
				},
				ImagePullPolicy: corev1.PullIfNotPresent,
				Ports: []corev1.ContainerPort{
					{
						ContainerPort: dpapi.ListenPort,
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "ca",
						MountPath: dpapp.CAFile,
						SubPath:   "ca",
						ReadOnly:  true,
					},
					{
						Name:      "tls",
						MountPath: dpapp.CertificateFile,
						SubPath:   "cert",
						ReadOnly:  true,
					},
					{
						Name:      "tls",
						MountPath: dpapp.KeyFile,
						SubPath:   "key",
						ReadOnly:  true,
					},
				},
			},
		},
	}

	return r.createOrUpdateResource(ctx, &dpDeployment)
}

func (r *InstanceReconciler) setDeployment(name, namespace string, replicas int32) appsv1.Deployment {
	return appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(replicas),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": name},
				},
			},
		},
	}
}

// createService sets up a k8s service.
func (r *InstanceReconciler) createService(ctx context.Context, name, namespace string, port uint16) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Protocol: corev1.ProtocolTCP,
					Port:     int32(port),
				},
			},
			Type:     corev1.ServiceTypeClusterIP,
			Selector: map[string]string{"app": name},
		},
	}

	return r.createResource(ctx, service)
}

// createPVC sets up k8s a persistent volume claim for the.
func (r *InstanceReconciler) createPVC(ctx context.Context, name, namespace string) error {
	// Create the PVC for cl-controlplane
	controlplanePVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("100Mi"),
				},
			},
		},
	}

	return r.createResource(ctx, controlplanePVC)
}

// createAccessControl sets up k8s ClusterRule and ClusterRoleBinding for the controlplane.
func (r *InstanceReconciler) createAccessControl(ctx context.Context, name, namespace string) error {
	// Create ServiceAccount object
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	if err := r.createResource(ctx, sa); err != nil {
		return err
	}
	// Create the ClusterRole for the controlplane.
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: name + namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"services"},
				Verbs: []string{
					"get", "list", "watch", "create", "delete", "update",
				},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"clusterlink.net"},
				Resources: []string{"peers", "exports", "accesspolicies"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"clusterlink.net"},
				Resources: []string{"imports"},
				Verbs:     []string{"get", "list", "watch", "update"},
			},
			{
				APIGroups: []string{"clusterlink.net"},
				Resources: []string{"peers/status"},
				Verbs:     []string{"update"},
			},
		},
	}

	if err := r.createResource(ctx, clusterRole); err != nil {
		return err
	}

	// Create ClusterRoleBinding for the controlplane.
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name + namespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     name + namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      ControlPlaneName,
				Namespace: namespace,
			},
		},
	}
	return r.createResource(ctx, clusterRoleBinding)
}

// createExternalService sets up the external service for the project.
func (r *InstanceReconciler) createExternalService(ctx context.Context, instance *clusterlink.Instance) error {
	if instance.Spec.Ingress.Type == clusterlink.IngressTypeNone {
		return nil
	}

	// Create a Service object
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      IngressName,
			Namespace: instance.Spec.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": dpapp.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Port:       clusterlink.ExternalDefaultPort,
					TargetPort: intstr.FromInt(dpapi.ListenPort),
					Name:       "https",
				},
			},
		},
	}

	switch instance.Spec.Ingress.Type {
	case clusterlink.IngressTypeNodePort:
		service.Spec.Type = corev1.ServiceTypeNodePort
		if instance.Spec.Ingress.Port != 0 {
			service.Spec.Ports[0].NodePort = instance.Spec.Ingress.Port
		}
	case clusterlink.IngressTypeLoadBalancer:
		service.Spec.Type = corev1.ServiceTypeLoadBalancer
		if instance.Spec.Ingress.Port != 0 {
			service.Spec.Ports[0].Port = instance.Spec.Ingress.Port
		}
	}

	return r.createResource(ctx, service)
}

// createFinalizer sets up finalizer for the instance CRD.
func (r *InstanceReconciler) createFinalizer(ctx context.Context, instance *clusterlink.Instance) error {
	if !controllerutil.ContainsFinalizer(instance, FinalizerName) {
		controllerutil.AddFinalizer(instance, FinalizerName)
		return r.Update(ctx, instance)
	}

	return nil
}

// deleteFinalizer remove the finalizer from the instance CRD.
func (r *InstanceReconciler) deleteFinalizer(ctx context.Context, instance *clusterlink.Instance) error {
	// remove our finalizer from the list and update it.
	controllerutil.RemoveFinalizer(instance, FinalizerName)
	return r.Update(ctx, instance)
}

func (r *InstanceReconciler) createOrUpdateResource(ctx context.Context, object client.Object) error {
	key := client.ObjectKeyFromObject(object)
	err := r.Client.Create(ctx, object)
	if err == nil {
		r.Logger.Infof("Create resource %s Name: %s Namespace: %s", reflect.TypeOf(object), object.GetName(), object.GetNamespace())
		return nil
	}

	if errors.IsAlreadyExists(err) { // If resource already exists, update it
		err = r.Client.Update(ctx, object)
	}

	if err != nil {
		r.Logger.Errorf("Failed to create/update resource %v %s: %v", reflect.TypeOf(object), key, err)
	}

	return err
}

// createResource uses for creates k8s resource.
func (r *InstanceReconciler) createResource(ctx context.Context, object client.Object) error {
	err := r.Get(ctx, types.NamespacedName{Name: object.GetName(), Namespace: object.GetNamespace()}, object)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}

		r.Logger.Infof("Create resource %s Name: %s Namespace: %s", reflect.TypeOf(object), object.GetName(), object.GetNamespace())
		return r.Create(ctx, object)
	}

	return nil
}

// deleteClusterLink delete all the ClusterLink resource.
func (r *InstanceReconciler) deleteClusterLink(ctx context.Context, namespace string) error {
	// Delete controlPlane Resources
	cpObj := metav1.ObjectMeta{Name: ControlPlaneName, Namespace: namespace}
	if err := r.deleteResource(ctx, &appsv1.Deployment{ObjectMeta: cpObj}); err != nil {
		return err
	}

	if err := r.deleteResource(ctx, &corev1.Service{ObjectMeta: cpObj}); err != nil {
		return err
	}

	if err := r.deleteResource(ctx, &corev1.PersistentVolumeClaim{ObjectMeta: cpObj}); err != nil {
		return err
	}

	if err := r.deleteResource(ctx, &rbacv1.ClusterRole{ObjectMeta: cpObj}); err != nil {
		return err
	}

	if err := r.deleteResource(ctx, &rbacv1.ClusterRoleBinding{ObjectMeta: cpObj}); err != nil {
		return err
	}

	// Delete dataplane Resources
	dpObj := metav1.ObjectMeta{Name: DataPlaneName, Namespace: namespace}
	if err := r.deleteResource(ctx, &appsv1.Deployment{ObjectMeta: dpObj}); err != nil {
		return err
	}

	if err := r.deleteResource(ctx, &corev1.Service{ObjectMeta: dpObj}); err != nil {
		return err
	}

	// Delete external ingress service
	ingerssObj := metav1.ObjectMeta{Name: IngressName, Namespace: namespace}
	return r.deleteResource(ctx, &corev1.Service{ObjectMeta: ingerssObj})
}

// deleteResource delete a k8s resource.
func (r *InstanceReconciler) deleteResource(ctx context.Context, object client.Object) error {
	if err := r.Delete(ctx, object); err != nil && !errors.IsNotFound(err) {
		r.Logger.Error("Delete resource error", err)
		return err
	}
	return nil
}

// Helper function to convert int32 to *int32.
func int32Ptr(i int32) *int32 {
	return &i
}

// checkStatus check the status of ClusterLink components.
func (r *InstanceReconciler) checkStatus(ctx context.Context, instance *clusterlink.Instance) error {
	cpUpdate, err := r.checkControlplaneStatus(ctx, instance)
	if err != nil {
		return err
	}

	dpUpdate, err := r.checkDataplaneStatus(ctx, instance)
	if err != nil {
		return err
	}

	ingressUpdate := false
	if instance.Spec.Ingress.Type != clusterlink.IngressTypeNone {
		ingressUpdate, err = r.checkIngressStatus(ctx, instance)
		if err != nil {
			return err
		}
	}

	if cpUpdate || dpUpdate || ingressUpdate {
		return r.Status().Update(ctx, instance)
	}

	return nil
}

// checkControlplaneStatus check the status of the controlplane components.
func (r *InstanceReconciler) checkControlplaneStatus(ctx context.Context, instance *clusterlink.Instance) (bool, error) {
	cp := types.NamespacedName{Name: ControlPlaneName, Namespace: instance.Spec.Namespace}
	deploymentStatus, err := r.checkDeploymnetStatus(ctx, cp)
	if err != nil {
		return false, err
	}
	_, serviceStatus, err := r.checkServiceStatus(ctx, cp)
	if err != nil {
		return false, err
	}

	if instance.Status.Controlplane.Conditions == nil {
		instance.Status.Controlplane.Conditions = make(map[string]metav1.Condition)
	}

	updateFlag := r.updateCondition(instance.Status.Controlplane.Conditions, []metav1.Condition{deploymentStatus, serviceStatus})

	return updateFlag, nil
}

// checkDataplaneStatus check the status of the dataplane components.
func (r *InstanceReconciler) checkDataplaneStatus(ctx context.Context, instance *clusterlink.Instance) (bool, error) {
	dp := types.NamespacedName{Name: DataPlaneName, Namespace: instance.Spec.Namespace}
	deploymentStatus, err := r.checkDeploymnetStatus(ctx, dp)
	if err != nil {
		return false, err
	}
	_, serviceStatus, err := r.checkServiceStatus(ctx, dp)
	if err != nil {
		return false, err
	}

	if instance.Status.Dataplane.Conditions == nil {
		instance.Status.Dataplane.Conditions = make(map[string]metav1.Condition)
	}

	updateFlag := r.updateCondition(instance.Status.Dataplane.Conditions, []metav1.Condition{deploymentStatus, serviceStatus})
	return updateFlag, nil
}

// checkIngressStatus check the status of the ingress components.
func (r *InstanceReconciler) checkIngressStatus(ctx context.Context, instance *clusterlink.Instance) (bool, error) {
	ingress := types.NamespacedName{Name: IngressName, Namespace: instance.Spec.Namespace}
	serviceStatus, err := r.checkExternalServiceStatus(ctx, ingress, &instance.Status.Ingress)
	if err != nil {
		return false, err
	}

	if instance.Status.Ingress.Conditions == nil {
		instance.Status.Ingress.Conditions = make(map[string]metav1.Condition)
	}

	updateFlag := r.updateCondition(instance.Status.Ingress.Conditions, []metav1.Condition{serviceStatus})
	return updateFlag, nil
}

// checkDeploymnetStatus check the status of a deployment.
func (r *InstanceReconciler) checkDeploymnetStatus(ctx context.Context, name types.NamespacedName) (metav1.Condition, error) {
	d := &appsv1.Deployment{}
	status := metav1.Condition{
		Type:               string(clusterlink.DeploymentReady),
		Status:             metav1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
	}

	if err := r.Get(ctx, name, d); err != nil {
		if errors.IsNotFound(err) {
			status.Reason = "NotExist"
			status.Message = "Deployment does not exist"
			return status, nil
		}
		return metav1.Condition{}, err
	}

	status.Reason = StatusModeProgressing
	status.Message = "Deployment is in progressing mode"

	// Check the conditions in the updated Deployment status.
	conditions := d.Status.Conditions
	for _, condition := range conditions {
		switch condition.Type {
		case appsv1.DeploymentAvailable:
			if condition.Status == corev1.ConditionTrue {
				status.Status = metav1.ConditionTrue
				status.Reason = StatusModeReady
				status.Message = "Deployment is ready"
				return status, nil
			}
		case appsv1.DeploymentProgressing, appsv1.DeploymentReplicaFailure:
			if condition.Status != corev1.ConditionTrue {
				status.Reason = condition.Reason
				status.Message = condition.Message
				return status, nil
			}
		}
	}

	return status, nil
}

// checkExternlaServiceStatus check the status of a external service.
//
//nolint:lll // Ignore line length on function names
func (r *InstanceReconciler) checkExternalServiceStatus(ctx context.Context, name types.NamespacedName, ingressStatus *clusterlink.IngressStatus) (metav1.Condition, error) {
	s, status, err := r.checkServiceStatus(ctx, name)
	if err != nil {
		return status, err
	}

	if status.Status == metav1.ConditionTrue {
		switch s.Spec.Type {
		case corev1.ServiceTypeLoadBalancer:
			ingressStatus.Port = s.Spec.Ports[0].Port
			if len(s.Status.LoadBalancer.Ingress) > 0 {
				ingressStatus.IP = s.Status.LoadBalancer.Ingress[0].IP
			} else {
				ingressStatus.IP = "pending"
			}
		case corev1.ServiceTypeNodePort:
			ingressStatus.Port = s.Spec.Ports[0].NodePort
			ip, err := r.getNodeIP(ctx)
			if err == nil {
				ingressStatus.IP = ip
			} else {
				r.Logger.Error("fail to get nodeport IP:", err)
			}
		}
	}
	return status, nil
}

// checkServiceStatus check the status of a service.
//
//nolint:lll // Ignore line length on function names
func (r *InstanceReconciler) checkServiceStatus(ctx context.Context, name types.NamespacedName) (corev1.Service, metav1.Condition, error) {
	s := corev1.Service{}
	status := metav1.Condition{
		Type:               string(clusterlink.ServiceReady),
		Status:             metav1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
	}

	if err := r.Get(ctx, name, &s); err != nil {
		if errors.IsNotFound(err) {
			status.Reason = StatusModeNotExist
			status.Message = "Service does not exist"
			return s, status, nil
		}
		return s, metav1.Condition{}, err
	}

	status.Status = metav1.ConditionTrue
	status.Reason = StatusModeReady
	status.Message = "Service is ready"
	return s, status, nil
}

// updateFailureStatuse updates the component status conditions to failure status.
func (r *InstanceReconciler) updateFailureStatus(ctx context.Context, instance *clusterlink.Instance) error {
	r.Logger.Info("updateFailureStatuse insert function")
	cond := []metav1.Condition{{
		Type:               string(clusterlink.DeploymentReady),
		Status:             metav1.ConditionFalse,
		Reason:             StatusModeNotExist,
		LastTransitionTime: metav1.Now(),
	}}

	if instance.Status.Controlplane.Conditions == nil {
		instance.Status.Controlplane.Conditions = make(map[string]metav1.Condition)
	}

	if instance.Status.Dataplane.Conditions == nil {
		instance.Status.Dataplane.Conditions = make(map[string]metav1.Condition)
	}

	cpUpdate := r.updateCondition(instance.Status.Controlplane.Conditions, cond)
	dpUpdate := r.updateCondition(instance.Status.Dataplane.Conditions, cond)

	if cpUpdate || dpUpdate {
		r.Logger.Info("updateFailureStatuse update")
		return r.Status().Update(ctx, instance)
	}
	return nil
}

// updateCondition updates the component status conditions.
func (r *InstanceReconciler) updateCondition(conditions map[string]metav1.Condition, newConditions []metav1.Condition) bool {
	update := false
	for _, newCondition := range newConditions {
		if c, ok := conditions[newCondition.Type]; ok { // Check if the condition already exists based on type
			if c.Status != newCondition.Status || c.Message != newCondition.Message {
				conditions[newCondition.Type] = newCondition
				update = true
			}
		} else {
			// Condition not exist
			conditions[newCondition.Type] = newCondition
			update = true
		}
	}

	return update
}

// triggerAnotherInstance checks if another CRD instance exists and triggers it.
func (r *InstanceReconciler) triggerAnotherInstance(ctx context.Context, name, namespace string) error {
	var earliestInstance *clusterlink.Instance
	instanceList := &clusterlink.InstanceList{}
	if err := r.List(ctx, instanceList, client.InNamespace(OperatorNamespace)); err != nil {
		return err
	}

	for i := range instanceList.Items {
		// Checks for new instnace in the same namespace.
		if instanceList.Items[i].Spec.Namespace == namespace && instanceList.Items[i].Name != name {
			// Check if the earliest instance is not yet set or if this instance is earlier than the current earliest.
			if earliestInstance == nil || instanceList.Items[i].CreationTimestamp.Before(&earliestInstance.CreationTimestamp) {
				earliestInstance = &instanceList.Items[i]
			}
		}
	}

	// Trigger the earliestInstance
	if earliestInstance != nil {
		req := ctrl.Request{NamespacedName: types.NamespacedName{Name: earliestInstance.Name, Namespace: earliestInstance.Namespace}}
		r.Logger.Infof("Trigger again the earliest instance: %s", earliestInstance.Name)
		_, err := r.Reconcile(ctx, req)
		return err
	}

	return nil
}

// updateCondition updates the component status conditions.
func (r *InstanceReconciler) getNodeIP(ctx context.Context) (string, error) {
	nodeList := corev1.NodeList{}
	err := r.List(ctx, &nodeList)
	if err != nil {
		return "", fmt.Errorf("failed to get nodes: %w", err)
	}

	if len(nodeList.Items) == 0 {
		return "", fmt.Errorf("no nodes found in the cluster")
	}
	ip := nodeList.Items[0].Status.Addresses[0].Address
	return ip, nil
}
