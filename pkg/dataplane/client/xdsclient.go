package client

import (
	"context"
	"strings"
	"sync"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	client "github.com/envoyproxy/go-control-plane/pkg/client/sotw/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/clusterlink-org/clusterlink/pkg/controlplane/api"
	"github.com/clusterlink-org/clusterlink/pkg/dataplane/server"
)

func runClusterFetcher(clusterFetcher client.ADSClient) error {
	for {
		resp, err := clusterFetcher.Fetch()
		if err != nil {
			log.Error("Failed to fetch cluster", err)
			continue
		}
		for _, r := range resp.Resources {
			myCluster := &cluster.Cluster{}
			err := anypb.UnmarshalTo(r, myCluster, proto.UnmarshalOptions{})
			if err != nil {
				log.Error("Failed to unmarshal cluster resource : ", err)
				return err
			}
			log.Infof("Cluster : %s", myCluster.Name)
			server.AddCluster(myCluster)
		}
	}
}

func runListenerFetcher(listenerFetcher client.ADSClient, dataplane *server.Dataplane) error {
	for {
		resp, err := listenerFetcher.Fetch()
		if err != nil {
			log.Error("Failed to fetch listener", err)
			continue
		}
		for _, r := range resp.Resources {
			myListener := &listener.Listener{}
			err := anypb.UnmarshalTo(r, myListener, proto.UnmarshalOptions{})
			if err != nil {
				log.Error("Failed to unmarshal listener resource : ", err)
				return err
			}
			log.Infof("Listener : %s", myListener.Name)
			listenerName := strings.TrimPrefix(myListener.Name, api.ImportListenerPrefix)
			err = server.AddListener(listenerName, myListener)
			if err != nil {
				continue
			}
			go func() {
				dataplane.CreateListenerToImportServiceEndpoint(listenerName, myListener.Address.GetSocketAddress().GetAddress(), myListener.Address.GetSocketAddress().GetPortValue())
			}()
		}
	}
}

// StartxDSClient starts the xDS client which fetches to clusters & listeners from controlplane
func StartxDSClient(dataplane *server.Dataplane, controlplaneTarget string, cred credentials.TransportCredentials) error {
	var wg sync.WaitGroup
	conn, err := grpc.Dial(controlplaneTarget, grpc.WithTransportCredentials(cred))
	if err != nil {
		return err
	}
	log.Infof("Successfully connected to the controlplane")

	c := client.NewADSClient(context.Background(), &core.Node{Id: dataplane.ID}, resource.ClusterType)
	err = c.InitConnect(conn)
	if err != nil {
		log.Error("Failed to init connect(cluster) : ", err)
		return err
	}
	log.Infof("Successfully initialized client for cluster ")

	l := client.NewADSClient(context.Background(), &core.Node{Id: dataplane.ID}, resource.ListenerType)
	err = l.InitConnect(conn)
	if err != nil {
		log.Error("Failed to init connect(listener) : ", err)
		return err
	}
	log.Infof("Successfully initialized client for listener")
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = runClusterFetcher(c)
		log.Errorf("failed to run cluster fetcher: %+v", err)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = runListenerFetcher(l, dataplane)
		log.Errorf("failed to run listener fetcher: %+v", err)
	}()
	wg.Wait()
	return nil
}
