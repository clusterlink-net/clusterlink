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

package controlplane

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/clusterlink-net/clusterlink/pkg/api"
	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/authz"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/control"
	cpstore "github.com/clusterlink-net/clusterlink/pkg/controlplane/store"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/xds"
	"github.com/clusterlink-net/clusterlink/pkg/store"
)

// Instance of a controlplane, where all API servers delegate their requested actions to.
type Instance struct {
	namespace string

	peers      *cpstore.Peers
	exports    *cpstore.Exports
	imports    *cpstore.Imports
	bindings   *cpstore.Bindings
	acPolicies *cpstore.AccessPolicies
	lbPolicies *cpstore.LBPolicies

	authzManager   *authz.Manager
	controlManager *control.Manager
	xdsManager     *xds.Manager

	initialized bool

	logger *logrus.Entry
}

func toK8SExport(export *cpstore.Export, namespace string) *v1alpha1.Export {
	return &v1alpha1.Export{
		ObjectMeta: metav1.ObjectMeta{
			Name:      export.Name,
			Namespace: namespace,
		},
	}
}

func toK8SImport(imp *cpstore.Import, namespace string) *v1alpha1.Import {
	return &v1alpha1.Import{
		ObjectMeta: metav1.ObjectMeta{
			Name:      imp.Name,
			Namespace: namespace,
		},
		Spec: v1alpha1.ImportSpec{
			Port:       imp.ImportSpec.Service.Port,
			TargetPort: imp.Port,
		},
	}
}

func toK8SPeer(pr *cpstore.Peer) *v1alpha1.Peer {
	k8sPeer := &v1alpha1.Peer{
		ObjectMeta: metav1.ObjectMeta{Name: pr.Name},
		Spec: v1alpha1.PeerSpec{
			Gateways: make([]v1alpha1.Endpoint, len(pr.Gateways)),
		},
	}

	for i, gw := range pr.PeerSpec.Gateways {
		k8sPeer.Spec.Gateways[i].Host = gw.Host
		k8sPeer.Spec.Gateways[i].Port = gw.Port
	}

	return k8sPeer
}

// CreatePeer defines a new route target for egress dataplane connections.
func (cp *Instance) CreatePeer(peer *cpstore.Peer) error {
	cp.logger.Infof("Creating peer '%s'.", peer.Name)

	if cp.initialized {
		if err := cp.peers.Create(peer); err != nil {
			return err
		}
	}

	k8sPeer := toK8SPeer(peer)
	cp.authzManager.AddPeer(k8sPeer)
	return cp.xdsManager.AddPeer(k8sPeer)
}

// UpdatePeer updates new route target for egress dataplane connections.
func (cp *Instance) UpdatePeer(peer *cpstore.Peer) error {
	cp.logger.Infof("Updating peer '%s'.", peer.Name)

	err := cp.peers.Update(peer.Name, func(old *cpstore.Peer) *cpstore.Peer {
		return peer
	})
	if err != nil {
		return err
	}

	k8sPeer := toK8SPeer(peer)
	cp.authzManager.AddPeer(k8sPeer)
	return cp.xdsManager.AddPeer(k8sPeer)
}

// GetPeer returns an existing peer.
func (cp *Instance) GetPeer(name string) *cpstore.Peer {
	cp.logger.Infof("Getting peer '%s'.", name)
	return cp.peers.Get(name)
}

// DeletePeer removes the possibility for egress dataplane connections to be routed to a given peer.
func (cp *Instance) DeletePeer(name string) (*cpstore.Peer, error) {
	cp.logger.Infof("Deleting peer '%s'.", name)

	pr, err := cp.peers.Delete(name)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, nil
	}

	cp.authzManager.DeletePeer(name)

	err = cp.xdsManager.DeletePeer(name)
	if err != nil {
		// practically impossible
		return nil, err
	}

	return pr, nil
}

// GetAllPeers returns the list of all peers.
func (cp *Instance) GetAllPeers() []*cpstore.Peer {
	cp.logger.Info("Listing all peers.")
	return cp.peers.GetAll()
}

// CreateExport defines a new route target for ingress dataplane connections.
func (cp *Instance) CreateExport(export *cpstore.Export) error {
	cp.logger.Infof("Creating export '%s'.", export.Name)

	cp.authzManager.AddExport(toK8SExport(export, cp.namespace))

	if cp.initialized {
		if err := cp.exports.Create(export); err != nil {
			return err
		}

		err := cp.controlManager.AddLegacyExport(
			export.Name, cp.namespace, &export.ExportSpec)
		if err != nil {
			return err
		}
	}

	return cp.xdsManager.AddLegacyExport(
		export.Name, cp.namespace, export.Service.Host, export.Service.Port)
}

