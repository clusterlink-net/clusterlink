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

package k8s_test

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/clusterlink-net/clusterlink/pkg/platform/k8s"
)

const (
	TestPodName      = "test-pod"
	TestPodNameSpace = "default"
	TestPodKeyLabel  = "app"
	TestPodIP        = "10.0.0.1"
)

func TestPodReconciler(t *testing.T) {
	// Test setup
	logger := logrus.WithField("component", "podReconciler")
	client, err := getFakeClient()
	require.NoError(t, err)
	ctx := context.Background()
	podReconciler := k8s.CreatePodReconciler(client, logger)

	req := ctrl.Request{NamespacedName: types.NamespacedName{
		Name:      TestPodName,
		Namespace: TestPodNameSpace,
	}}

	// Pod creation check
	createLabel := "create-label"
	pod := getFakePod(createLabel)
	err = podReconciler.Create(ctx, pod)
	require.NoError(t, err)
	_, err = podReconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	actualLabels := podReconciler.GetLabelsFromIP(TestPodIP)[TestPodKeyLabel]
	expectedLabels := createLabel
	require.Equal(t, expectedLabels, actualLabels, "Labels should be equal")

	// Pod update check
	updateLabel := "update-label"
	pod = getFakePod(updateLabel)
	err = podReconciler.Update(ctx, pod)
	require.NoError(t, err)
	_, err = podReconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	actualLabels = podReconciler.GetLabelsFromIP(TestPodIP)[TestPodKeyLabel]
	expectedLabels = updateLabel
	require.Equal(t, expectedLabels, actualLabels, "Labels should be equal")

	// Pod deletion check
	err = podReconciler.Delete(ctx, pod)
	require.NoError(t, err)
	_, err = podReconciler.Reconcile(ctx, req)
	require.NoError(t, err)
	labels := podReconciler.GetLabelsFromIP(TestPodIP)[TestPodKeyLabel]
	require.Empty(t, labels)
}

func getFakeClient(initObjs ...client.Object) (client.WithWatch, error) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, err
	}

	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjs...).Build(), nil
}

func getFakePod(label string) *corev1.Pod {
	return &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      TestPodName,
			Namespace: TestPodNameSpace,
			Labels: map[string]string{
				TestPodKeyLabel: label,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            TestPodName,
					Image:           TestPodName,
					ImagePullPolicy: "Always",
				},
			},
		},
		Status: corev1.PodStatus{
			PodIPs: []corev1.PodIP{
				{
					IP: TestPodIP,
				},
			},
		},
	}
}
