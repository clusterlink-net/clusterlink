package mbgControlplane

import (
	"github.ibm.com/mbg-agent/pkg/controlplane/store"
	"github.ibm.com/mbg-agent/pkg/protocol"
)

// Todo to replace with dexpose
func Disconnect(d protocol.DisconnectRequest) {
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
