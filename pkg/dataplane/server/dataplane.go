package server

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
	"github.com/clusterlink-org/clusterlink/pkg/dataplane/api"
	"github.com/clusterlink-org/clusterlink/pkg/util"
)

// Server implementing the management API, allowing to manage the set of peers, imports, exports and bindings.
// Furthermore, this server implements the various authorization APIs.
type Dataplane struct {
	Id                 string
	peerName           string
	router             *chi.Mux
	apiClient          *http.Client
	authClient         *http.Client
	parsedCertData     *util.ParsedCertData
	controlplaneTarget string
	logger             *logrus.Entry
}

var (
	clusterMap   map[string]*cluster.Cluster
	listenerMap  map[string]*listener.Listener
	listenerChan map[string]chan bool
)

// GetTargetCluster returns the cluster address:port from the cluster map
func GetClusterTarget(name string) (string, error) {
	if _, ok := clusterMap[name]; !ok {
		return "", fmt.Errorf("unable to find %s in clustermap ", name)
	}
	address := clusterMap[name].LoadAssignment.GetEndpoints()[0].LbEndpoints[0].GetEndpoint().Address.GetSocketAddress().GetAddress()
	port := clusterMap[name].LoadAssignment.GetEndpoints()[0].LbEndpoints[0].GetEndpoint().Address.GetSocketAddress().GetPortValue()
	return address + ":" + strconv.Itoa(int(port)), nil
}

func AddCluster(cluster *cluster.Cluster) {
	clusterMap[cluster.Name] = cluster
}

func AddListener(listenerName string, listener *listener.Listener) error {
	if _, ok := listenerMap[listenerName]; ok {
		return fmt.Errorf("listener %s already exists", listenerName)
	}
	listenerMap[listenerName] = listener
	return nil
}

// NewDataplane returns a new dataplane HTTP server.
func NewDataplane(dataplaneID, controlplaneTarget, peerName string, parsedCertData *util.ParsedCertData) *Dataplane {
	d := &Dataplane{
		Id:       dataplaneID,
		peerName: peerName,
		router:   chi.NewRouter(),
		apiClient: &http.Client{
			Timeout: 10 & time.Second,
			Transport: &http.Transport{
				TLSClientConfig: parsedCertData.ClientConfig(peerName),
			},
		},
		authClient: &http.Client{
			Timeout: 10 & time.Second,
			Transport: &http.Transport{
				TLSClientConfig: parsedCertData.ClientConfig(api.DataplaneServerName(peerName)),
			},
		},
		parsedCertData:     parsedCertData,
		controlplaneTarget: controlplaneTarget,
		logger:             logrus.WithField("component", "dataplane.server.http"),
	}
	clusterMap = make(map[string]*cluster.Cluster)
	listenerMap = make(map[string]*listener.Listener)
	listenerChan = make(map[string]chan bool)
	d.addAuthzHandlers()
	return d
}
