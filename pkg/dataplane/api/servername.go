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

package api

import (
	"fmt"
	"strings"
)

const (
	// serverPrefix is the prefix such that <serverPrefix>.<peer name> is the dataplane server name.
	serverPrefix = "dataplane"
	// ListenPort is the dataplane external listening port.
	ListenPort = 443
)

// DataplaneServerName returns the dataplane server name for a specific peer.
func DataplaneServerName(peer string) string {
	return fmt.Sprintf("%s.%s", serverPrefix, peer)
}

// DataplaneSNI returns the dataplane SNI for a specific peer.
func DataplaneSNI(peer string) string {
	return fmt.Sprintf("%s:%d", DataplaneServerName(peer), ListenPort)
}

// StripServerPrefix strips the dataplane server prefix from the dataplane server name, yielding the peer name.
func StripServerPrefix(serverName string) (string, error) {
	toStrip := serverPrefix + "."
	if !strings.HasPrefix(serverName, toStrip) {
		return "", fmt.Errorf("expected dataplane server name to start with '%s', but got: '%s'",
			toStrip, serverName)
	}

	return strings.TrimPrefix(serverName, toStrip), nil
}
