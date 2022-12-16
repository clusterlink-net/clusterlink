package main

import (
	"io"
	"log"
	"net"
	"os"
	"time"

	md "github.ibm.com/mbg-agent/pkg/mbgDataplane"
)

var (
	maxDataBufferSize = 64 * 1024
)

func testsendServiceData(clusterIn string, data []byte) {
	nodeConn, err := net.Dial("tcp", clusterIn)
	if err != nil {
		log.Fatalf("Failed to connect to socket %+v", err)
	}
	go testrecvServiceData(nodeConn)
	for {
		nodeConn.Write(data)
		time.Sleep(1 * time.Second)
	}
}

func testrecvServiceData(conn net.Conn) {
	bufData := make([]byte, maxDataBufferSize)
	for {
		numBytes, err := conn.Read(bufData)
		if err != nil {
			if err == io.EOF {
				err = nil //Ignore EOF error
			} else {
				log.Fatalf("Read error %v\n", err)
			}
			break
		}
		log.Printf("Received %s in Socket Connection", bufData[:numBytes])
	}
}

func tcnode6() {
	go md.StartMtlsServer("10.20.20.1", "/home/pravein/mtls/tcnode6_cert.pem", "/home/pravein/mtls/tcnode6_key.pem")
	go md.StartClusterService("testService1", ":9000", "https://10.20.20.2:8443/mbgData", "/home/pravein/mtls/tcnode7_cert.pem", "/home/pravein/mtls/tcnode7_key.pem")

	time.Sleep(1 * time.Second)
	testsendServiceData(":9000", []byte("I am tcnode6-test1 cluster"))
}

//run in tcnode7
func tcnode7() {
	go md.StartMtlsServer("10.20.20.2", "/home/pravein/mtls/tcnode7_cert.pem", "/home/pravein/mtls/tcnode7_key.pem")
	go md.StartClusterService("testService2", ":9000", "https://10.20.20.1:8443/mbgData", "/home/pravein/mtls/tcnode6_cert.pem", "/home/pravein/mtls/tcnode6_key.pem")

	time.Sleep(1 * time.Second)
	testsendServiceData(":9000", []byte("I am tcnode7-test1 cluster"))
}

// This example connects a testService1 and testService2 running in different nodes using the mtls_forwader
func main() {
	host, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	if host == "tcnode6" {
		tcnode6()
	} else {
		tcnode7()
	}
}
