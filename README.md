# Multi-cloud Border Gateway(MBG) project  
Through the Multi-cloud border gateway, users can simplify the connection between various application services that are located in different domains, networks, and cloud infrastructures. 
For more details, see the document: TBD 
This project contains two main components: 
1) MBG - THe Multi-cloud border gateway that allows secure connections between different services in different network domains.
   The MBG has different APIs like hello, expose and connect, enabling service connectivity.  
   The MBG can also apply some network functions (TCP-split, compression, etc.)
2) Cluster - is K8s cluster implementation that uses MBG APIs to connect the service inside the network domain to the MBG.
   The cluster uses commands like expose, connect and disconnect to create connectivity to service in different network domains using the MBG. 

![alt text](./tests/figures/mbg-proto.png)


## <ins>Run MBG in local environment (Kind)<ins>
In this test we setup 4 clusters that run in local kind clusters: 
1) Host cluster (iPrf3 client) 
2) MBG1 (the mbg connect to the host domain) 
3) MBG2 (the mbg connect to the destination domain) 
4) Destination cluster (iperf3 server)

The test check iPerf3 connectivity between the host and destination cluster
* Check all pre-requires  are installed (Go, docker, Kubectl, Kind ): ```make prereqs```
* Build the kind cluster and run the iPerf3 test:
      ```python3 tests/kind/Iperf3/iperf3Test.py```
## <ins>Run MBG in Bare-metal environment with 2 hosts<ins> 
Follow instructions from [Here](tests/bare-metal/commands.txt)

## <ins>Run MBG in cloud environment<ins> 
TBD
