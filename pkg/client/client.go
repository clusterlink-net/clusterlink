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

package client

import (
	"crypto/tls"
	"encoding/json"

	"github.com/clusterlink-net/clusterlink/pkg/api"
	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	event "github.com/clusterlink-net/clusterlink/pkg/controlplane/eventmanager"
	"github.com/clusterlink-net/clusterlink/pkg/util/jsonapi"
	"github.com/clusterlink-net/clusterlink/pkg/util/rest"
)

// Client for accessing the API.
type Client struct {
	client *jsonapi.Client

	// Peers client.
	Peers *rest.Client
	// Exports client.
	Exports *rest.Client
	// Imports client.
	Imports *rest.Client
	// Access policies client.
	AccessPolicies *rest.Client
	// Load-balancing policies client.
	LBPolicies *rest.Client
}

// New returns a new client.
func New(host string, port uint16, tlsConfig *tls.Config) *Client {
	client := jsonapi.NewClient(host, port, tlsConfig)
	return &Client{
		client: client,
		Peers: rest.NewClient(&rest.Config{
			Client:       client,
			BasePath:     "/peers",
			SampleObject: api.Peer{},
			SampleList:   []api.Peer{},
		}),
		Exports: rest.NewClient(&rest.Config{
			Client:       client,
			BasePath:     "/exports",
			SampleObject: v1alpha1.Export{},
			SampleList:   []v1alpha1.Export{},
		}),
		Imports: rest.NewClient(&rest.Config{
			Client:       client,
			BasePath:     "/imports",
			SampleObject: api.Import{},
			SampleList:   []api.Import{},
		}),
		AccessPolicies: rest.NewClient(&rest.Config{
			Client:       client,
			BasePath:     "/policies",
			SampleObject: api.Policy{},
			SampleList:   []api.Policy{},
		}),
		LBPolicies: rest.NewClient(&rest.Config{
			Client:       client,
			BasePath:     "/lbpolicies",
			SampleObject: api.Policy{},
			SampleList:   []api.Policy{},
		}),
	}
}

func (c *Client) GetMetrics() (map[string]event.ConnectionStatusAttr, error) {
	var connections map[string]event.ConnectionStatusAttr
	path := "/metrics/" + event.ConnectionStatus
	resp, err := c.client.Get(path)
	if err != nil {
		return make(map[string]event.ConnectionStatusAttr), err
	}

	if err := json.Unmarshal(resp.Body, &connections); err != nil {
		return make(map[string]event.ConnectionStatusAttr), err
	}
	return connections, nil
}
