package client

import (
	"context"
	"fmt"
	"strings"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	client "github.com/envoyproxy/go-control-plane/pkg/client/sotw/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
	"github.com/clusterlink-net/clusterlink/pkg/dataplane/server"
)

type fetcher struct {
	client       client.ADSClient
	resourceType string
	dataplane    *server.Dataplane
	logger       *logrus.Entry
}

func (f *fetcher) handleClusters(resources []*anypb.Any) error {
	for _, r := range resources {
		var c cluster.Cluster
		err := anypb.UnmarshalTo(r, &c, proto.UnmarshalOptions{})
		if err != nil {
			return err
		}

		f.logger.Debugf("Cluster : %s.", c.Name)
		f.dataplane.AddCluster(&c)
	}
	return nil
}

func (f *fetcher) handleListeners(resources []*anypb.Any) error {
	for _, r := range resources {
		l := &listener.Listener{}
		err := anypb.UnmarshalTo(r, l, proto.UnmarshalOptions{})
		if err != nil {
			return err
		}
		f.logger.Debugf("Listener : %s.", l.Name)
		listenerName := strings.TrimPrefix(l.Name, api.ImportListenerPrefix)
		err = f.dataplane.AddListener(listenerName, l)
		if err != nil {
			continue
		}
		go func() {
			f.dataplane.CreateListener(listenerName, l.Address.GetSocketAddress().GetAddress(), l.Address.GetSocketAddress().GetPortValue())
		}()
	}
	return nil
}

func (f *fetcher) Run() error {
	for {
		resp, err := f.client.Fetch()
		if err != nil {
			f.logger.Errorf("Failed to fetch %s: %v.", f.resourceType, err)
			return err
		}
		switch f.resourceType {
		case resource.ClusterType:
			err := f.handleClusters(resp.Resources)
			if err != nil {
				f.logger.Errorf("Failed to handle clusters: %v.", err)
			}
		case resource.ListenerType:
			err := f.handleListeners(resp.Resources)
			if err != nil {
				f.logger.Errorf("Failed to handle listeners: %v.", err)
			}
		default:
			return fmt.Errorf("unknown resource type")
		}

		err = f.client.Ack()
		if err != nil {
			f.logger.Errorf("failed to ack: %v.", err)
		}
	}
}

func newFetcher(ctx context.Context, conn *grpc.ClientConn, resourceType string, dataplane *server.Dataplane) (*fetcher, error) {
	client := client.NewADSClient(ctx, &core.Node{Id: dataplane.ID}, resourceType)
	err := client.InitConnect(conn)
	if err != nil {
		return nil, err
	}
	logrus.Infof("Successfully initialized client for %s type.", resourceType)
	return &fetcher{client: client,
		resourceType: resourceType,
		dataplane:    dataplane,
		logger:       logrus.WithField("component", "fetcher.xds.client")}, nil
}
