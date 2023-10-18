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

package apiobject

import (
	"time"

	"github.com/clusterlink-net/clusterlink/pkg/controlplane/eventmanager"
)

// ImportReply return dataplane information (port) about new import endpoint
type ImportReply struct {
	ID   string
	Port string
}

// HeartBeats
type HeartBeat struct {
	ID string
}

// ConnectRequest
type ConnectRequest struct {
	ID     string
	IDDest string
	Policy string
	MbgID  string
}

type ConnectReply struct {
	Connect     bool
	ConnectType string
	ConnectDest string
}

// New connection to import service request
type NewImportConnParmaReq struct {
	SrcIP  string
	DestIP string
	DestID string
}

// New connection to import service reply
type NewImportConnParmaReply struct {
	Action string
	Target string
	SrcID  string
	ConnID string
}

// New connection to export service struct request
type NewExportConnParmaReq struct {
	SrcID   string
	SrcGwID string
	DestID  string
}

// New connection to import service struct reply
type NewExportConnParmaReply struct {
	Action          string
	SrcGwEndpoint   string
	DestSvcEndpoint string
	ConnID          string
}

// Connection Status
type ConnectionStatus struct {
	ConnectionID  string
	GlobalID      string // To be used to trace a flow across gateways
	IncomingBytes int
	OutgoingBytes int
	StartTstamp   time.Time
	LastTstamp    time.Time
	Direction     eventmanager.Direction
	State         eventmanager.ConnectionState
}
