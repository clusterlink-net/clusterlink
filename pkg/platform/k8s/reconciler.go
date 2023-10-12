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
type reconciler struct {
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
func (r *reconciler) CreateResource(obj client.Object) {
	err := r.client.Create(context.Background(), obj)
	if err != nil {
		r.logger.Errorf("error occurred while creating K8s %v %v:", reflect.TypeOf(obj).String(), err)
		r.failedList[obj.GetName()] = reconcileObj{spec: obj, op: createOp}
		return
	}

	r.list[obj.GetName()] = obj
}

// UpdateResource updates k8s resource.
func (r *reconciler) UpdateResource(obj client.Object) {
	err := r.client.Update(context.Background(), obj)
	if err != nil {
		r.logger.Errorf("error occurred while updating K8s %v %v:", reflect.TypeOf(obj).String(), err)
		r.failedList[obj.GetName()] = reconcileObj{spec: obj, op: updateOp}
		return
	}

	r.list[obj.GetName()] = obj
}

// DeleteResource deletes k8s resource.
func (r *reconciler) DeleteResource(obj client.Object) {
	err := r.client.Delete(context.Background(), obj)
	if err != nil {
		r.logger.Errorf("error occurred while deleting K8s %v %v:", reflect.TypeOf(obj).String(), err)
		r.failedList[obj.GetName()] = reconcileObj{spec: obj, op: updateOp}
		return
	}

	delete(r.list, obj.GetName())
}

// func (r *reconciler) reconcile() {
// 	// TODO -to reconcile the failedList resources
// }

// NewReconciler returns reconciler for k8s objects.
func NewReconciler(cl client.Client) *reconciler {
	logger := logrus.WithField("component", "reconciler.k8s")

	return &reconciler{
		client:     cl,
		list:       make(map[string]client.Object),
		failedList: make(map[string]reconcileObj),
		logger:     logger,
	}
}
