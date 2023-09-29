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

// XDSClient implements the client which fetches clusters and listeners
type XDSClient struct {
	dataplane          *server.Dataplane
	controlplaneTarget string
	tlsConfig          *tls.Config
	fetchers           []*fetcher
	errors             []error
	logger             *logrus.Entry
}

func (x *XDSClient) runFetcher(_ *fetcher, resourceType string) error {
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
		err = f.Run()
		x.logger.Infof("Fetcher '%s' stopped: %v.", resourceType, err)
	}
}

// Run starts the running xDS client which fetches clusters and listeners from the controlplane.
func (x *XDSClient) Run() error {
	var wg sync.WaitGroup
	wg.Add(len(x.fetchers))
	go func() {
		defer wg.Done()
		err := x.runFetcher(x.fetchers[0], resource.ClusterType)
		x.logger.Errorf("Fetcher (cluster) stopped: %v", err)
		x.errors[0] = err
	}()
	go func() {
		defer wg.Done()
		err := x.runFetcher(x.fetchers[1], resource.ListenerType)
		x.logger.Errorf("Fetcher (listener) stopped: %v", err)
		x.errors[1] = err
	}()
	wg.Wait()
	return errors.Join(x.errors...)
}

// NewXDSClient returns am xDS client which can fetch clusters and listeners from the controlplane.
func NewXDSClient(dataplane *server.Dataplane, controlplaneTarget string, tlsConfig *tls.Config) *XDSClient {
	return &XDSClient{dataplane: dataplane,
		controlplaneTarget: controlplaneTarget,
		tlsConfig:          tlsConfig,
		fetchers:           make([]*fetcher, 2),
		logger:             logrus.WithField("component", "xds.client")}
}
