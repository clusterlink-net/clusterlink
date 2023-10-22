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

package k8s

import (
	"context"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PodReconciler contain information on the clusters pods.
type PodReconciler struct {
	client.Client
	ipToPod map[string]metav1.ObjectMeta
	logger  *logrus.Entry
}

// Reconcile watches all pods events and updates the PodReconciler.
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var pod corev1.Pod
	if err := r.Get(ctx, req.NamespacedName, &pod); err != nil {
		if apierrors.IsNotFound(err) {
			// the pod was deleted.
			r.deletePod(req.NamespacedName.Name, req.NamespacedName.Namespace)
			return ctrl.Result{}, nil
		}
		r.logger.Error(err, "unable to fetch Pod")
		return ctrl.Result{}, err
	}

	r.updatePod(pod)
	return ctrl.Result{}, nil
}

// deletePod deletes pod to ipToPod list.
func (r *PodReconciler) deletePod(name, namespace string) {
	for key, pod := range r.ipToPod {
		if pod.Name == name && pod.Namespace == namespace {
			delete(r.ipToPod, key)
		}
	}
}

// updatePod adds or updates pod to ipToPod list.
func (r *PodReconciler) updatePod(pod corev1.Pod) {
	for _, ip := range pod.Status.PodIPs {
		// ignoring host-networked Pod IPs
		if ip.IP != pod.Status.HostIP {
			r.ipToPod[ip.IP] = pod.ObjectMeta
		}
	}
}

// getLabelFromIP return all the labels for specific ip.
func (r *PodReconciler) getLabelFromIP(ip string) map[string]string {
	if p, ok := r.ipToPod[ip]; ok {
		return p.Labels
	}
	return nil
}

// setupWithManager setup PodReconciler for all the pods.
func (r *PodReconciler) setupWithManager(mgr *ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(*mgr).
		For(&corev1.Pod{}).
		Complete(r)
}

// NewPodReconciler creates pod reconciler for monitoring pods in the cluster.
func NewPodReconciler(mgr *ctrl.Manager) (*PodReconciler, error) {
	logger := logrus.WithField("component", "platform.k8s.podReconciler")
	r := PodReconciler{
		Client:  (*mgr).GetClient(),
		ipToPod: make(map[string]metav1.ObjectMeta),
		logger:  logger,
	}

	if err := r.setupWithManager(mgr); err != nil {
		return nil, err
	}
	r.logger.Info("start podReconciler")
	return &r, nil
}
