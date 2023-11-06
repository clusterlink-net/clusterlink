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

func (d Direction) String() string {
	return [...]string{"Incoming", "Outgoing"}[d]
}

type Action int

const (
	Allow Action = iota
	Deny
	AllowAll
	AllowPartial
)

type ConnectionState int

const (
	Ongoing ConnectionState = iota
	Complete
	Denied
	PeerDenied
)

func (a Action) String() string {
	return [...]string{"Allow", "Deny", "AllowAll", "AllowPartial"}[a]
}

const Wildcard = "*"

const (
	NewConnectionRequest = "NewConnectionRequest"
	ConnectionStatus     = "ConnectionStatus"
	AddPeerRequest       = "AddPeerRequest"
	NewRemoteService     = "NewRemoteService"
	ExposeRequest        = "ExposeRequest"
	RemovePeerRequest    = "RemovePeerRequest"
	RemoveRemoteService  = "RemoveRemoteService"
)

type ConnectionRequestAttr struct {
	SrcService string
	DstService string
	Direction  Direction
	OtherPeer   string //Optional: Would not be set if its an outgoing connection
}

type ConnectionRequestResp struct {
	Action    Action
	TargetPeer string
	BitRate   int // Mbps
}

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

type NewRemoteServiceAttr struct {
	Service string
	Peer    string
}

type RemoveRemoteServiceAttr struct {
	Service string
	Peer  	string
}

type NewRemoteServiceResp struct {
	Action Action
}

type ExposeRequestAttr struct {
	Service string
}

type ExposeRequestResp struct {
	Action     	Action
	TargetPeers []string
}

type AddPeerAttr struct {
	Peer string
}

type AddPeerResp struct {
	Action Action
}

type RemovePeerAttr struct {
	Peer string
}

type ServiceListRequestAttr struct {
	SrcPeer string
}

type ServiceListRequestResp struct {
	Action   Action
	Services []string
}

type ServiceRequestAttr struct {
	SrcPeer string
}

type ServiceRequestResp struct {
	Action Action
}
