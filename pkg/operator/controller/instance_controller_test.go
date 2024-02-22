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

package controller_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	clusterlink "github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/operator/controller"
)

const (
	ClusterLinkName = "cl-job"
	Timeout         = time.Second * 30
	duration        = time.Second * 10
	IntervalTime    = time.Millisecond * 250
)

var (
	k8sClient client.Client
	ctx       context.Context
	cancel    context.CancelFunc
)

func TestMain(m *testing.M) {
	// Bootstrap test environment
	ctx, cancel = context.WithCancel(context.TODO())
	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "config", "operator", "crds")},
		ErrorIfCRDPathMissing: true,
		BinaryAssetsDirectory: filepath.Join("..", "..", "..", "bin", "k8s",
			fmt.Sprintf("1.28.3-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}

	var err error
	cfg, err := testEnv.Start()
	if err != nil {
		fmt.Printf("Failed to start test environment: %v\n", err)
		cancel()
		return
	}

	err = clusterlink.AddToScheme(scheme.Scheme)
	if err != nil {
		fmt.Printf("Failed to add API to scheme: %v\n", err)
		cancel()
		return
	}

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		fmt.Printf("Failed to create Kubernetes client: %v\n", err)
		cancel()
		return
	}

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	if err != nil {
		fmt.Printf("Failed to create controller manager: %v\n", err)
		cancel()
		return
	}

	err = (&controller.InstanceReconciler{
		Client:    k8sManager.GetClient(),
		Scheme:    k8sManager.GetScheme(),
		Logger:    logrus.WithField("component", "reconciler"),
		Instances: make(map[string]string),
	}).SetupWithManager(k8sManager)
	if err != nil {
		fmt.Printf("Failed to set up reconciler: %v\n", err)
		cancel()
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Recovered from panic: %v\n", r)
			}
		}()
		err := k8sManager.Start(ctx)
		if err != nil {
			fmt.Printf("Failed to run controller manager: %v\n", err)
		}
	}()

	code := m.Run()

	// Tearing down the test environment
	err = testEnv.Stop()
	if err != nil {
		fmt.Printf("Failed to stop test environment: %v\n", err)
	}

	// Exiting with the status code from the test run
	os.Exit(code)
}

