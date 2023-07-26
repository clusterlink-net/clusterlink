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

// DataplaneSNI returns the dataplane SNI for a specific peer.
func DataplaneSNI(peer string) string {
	return fmt.Sprintf("%s.%s:%d", serverPrefix, peer, ListenPort)
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