// UpdateExport updates a new route target for ingress dataplane connections.
func (cp *Instance) UpdateExport(export *cpstore.Export) error {
	cp.logger.Infof("Updating export '%s'.", export.Name)

	cp.authzManager.AddExport(toK8SExport(export, cp.namespace))

	err := cp.exports.Update(export.Name, func(old *cpstore.Export) *cpstore.Export {
		return export
	})
	if err != nil {
		return err
	}

	err = cp.controlManager.AddLegacyExport(
		export.Name, cp.namespace, &export.ExportSpec)
	if err != nil {
		return err
	}

	return cp.xdsManager.AddLegacyExport(export.Name, cp.namespace, export.Service.Host, export.Service.Port)
}

// GetExport returns an existing export.
func (cp *Instance) GetExport(name string) *cpstore.Export {
	cp.logger.Infof("Getting export '%s'.", name)
	return cp.exports.Get(name)
}

// DeleteExport removes the possibility for ingress dataplane connections to access a given service.
func (cp *Instance) DeleteExport(name string) (*cpstore.Export, error) {
	cp.logger.Infof("Deleting export '%s'.", name)

	export, err := cp.exports.Delete(name)
	if err != nil {
		return nil, err
	}
	if export == nil {
		return nil, nil
	}

	err = cp.controlManager.DeleteLegacyExport(cp.namespace, &export.ExportSpec)
	if err != nil {
		return nil, err
	}

	namespacedName := types.NamespacedName{
		Name:      name,
		Namespace: cp.namespace,
	}
	if err := cp.xdsManager.DeleteExport(namespacedName); err != nil {
		// practically impossible
		return export, err
	}

	cp.authzManager.DeleteExport(namespacedName)

	return export, nil
}

// GetAllExports returns the list of all exports.
func (cp *Instance) GetAllExports() []*cpstore.Export {
	cp.logger.Info("Listing all exports.")
	return cp.exports.GetAll()
}

// CreateImport creates a listening socket for an imported remote service.
func (cp *Instance) CreateImport(imp *cpstore.Import) error {
	cp.logger.Infof("Creating import '%s'.", imp.Name)

	k8sImp := toK8SImport(imp, cp.namespace)

	if cp.initialized {
		if err := cp.imports.Create(imp); err != nil {
			return err
		}

		err := cp.controlManager.AddImport(context.Background(), k8sImp)
		if err != nil {
			return err
		}

		imp.Port = k8sImp.Spec.TargetPort

		err = cp.imports.Update(imp.Name, func(old *cpstore.Import) *cpstore.Import {
			return imp
		})
		if err != nil {
			return err
		}
	}

	if err := cp.xdsManager.AddImport(k8sImp); err != nil {
		// practically impossible
		return err
	}

	return nil
}

// UpdateImport updates a listening socket for an imported remote service.
func (cp *Instance) UpdateImport(imp *cpstore.Import) error {
	cp.logger.Infof("Updating import '%s'.", imp.Name)

	err := cp.imports.Update(imp.Name, func(old *cpstore.Import) *cpstore.Import {
		return imp
	})
	if err != nil {
		return err
	}

	k8sImp := toK8SImport(imp, cp.namespace)
	err = cp.controlManager.AddImport(context.Background(), k8sImp)
	if err != nil {
		return err
	}

	imp.Port = k8sImp.Spec.TargetPort

	err = cp.imports.Update(imp.Name, func(old *cpstore.Import) *cpstore.Import {
		return imp
	})
	if err != nil {
		return err
	}

	if err := cp.xdsManager.AddImport(k8sImp); err != nil {
		// practically impossible
		return err
	}

	return nil
}

// GetImport returns an existing import.
func (cp *Instance) GetImport(name string) *cpstore.Import {
	cp.logger.Infof("Getting import '%s'.", name)
	return cp.imports.Get(name)
}

// DeleteImport removes the listening socket of a previously imported service.
func (cp *Instance) DeleteImport(name string) (*cpstore.Import, error) {
	cp.logger.Infof("Deleting import '%s'.", name)

	imp, err := cp.imports.Delete(name)
	if err != nil {
		return nil, err
	}
	if imp == nil {
		return nil, nil
	}

	namespacedName := types.NamespacedName{
		Name:      name,
		Namespace: cp.namespace,
	}
	if err := cp.xdsManager.DeleteImport(namespacedName); err != nil {
		// practically impossible
		return imp, err
	}

	err = cp.controlManager.DeleteImport(
		context.Background(),
		toK8SImport(imp, cp.namespace))
	if err != nil {
		return nil, err
	}

	return imp, nil
}

// GetAllImports returns the list of all imports.
func (cp *Instance) GetAllImports() []*cpstore.Import {
	cp.logger.Info("Listing all imports.")
	return cp.imports.GetAll()
}

