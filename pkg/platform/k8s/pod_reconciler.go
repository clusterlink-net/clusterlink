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
	"sync"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type podInfo struct {
	name      string
	namespace string
	labels    map[string]string
}

// PodReconciler contain information on the clusters pods.
type PodReconciler struct {
	client.Client
	lock    sync.RWMutex
	ipToPod map[string]types.NamespacedName
	podList map[types.NamespacedName]podInfo
	logger  *logrus.Entry
}

// Reconcile watches all pods events and updates the PodReconciler.
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var pod corev1.Pod
	if err := r.Get(ctx, req.NamespacedName, &pod); err != nil {
		if apierrors.IsNotFound(err) {
			// the pod was deleted.
			r.deletePod(req.NamespacedName)
			return ctrl.Result{}, nil
		}
		r.logger.Error(err, "unable to fetch Pod")
		return ctrl.Result{}, err
	}

	r.updatePod(pod)
	return ctrl.Result{}, nil
}

// deletePod deletes pod to ipToPod list.
func (r *PodReconciler) deletePod(podID types.NamespacedName) {
	r.lock.Lock()
	defer r.lock.Unlock()

	delete(r.podList, podID)
	for key, pod := range r.ipToPod {
		if pod.Name == podID.Name && pod.Namespace == podID.Namespace {
			delete(r.ipToPod, key)
		}
	}
}

// updatePod adds or updates pod to ipToPod and podList.
func (r *PodReconciler) updatePod(pod corev1.Pod) {
	r.lock.Lock()
	defer r.lock.Unlock()

	podID := types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}
	r.podList[podID] = podInfo{name: pod.Name, namespace: pod.Namespace, labels: pod.Labels}
	for _, ip := range pod.Status.PodIPs {
		// ignoring host-networked Pod IPs
		if ip.IP != pod.Status.HostIP {
			r.ipToPod[ip.IP] = podID
		}
	}
}

// getLabelsFromIP return all the labels for specific ip.
func (r *PodReconciler) getLabelsFromIP(ip string) map[string]string {
	r.lock.RLock()
	defer r.lock.RUnlock()

	if p, ipExsit := r.ipToPod[ip]; ipExsit {
		if pInfo, podExist := r.podList[p]; podExist {
			return pInfo.labels
		}
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
		ipToPod: make(map[string]types.NamespacedName),
		podList: make(map[types.NamespacedName]podInfo),
		logger:  logger,
	}

	if err := r.setupWithManager(mgr); err != nil {
		return nil, err
	}
	r.logger.Info("start podReconciler")
	return &r, nil
}