func TestClusterLinkController(t *testing.T) {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	cl := clusterlink.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "clusterlink.net/v1alpha1",
			Kind:       "Clusterlink",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      ClusterLinkName,
			Namespace: controller.OperatorNamespace,
		},
	}

	cpID := types.NamespacedName{Name: controller.ControlPlaneName, Namespace: controller.InstanceNamespace}
	cpResource := []client.Object{&appsv1.Deployment{}, &corev1.Service{}, &corev1.PersistentVolumeClaim{}}
	roleID := types.NamespacedName{
		Name:      controller.ControlPlaneName + controller.InstanceNamespace,
		Namespace: controller.InstanceNamespace,
	}
	roleResource := []client.Object{&rbacv1.ClusterRole{}, &rbacv1.ClusterRoleBinding{}}
	dpID := types.NamespacedName{Name: controller.DataPlaneName, Namespace: controller.InstanceNamespace}
	dpResource := []client.Object{&appsv1.Deployment{}, &corev1.Service{}}
	ingressID := types.NamespacedName{Name: controller.IngressName, Namespace: controller.InstanceNamespace}

	t.Run("Create ClusterLink deployment", func(t *testing.T) {
		// Create ClusterLink namespaces
		opearatorNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: controller.OperatorNamespace,
			},
		}

		err := k8sClient.Create(ctx, opearatorNamespace)
		require.Nil(t, err)

		systemNamespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: controller.InstanceNamespace,
			},
		}

		err = k8sClient.Create(ctx, systemNamespace)
		require.Nil(t, err)

		// Create ClusterLink deployment
		err = k8sClient.Create(ctx, &cl)
		require.Nil(t, err)

		// Check Controlplane resources
		for _, r := range cpResource {
			checkResourceCreated(t, cpID, r)
		}

		for _, r := range roleResource {
			checkResourceCreated(t, roleID, r)
		}
		// Check controlplane fields
		cp := &appsv1.Deployment{}
		getResource(t, cpID, cp)
		cpImage := "ghcr.io/clusterlink-net/" + controller.ControlPlaneName + ":latest"
		require.Equal(t, cpImage, cp.Spec.Template.Spec.Containers[0].Image)
		require.Equal(t, "info", cp.Spec.Template.Spec.Containers[0].Args[1])

		// Check Dataplane resources
		for _, r := range dpResource {
			checkResourceCreated(t, dpID, r)
		}

		// Check Dataplane fields
		dp := &appsv1.Deployment{}
		getResource(t, dpID, dp)
		envoyImage := "ghcr.io/clusterlink-net/" + controller.DataPlaneName + ":latest"
		require.Equal(t, envoyImage, dp.Spec.Template.Spec.Containers[0].Image)
		require.Equal(t, int32(1), *dp.Spec.Replicas)
		require.Equal(t, "info", dp.Spec.Template.Spec.Containers[0].Args[1])

		// Check ingress resource not exist
		require.True(t, checkResourceNotExist(ingressID, &corev1.Service{}))
	})

	t.Run("Update ClusterLink deployment", func(t *testing.T) {
		svc := &corev1.Service{}
		// Update ingress resource to LoadBalancer type
		getResource(t, types.NamespacedName{Name: ClusterLinkName, Namespace: controller.OperatorNamespace}, &cl)
		cl.Spec.Ingress.Type = clusterlink.IngressTypeLoadBalancer
		err := k8sClient.Update(ctx, &cl)
		require.Nil(t, err)
		checkResourceCreated(t, ingressID, svc)
		require.Equal(t, corev1.ServiceTypeLoadBalancer, svc.Spec.Type)

		// Update ingress resource to NodePort type
		err = k8sClient.Delete(ctx, svc)
		require.Nil(t, err)
		getResource(t, types.NamespacedName{Name: ClusterLinkName, Namespace: controller.OperatorNamespace}, &cl)
		cl.Spec.Ingress.Type = clusterlink.IngressTypeNodePort
		cl.Spec.Ingress.Port = 30444
		err = k8sClient.Update(ctx, &cl)
		require.Nil(t, err)
		checkResourceCreated(t, ingressID, svc)
		require.Equal(t, corev1.ServiceTypeNodePort, svc.Spec.Type)
		require.Equal(t, int32(30444), svc.Spec.Ports[0].NodePort)

		// Update dataplane and controlpane
		goReplicas := 2
		loglevel := "debug"
		containerRegistry := "quay.com"
		imageTag := "v1.0.1"
		cp := &appsv1.Deployment{}
		dp := &appsv1.Deployment{}

		getResource(t, cpID, cp)
		getResource(t, dpID, dp)
		getResource(t, types.NamespacedName{Name: ClusterLinkName, Namespace: controller.OperatorNamespace}, &cl)

		/// Update Spec
		cl.Spec.DataPlane.Type = clusterlink.DataplaneTypeGo
		cl.Spec.DataPlane.Replicas = goReplicas
		cl.Spec.LogLevel = loglevel
		cl.Spec.ImageTag = imageTag
		cl.Spec.ContainerRegistry = containerRegistry
		err = k8sClient.Update(ctx, &cl)
		require.Nil(t, err)

		/// Check controlplane
		checkResourceCreated(t, cpID, cp)
		cpImage := containerRegistry + "/" + controller.ControlPlaneName + ":" + imageTag
		require.Equal(t, cpImage, cp.Spec.Template.Spec.Containers[0].Image)
		require.Equal(t, loglevel, cp.Spec.Template.Spec.Containers[0].Args[1])

		/// Check dataplane
		checkResourceCreated(t, dpID, dp)
		goImage := containerRegistry + "/" + controller.GoDataPlaneName + ":" + imageTag
		require.Equal(t, goImage, dp.Spec.Template.Spec.Containers[0].Image)
		require.Equal(t, int32(goReplicas), *dp.Spec.Replicas)
		require.Equal(t, loglevel, dp.Spec.Template.Spec.Containers[0].Args[1])
	})

	t.Run("Delete ClusterLink deployment", func(t *testing.T) {
		// Delete ClusterLink deployment
		err := k8sClient.Delete(ctx, &cl)
		require.Nil(t, err)
		// Controlplane resources
		for _, r := range cpResource {
			checkResourceDeleted(t, cpID, r)
		}

		// Dataplane resources
		for _, r := range dpResource {
			checkResourceDeleted(t, dpID, r)
		}

		// Ingress resource
		checkResourceDeleted(t, ingressID, &corev1.Service{})
	})
}

// checkResourceCreated checks k8s resource was deleted.
func checkResourceDeleted(t *testing.T, id types.NamespacedName, object client.Object) {
	t.Helper()
	require.Eventually(t, func() bool {
		err := k8sClient.Get(ctx, id, object)
		if err != nil {
			return apierrors.IsNotFound(err)
		}
		// Check if the resource has a deletion timestamp
		if !object.GetDeletionTimestamp().IsZero() {
			return true
		}
		// If the resource exists and has no deletion timestamp, continue waiting
		return false
	}, Timeout, IntervalTime)
}

// checkResourceCreated checks k8s resource was created.
func checkResourceCreated(t *testing.T, id types.NamespacedName, object client.Object) {
	t.Helper()
	require.Eventually(t, func() bool {
		err := k8sClient.Get(context.Background(), id, object)
		return err == nil
	}, Timeout, IntervalTime)
}

// checkResourceNotExist checks k8s resource is not existed.
func checkResourceNotExist(id types.NamespacedName, object client.Object) bool {
	err := k8sClient.Get(ctx, id, object)
	if err != nil {
		return apierrors.IsNotFound(err)
	}

	return false
}

// getResource gets the k8s resource according to the resource id.
func getResource(t *testing.T, id types.NamespacedName, object client.Object) {
	t.Helper()
	err := k8sClient.Get(context.Background(), id, object)
	require.Nil(t, err)
}