// CreateBinding creates a binding of an imported service to a remote exported service.
func (cp *Instance) CreateBinding(binding *cpstore.Binding) error {
	cp.logger.Infof("Creating binding '%s'->'%s'.", binding.Import, binding.Peer)

	cp.authzManager.AddImport(&v1alpha1.Import{
		ObjectMeta: metav1.ObjectMeta{Name: binding.Import},
		Spec: v1alpha1.ImportSpec{
			Sources: []v1alpha1.ImportSource{
				{
					Peer:       binding.Peer,
					ExportName: binding.Import,
				},
			},
		},
	})

	if cp.initialized {
		if err := cp.bindings.Create(binding); err != nil {
			return err
		}
	}

	return nil
}

// UpdateBinding updates a binding of an imported service to a remote exported service.
func (cp *Instance) UpdateBinding(binding *cpstore.Binding) error {
	cp.logger.Infof("Updating binding '%s'->'%s'.", binding.Import, binding.Peer)

	cp.authzManager.AddImport(&v1alpha1.Import{
		ObjectMeta: metav1.ObjectMeta{Name: binding.Import},
		Spec: v1alpha1.ImportSpec{
			Sources: []v1alpha1.ImportSource{
				{
					Peer:       binding.Peer,
					ExportName: binding.Import,
				},
			},
		},
	})

	err := cp.bindings.Update(binding, func(old *cpstore.Binding) *cpstore.Binding {
		return binding
	})
	if err != nil {
		return err
	}

	return nil
}

// GetBindings returns all bindings for a given imported service.
func (cp *Instance) GetBindings(imp string) []*cpstore.Binding {
	cp.logger.Infof("Getting bindings for import '%s'.", imp)
	return cp.bindings.Get(imp)
}

// DeleteBinding removes a binding of an imported service to a remote exported service.
func (cp *Instance) DeleteBinding(binding *cpstore.Binding) (*cpstore.Binding, error) {
	cp.logger.Infof("Deleting binding '%s'->'%s'.", binding.Import, binding.Peer)

	// TODO: m.authzManager.Delete*

	return cp.bindings.Delete(binding)
}

// GetAllBindings returns the list of all bindings.
func (cp *Instance) GetAllBindings() []*cpstore.Binding {
	cp.logger.Info("Listing all bindings.")
	return cp.bindings.GetAll()
}

// CreateAccessPolicy creates an access policy to allow/deny specific connections.
func (cp *Instance) CreateAccessPolicy(policy *cpstore.AccessPolicy) error {
	cp.logger.Infof("Creating access policy '%s'.", policy.Spec.Blob)

	if cp.initialized {
		if err := cp.acPolicies.Create(policy); err != nil {
			return err
		}
	}

	return cp.authzManager.AddAccessPolicy(&api.Policy{Spec: policy.Spec})
}

// UpdateAccessPolicy updates an access policy to allow/deny specific connections.
func (cp *Instance) UpdateAccessPolicy(policy *cpstore.AccessPolicy) error {
	cp.logger.Infof("Updating access policy '%s'.", policy.Spec.Blob)

	err := cp.acPolicies.Update(policy.Name, func(old *cpstore.AccessPolicy) *cpstore.AccessPolicy {
		return policy
	})
	if err != nil {
		return err
	}

	return cp.authzManager.AddAccessPolicy(&api.Policy{Spec: policy.Spec})
}

// DeleteAccessPolicy removes an access policy to allow/deny specific connections.
func (cp *Instance) DeleteAccessPolicy(name string) (*cpstore.AccessPolicy, error) {
	cp.logger.Infof("Deleting access policy '%s'.", name)

	policy, err := cp.acPolicies.Delete(name)
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return nil, nil
	}

	if err := cp.authzManager.DeleteAccessPolicy(&policy.Policy); err != nil {
		return nil, err
	}

	return policy, err
}

// GetAccessPolicy returns an access policy with the given name.
func (cp *Instance) GetAccessPolicy(name string) *cpstore.AccessPolicy {
	cp.logger.Infof("Getting access policy '%s'.", name)
	return cp.acPolicies.Get(name)
}

// GetAllAccessPolicies returns the list of all AccessPolicies.
func (cp *Instance) GetAllAccessPolicies() []*cpstore.AccessPolicy {
	cp.logger.Info("Listing all access policies.")
	return cp.acPolicies.GetAll()
}

// CreateLBPolicy creates a load-balancing policy to set a load-balancing scheme for specific connections.
func (cp *Instance) CreateLBPolicy(policy *cpstore.LBPolicy) error {
	cp.logger.Infof("Creating load-balancing policy '%s'.", policy.Spec.Blob)

	if cp.initialized {
		if err := cp.lbPolicies.Create(policy); err != nil {
			return err
		}
	}

	return cp.authzManager.AddLBPolicy(&api.Policy{Spec: policy.Spec})
}

