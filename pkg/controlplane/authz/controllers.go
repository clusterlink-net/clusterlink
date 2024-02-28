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

package authz

import (
	"context"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type podReconciler struct {
	client  client.Client
	manager *Manager
	logger  *logrus.Entry
}

// Reconcile Pod objects.
func (r *podReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger.Debugf("Reconcile: %v", req.NamespacedName)

	var pod v1.Pod
	if err := r.client.Get(ctx, req.NamespacedName, &pod); err != nil {
		if errors.IsNotFound(err) {
			r.manager.deletePod(req.NamespacedName)
			return ctrl.Result{}, nil
		}

		r.logger.Errorf("Unable to get pod: %v", err)
		return ctrl.Result{}, err
	}

	r.manager.addPod(&pod)
	return ctrl.Result{}, nil
}

func newPodReconciler(manager *Manager, clnt client.Client) *podReconciler {
	return &podReconciler{
		client:  clnt,
		manager: manager,
		logger: logrus.WithField(
			"component", "controlplane.authz.pod-reconciler"),
	}
}

// CreateControllers creates the various k8s controllers used to update the authz manager.
func CreateControllers(mgr *Manager, controllerManager ctrl.Manager) error {
	k8sClient := controllerManager.GetClient()

	return ctrl.NewControllerManagedBy(controllerManager).
		For(&v1.Pod{}).
		Complete(newPodReconciler(mgr, k8sClient))
}
