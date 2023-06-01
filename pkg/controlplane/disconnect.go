package controlplane

import (
	apiObject "github.ibm.com/mbg-agent/pkg/controlplane/api/object"
	"github.ibm.com/mbg-agent/pkg/controlplane/store"
)

// Todo to replace with dexpose
func Disconnect(d apiObject.DisconnectRequest) {
	//Update MBG state
	store.UpdateState()
	connectionID := d.Id
	if store.IsServiceLocal(d.IdDest) {
		store.FreeUpPorts(connectionID)
		// Need to Kill the corresponding process
	} else {
		// Need to just Kill the corresponding process
	}

}
