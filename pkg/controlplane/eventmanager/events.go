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

package eventmanager

import "time"

type Direction int

const (
	Incoming Direction = iota
	Outgoing
)

type ConnectionState int

const (
	Ongoing ConnectionState = iota
	Complete
	Denied
	PeerDenied
)

const (
	ConnectionStatus = "ConnectionStatus"
)

type ConnectionStatusAttr struct {
	ConnectionID    string // Unique ID to track a connection from start to end within the gateway
	SrcService      string // Source application/service initiating the connection
	DstService      string // Destination application/service receiving the connection
	IncomingBytes   int
	OutgoingBytes   int
	DestinationPeer string // The peer(gateway) where the destination/source service is located depending on the Direction
	StartTstamp     time.Time
	LastTstamp      time.Time
	Direction       Direction // Incoming/Outgoing
	State           ConnectionState
}
