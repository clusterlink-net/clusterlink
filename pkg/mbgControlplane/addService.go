package mbgControlplane

import (
	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/pkg/protocol"
)

//var mlog = logrus.WithField("component", "mbgControlPlane/AddService")

func AddService(e protocol.AddServiceRequest) {
	//Update MBG state
	state.UpdateState()
	state.AddLocalService(e.Id, e.Ip, e.Domain)
}
