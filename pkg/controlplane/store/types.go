package store

import (
	"github.com/clusterlink-net/clusterlink/pkg/api"
)

const (
	peerStoreName         = "peer"
	exportStoreName       = "export"
	importStoreName       = "import"
	bindingStoreName      = "binding"
	accessPolicyStoreName = "accessPolicy"

	bindingStructVersion      = 1
	exportStructVersion       = 1
	importStructVersion       = 1
	peerStructVersion         = 1
	accessPolicyStructVersion = 1
)

// Peer represents a remote peer.
type Peer struct {
	api.PeerSpec
	// Name of the peer.
	Name string
	// Version of the struct when object was created.
	Version uint32
}

// NewPeer creates a new peer.
func NewPeer(peer *api.Peer) *Peer {
	return &Peer{
		PeerSpec: peer.Spec,
		Name:     peer.Name,
		Version:  peerStructVersion,
	}
}

// Export represents a local service that may be exported.
type Export struct {
	api.ExportSpec
	// Name of the export.
	Name string
	// Version of the struct when object was created.
	Version uint32
}

// Keys return the keys identifying the export.
func (exp *Export) Keys() []string {
	return []string{exp.Name}
}

// NewExport creates a new export.
func NewExport(export *api.Export) *Export {
	return &Export{
		ExportSpec: export.Spec,
		Name:       export.Name,
		Version:    exportStructVersion,
	}
}

// Import represents an external service that can be bound to (multiple) exported services of remote peers.
type Import struct {
	api.ImportSpec
	// Name of import.
	Name string
	// Version of the struct when object was created.
	Version uint32
	// Port is the port where the imported service should listen on.
	Port uint16
}

// Keys return the keys identifying the import.
func (imp *Import) Keys() []string {
	return []string{imp.Name}
}

// NewImport creates a new import.
func NewImport(imp *api.Import) *Import {
	return &Import{
		ImportSpec: imp.Spec,
		Name:       imp.Name,
		Version:    importStructVersion,
	}
}

// Binding of an imported service to a remote exported service.
type Binding struct {
	api.BindingSpec
	// Version of the struct when object was created.
	Version uint32
}

// Keys return the keys identifying the binding.
func (b *Binding) Keys() []string {
	return []string{b.Import, b.Peer}
}

// NewBinding creates a new binding.
func NewBinding(binding *api.Binding) *Binding {
	return &Binding{
		BindingSpec: binding.Spec,
		Version:     bindingStructVersion,
	}
}

// AccessPolicy to allow/deny specific connections
type AccessPolicy struct {
	api.Policy
	// Version of the struct when object was created.
	Version uint32
}

// NewAccessPolicy creates a new access policy
func NewAccessPolicy(policy *api.Policy) *AccessPolicy {
	return &AccessPolicy{
		Policy:  *policy,
		Version: accessPolicyStructVersion,
	}
}
