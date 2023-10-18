# Dataplane metrics

This document lays out the metrics available in the envoy dataplane and explains the design of the metrics module of the go dataplane.
Envoy dataplane stats (as explained [here](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/observability/statistics))
provide upstream, downstream and the server statistics. While upstream and downstream stats are related to connections/requests, server stats are related to the dataplane process health like CPU and memory utilization.

The envoy statistics are reported as counters, gauges and histograms. Counter and gauges are batched and reported periodically, while histogram is reported as they are received.

## Upstream stats
[Reference](https://www.envoyproxy.io/docs/envoy/latest/configuration/upstream/cluster_manager/cluster_stats)
example statistics: upstream_cx_total, upstream_cx_active, upstream_cx_connect_fail, upstream_cx_rx_bytes_total, upstream_cx_tx_bytes_total, etc

## Downstream stats
[Reference](https://www.envoyproxy.io/docs/envoy/latest/configuration/listeners/stats)
example statistics: downstream_cx_total, downstream_cx_active, 
