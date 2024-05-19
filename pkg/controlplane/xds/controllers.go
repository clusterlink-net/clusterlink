// Copyright (c) The ClusterLink Authors.
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

package xds

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/util/controller"
)

// CreateControllers creates the various k8s controllers used to update the xDS manager.
func CreateControllers(mgr *Manager, controllerManager ctrl.Manager) error {
	err := controller.AddToManager(controllerManager, &controller.Spec{
		Name:   "xds.export",
		Object: &v1alpha1.Export{},
		AddHandler: func(ctx context.Context, object any) error {
			return mgr.AddExport(object.(*v1alpha1.Export))
		},
		DeleteHandler: func(ctx context.Context, name types.NamespacedName) error {
			return mgr.DeleteExport(name)
		},
	})
	if err != nil {
		return err
	}

	err = controller.AddToManager(controllerManager, &controller.Spec{
		Name:   "xds.peer",
		Object: &v1alpha1.Peer{},
		AddHandler: func(ctx context.Context, object any) error {
			return mgr.AddPeer(object.(*v1alpha1.Peer))
		},
		DeleteHandler: func(ctx context.Context, name types.NamespacedName) error {
			return mgr.DeletePeer(name.Name)
		},
	})
	if err != nil {
		return err
	}

	return controller.AddToManager(controllerManager, &controller.Spec{
		Name:   "xds.import",
		Object: &v1alpha1.Import{},
		AddHandler: func(ctx context.Context, object any) error {
			return mgr.AddImport(object.(*v1alpha1.Import))
		},
		DeleteHandler: func(ctx context.Context, name types.NamespacedName) error {
			return mgr.DeleteImport(name)
		},
	})
}
