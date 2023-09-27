# Controlplane & dataplane interactions

This Document lays out the interaction between control plane and data plane for establishing connectivity

## Data plane configuration

The Data plane relies on both xDS (Discovery services) and HTTP to communicate with the control plane. xDS (gRPC-based) is used for top-down information dispersal from control plane to data plane, while HTTP is used for connection authorization requests from the data plane to the control plane.

### The role of xDS client:
1) Fetches clusters & listeners from the control plane, and stores their information.
2) A cluster message contains information about peers (targets to reach), and exported services (address:port). The cluster name is prefixed with "remote-peer-" in the case of peers and "export-" in the case of exported service.
3) A listener message contains information about an imported service (name and listening port)

## Scenario - Establishing connection between applications in two clusters
Assume clusterlink deployed in two clusters. Peer1 and Peer2 are the clusterlink instances running in cluster1 and cluster2 respectively.
A clusterlink instance peer1 consists of two components peer1-controlplane and peer1-dataplane.
Peer1 exports a service S and Peer2 imports the service and connects to it

### The steps in establishing connection:

1) An export of a service by peer1 is propagated to the peer1-dataplane as a cluster via xDS. In this case the name of the cluster is "export-s".
2) Upon import of the service by peer2, the peer2-controlplane processes the import and sends the listener config to peer2-dataplane (via xDS) which then sets up listener to listen on the specified port.
3) When a connection is received at the listener of peer2-dataplane, it needs to authorize this connection. For this purpose, the dataplane sends a HTTP request to peer2-controlplane with the connecting client working IP address and imported service name. This information is embedded as headers ("x-forwarded-for" set to the client IP and "x-import" as name of the imported service) in the HTTP request sent to egressAuthorization path (/auth/egress)
4) The peer2-controlplane now sends an authorization requests to the peer1-controlplane (via peer1-dataplane ingressAuthorization HTTP request)
5) The peer1-controlplane returns a JWT authorization with the cluster name (i.e exported service name) in the response header ("authorization") if its allowed.
6) The peer2-controlplane receives this token and replies this token back to peer2-dataplane (egressAuthorization) in the response header
7) The peer2-dataplane sends a HTTP Post request with the JWT embedded in the request header ("authorization") to peer1-dataplane:443.
8) The peer1-dataplane passes the token to peer1-controlplane which parses the JWT (by sending the auth token to controlplane) to know the "cluster" to redirect the message to and sends the cluster destination (embedded in the header) to peer1-dataplane.
9) Peer1-dataplane hijacks the connection and establishes the lastmile connection with exported service using the "cluster" information.
10) For further messages the channel is now formed between the applications.
