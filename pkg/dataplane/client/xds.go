package client

import (
	"context"
	"crypto/tls"
	"errors"
	"sync"
	"time"

	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/clusterlink-net/clusterlink/pkg/dataplane/server"
)

// resources indicate the xDS resources that would be fetched
var resources = [...]string{resource.ClusterType, resource.ListenerType}

// XDSClient implements the client which fetches clusters and listeners
type XDSClient struct {
	dataplane          *server.Dataplane
	controlplaneTarget string
	tlsConfig          *tls.Config
	errors             []error
	logger             *logrus.Entry
}

func (x *XDSClient) runFetcher(resourceType string) error {
	for {
		time.Sleep(5 * time.Second)
		conn, err := grpc.Dial(x.controlplaneTarget, grpc.WithTransportCredentials(credentials.NewTLS(x.tlsConfig)))
		if err != nil {
			x.logger.Errorf("Failed to dial controlplane xDS server: %v.", err)
			continue
		}
		x.logger.Infof("Successfully connected to the controlplane xDS server.")

		f, err := newFetcher(context.Background(), conn, resourceType, x.dataplane)
		if err != nil {
			x.logger.Errorf("Failed to initialize fetcher: %v.", err)
			continue
		}
		x.logger.Infof("Successfully initialized client for %s type.", resourceType)
		err = f.Run()
		x.logger.Infof("Fetcher '%s' stopped: %v.", resourceType, err)
	}
}

// Run starts the running xDS client which fetches clusters and listeners from the controlplane.
func (x *XDSClient) Run() error {
	var wg sync.WaitGroup
	wg.Add(len(resources))
	for i, res := range resources {
		go func(i int, res string) {
			defer wg.Done()
			err := x.runFetcher(res)
			x.logger.Errorf("Fetcher (%s) stopped: %v", res, err)
			x.errors[i] = err
		}(i, res)
	}
	wg.Wait()
	return errors.Join(x.errors...)
}

// NewXDSClient returns am xDS client which can fetch clusters and listeners from the controlplane.
func NewXDSClient(dataplane *server.Dataplane, controlplaneTarget string, tlsConfig *tls.Config) *XDSClient {
	return &XDSClient{dataplane: dataplane,
		controlplaneTarget: controlplaneTarget,
		tlsConfig:          tlsConfig,
		logger:             logrus.WithField("component", "xds.client")}
}