// UpdateLBPolicy updates a load-balancing policy.
func (cp *Instance) UpdateLBPolicy(policy *cpstore.LBPolicy) error {
	cp.logger.Infof("Updating load-balancing policy '%s'.", policy.Spec.Blob)

	err := cp.lbPolicies.Update(policy.Name, func(old *cpstore.LBPolicy) *cpstore.LBPolicy {
		return policy
	})
	if err != nil {
		return err
	}

	return cp.authzManager.AddLBPolicy(&api.Policy{Spec: policy.Spec})
}

// DeleteLBPolicy removes a load-balancing policy.
func (cp *Instance) DeleteLBPolicy(name string) (*cpstore.LBPolicy, error) {
	cp.logger.Infof("Deleting load-balancing policy '%s'.", name)

	policy, err := cp.lbPolicies.Delete(name)
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return nil, nil
	}

	if err := cp.authzManager.DeleteLBPolicy(&policy.Policy); err != nil {
		return nil, err
	}

	return policy, nil
}

// GetLBPolicy returns a load-balancing policy with the given name.
func (cp *Instance) GetLBPolicy(name string) *cpstore.LBPolicy {
	cp.logger.Infof("Getting load-balancing policy '%s'.", name)
	return cp.lbPolicies.Get(name)
}

// GetAllLBPolicies returns the list of all load-balancing Policies.p.
func (cp *Instance) GetAllLBPolicies() []*cpstore.LBPolicy {
	cp.logger.Info("Listing all load-balancing policies.")
	return cp.lbPolicies.GetAll()
}

// init initializes the controlplane manager.
func (cp *Instance) init() error {
	// add peers
	for _, p := range cp.GetAllPeers() {
		if err := cp.CreatePeer(p); err != nil {
			return err
		}
	}

	// add exports
	for _, export := range cp.GetAllExports() {
		if err := cp.CreateExport(export); err != nil {
			return err
		}
	}

	// add exports
	for _, imp := range cp.GetAllImports() {
		if err := cp.CreateImport(imp); err != nil {
			return err
		}
	}

	// add bindings
	for _, binding := range cp.GetAllBindings() {
		if err := cp.CreateBinding(binding); err != nil {
			return err
		}
	}

	// add access policies
	for _, policy := range cp.GetAllAccessPolicies() {
		if err := cp.CreateAccessPolicy(policy); err != nil {
			return err
		}
	}

	// add load-balancing policies
	for _, policy := range cp.GetAllLBPolicies() {
		if err := cp.CreateLBPolicy(policy); err != nil {
			return err
		}
	}

	cp.initialized = true

	return nil
}

// NewInstance returns a new controlplane instance.
func NewInstance(
	storeManager store.Manager,
	authzManager *authz.Manager,
	controlManager *control.Manager,
	xdsManager *xds.Manager,
	namespace string,
) (*Instance, error) {
	logger := logrus.WithField("component", "controlplane")

	peers, err := cpstore.NewPeers(storeManager)
	if err != nil {
		return nil, fmt.Errorf("cannot load peers from store: %w", err)
	}
	logger.Infof("Loaded %d peers.", peers.Len())

	exports, err := cpstore.NewExports(storeManager)
	if err != nil {
		return nil, fmt.Errorf("cannot load exports from store: %w", err)
	}
	logger.Infof("Loaded %d exports.", exports.Len())

	imports, err := cpstore.NewImports(storeManager)
	if err != nil {
		return nil, fmt.Errorf("cannot load imports from store: %w", err)
	}
	logger.Infof("Loaded %d imports.", imports.Len())

	bindings, err := cpstore.NewBindings(storeManager)
	if err != nil {
		return nil, fmt.Errorf("cannot load bindings from store: %w", err)
	}
	logger.Infof("Loaded %d bindings.", bindings.Len())

	acPolicies, err := cpstore.NewAccessPolicies(storeManager)
	if err != nil {
		return nil, fmt.Errorf("cannot load access policies from store: %w", err)
	}
	logger.Infof("Loaded %d access policies.", acPolicies.Len())

	lbPolicies, err := cpstore.NewLBPolicies(storeManager)
	if err != nil {
		return nil, fmt.Errorf("cannot load load-balancing policies from store: %w", err)
	}
	logger.Infof("Loaded %d load-balancing policies.", lbPolicies.Len())

	cp := &Instance{
		namespace:      namespace,
		peers:          peers,
		exports:        exports,
		imports:        imports,
		bindings:       bindings,
		acPolicies:     acPolicies,
		lbPolicies:     lbPolicies,
		authzManager:   authzManager,
		controlManager: controlManager,
		xdsManager:     xdsManager,
		initialized:    false,
		logger:         logger,
	}

	// initialize instance
	if err := cp.init(); err != nil {
		return nil, err
	}

	return cp, nil
}
