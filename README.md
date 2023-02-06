# Multi-cloud Border Gateway(MBG) project
Through the Multi-cloud border gateway, users can simplify the connection between various application services that are located in different domains, networks, and cloud infrastructures. 
For more details, see the document: TBD 
This project contains two main components: 
1) MBG - the Multi-cloud border gateway that allows secure connections between different services in different network domains.
   The MBG has different APIs like hello, expose and connect, enabling service connectivity.
   The MBG can also apply some network policies (ACL, load-balancer etc.)
2) mbgctl - the mbgctl is CLI implementation that uses MBG APIs to send control messages to thr MBG.
   The mbgctl uses commands like expose, connect and disconnect to create connectivity to service in different network domains using the MBG. 

![alt text](./docs/mbg-proto.png)


## <ins>How to setup and run the MBG<ins>
The MBG can be set up and run on different environments: local environment (Kind), Bare-metal environment, or cloud environment.
### <ins> Run MBG in local environment (Kind) <ins>
MBG can run in any K8s environment, such as Kind.
To run the MBG in a Kind environment, follow one of the examples:
1) Performance example - Run iPerf3 test between iPerf3 client and server using MBG components. This example is used for performance measuring. Instructions can be found [Here](tests/iperf3/kind/README.md).
1) Application example - Run the BookInfo application in different clusters using MBG components. This example demonstrates communication distributed applications (in different clusters) with different policies.Instructions can be found [Here](tests/bookinfo/kind/README.md).

### <ins>Run MBG in Bare-metal environment with 2 hosts<ins> 
Follow instructions from [Here](tests/iperf3/bare-metal/commands.txt)

### <ins>Run MBG in cloud environment<ins> 
TBD
