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
	"reflect"

	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type reconcileObj struct {
	spec client.Object
	op   string
}

// Reconciler contain list of k8s objects and their state.
type Reconciler struct {
	client     client.Client
	list       map[string]client.Object
	failedList map[string]reconcileObj
	logger     *logrus.Entry
}

const (
	createOp = "create"
	deleteOp = "delete"
	updateOp = "update"
)

// CreateResource creates k8s resource.
func (r *Reconciler) CreateResource(name string, obj client.Object) {
	err := r.client.Create(context.Background(), obj)
	if err != nil {
		r.logger.Errorf("error occurred while creating K8s %v %v:", reflect.TypeOf(obj).String(), err)
		r.failedList[name] = reconcileObj{spec: obj, op: createOp}
		return
	}

	r.list[name] = obj
}

// UpdateResource updates k8s resource.
func (r *Reconciler) UpdateResource(name string, obj client.Object) {
	oldObj, ok := r.list[name]
	if !ok {
		r.logger.Errorf("error occurred while updating K8s %v: %v is not exist", reflect.TypeOf(obj).String(), name)
		return
	}

	if obj.GetName() == oldObj.GetName() { // object existed, update
		err := r.client.Update(context.Background(), obj)
		if err != nil {
			r.logger.Errorf("error occurred while updating K8s %v %v:", reflect.TypeOf(obj).String(), err)
			r.failedList[name] = reconcileObj{spec: obj, op: updateOp}
			return
		}
		r.list[name] = obj
		return
	}

	r.CreateResource(name, obj) // on success r.list[name] is set to object
	if r.list[name] == obj {    // deleting old object if a new one was successfully created
		if err := r.client.Delete(context.Background(), oldObj); err != nil {
			r.logger.Errorf("error occurred while deleting K8s %v %v:", reflect.TypeOf(oldObj).String(), err)
			r.failedList[name] = reconcileObj{spec: oldObj, op: updateOp}
		}
	}
}

// DeleteResource deletes k8s resource.
func (r *Reconciler) DeleteResource(name string, obj client.Object) {
	err := r.client.Delete(context.Background(), obj)
	if err != nil {
		r.logger.Errorf("error occurred while deleting K8s %v %v:", reflect.TypeOf(obj).String(), err)
		r.failedList[name] = reconcileObj{spec: obj, op: updateOp}
	}

	delete(r.list, name)
}

// NewReconciler returns reconciler for k8s objects.
func NewReconciler(cl client.Client) *Reconciler {
	logger := logrus.WithField("component", "reconciler.k8s")

	return &Reconciler{
		client:     cl,
		list:       make(map[string]client.Object),
		failedList: make(map[string]reconcileObj),
		logger:     logger,
	}
}
