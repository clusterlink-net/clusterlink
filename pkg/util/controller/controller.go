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
	"reflect"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Spec holds everything needed to create a controller.
type Spec struct {
	// Name of the controller
	Name string
	// Object being watched.
	Object client.Object
	// AddHandler handles object create/update.
	AddHandler func(ctx context.Context, object any) error
	// DeleteHandler handles object deletes.
	DeleteHandler func(ctx context.Context, name types.NamespacedName) error
}

type reconciler struct {
	client        client.Client
	objectType    reflect.Type
	addHandler    func(ctx context.Context, object any) error
	deleteHandler func(ctx context.Context, name types.NamespacedName) error
	logger        *logrus.Entry
}

// Reconcile handles a single reconcile request.
func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger.Debugf("Reconcile: %v", req.NamespacedName)

	objectPtr := reflect.New(r.objectType).Interface()
	if err := r.client.Get(ctx, req.NamespacedName, objectPtr.(client.Object)); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, r.deleteHandler(ctx, req.NamespacedName)
		}

		r.logger.Errorf("Unable to get object: %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, r.addHandler(ctx, objectPtr)
}

func newReconciler(clnt client.Client, spec *Spec) *reconciler {
	return &reconciler{
		client:        clnt,
		objectType:    reflect.TypeOf(spec.Object).Elem(),
		addHandler:    spec.AddHandler,
		deleteHandler: spec.DeleteHandler,
		logger:        logrus.WithField("component", "controllers."+spec.Name),
	}
}

// AddToManager adds a new controller to the given manager.
func AddToManager(manager ctrl.Manager, spec *Spec) error {
	return ctrl.NewControllerManagedBy(manager).
		For(spec.Object).
		Complete(newReconciler(manager.GetClient(), spec))
}
