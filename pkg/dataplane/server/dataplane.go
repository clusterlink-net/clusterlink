package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"

	"github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
	"github.com/clusterlink-net/clusterlink/pkg/util"
)

// Dataplane implements the server and api client which sends authorization to the control plane
type Dataplane struct {
	ID                 string
	peerName           string
	router             *chi.Mux
	apiClient          *http.Client
	parsedCertData     *util.ParsedCertData
	controlplaneTarget string
	clusters           map[string]*cluster.Cluster
	listeners          map[string]*listener.Listener
	listenerEnd        map[string]chan bool
	logger             *logrus.Entry
}

// GetClusterTarget returns the cluster address:port from the cluster map
func (d *Dataplane) GetClusterTarget(name string) (string, error) {
	if _, ok := d.clusters[name]; !ok {
		return "", fmt.Errorf("unable to find %s in clustermap ", name)
	}
	address := d.clusters[name].LoadAssignment.GetEndpoints()[0].LbEndpoints[0].GetEndpoint().Address.GetSocketAddress().GetAddress()
	port := d.clusters[name].LoadAssignment.GetEndpoints()[0].LbEndpoints[0].GetEndpoint().Address.GetSocketAddress().GetPortValue()
	return address + ":" + strconv.Itoa(int(port)), nil
}

// AddCluster adds a cluster to the map
func (d *Dataplane) AddCluster(cluster *cluster.Cluster) {
	d.clusters[cluster.Name] = cluster
}

// AddListener adds a listener to the map
func (d *Dataplane) AddListener(listener *listener.Listener) {
	listenerName := strings.TrimPrefix(listener.Name, api.ImportListenerPrefix)
	if _, ok := d.listeners[listenerName]; ok {
		return
	}
	d.listeners[listenerName] = listener
	go func() {
		d.CreateListener(listenerName, listener.Address.GetSocketAddress().GetAddress(), listener.Address.GetSocketAddress().GetPortValue())
	}()
}

// NewDataplane returns a new dataplane HTTP server.
func NewDataplane(dataplaneID, controlplaneTarget, peerName string, parsedCertData *util.ParsedCertData) *Dataplane {
	d := &Dataplane{
		ID:       dataplaneID,
		peerName: peerName,
		router:   chi.NewRouter(),
		apiClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: parsedCertData.ClientConfig(peerName),
			},
		},
		parsedCertData:     parsedCertData,
		controlplaneTarget: controlplaneTarget,
		clusters:           make(map[string]*cluster.Cluster),
		listeners:          make(map[string]*listener.Listener),
		listenerEnd:        make(map[string]chan bool),
		logger:             logrus.WithField("component", "dataplane.server.http"),
	}

	d.addAuthzHandlers()
	return d
}
