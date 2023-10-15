// Copyright 2023 The ClusterLink Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
This file is for temporary testing purpose. TODO : Remove this in future
*/
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

var (
	ctype   = flag.String("type", "server", "server/client")
	port    = flag.String("port", "9999", "TCP port to listen for flows")
	message = flag.String("message", "test message", "message")
)

var (
	maxDataBufferSize = 64 * 1024
)

func testrecvServiceData(clusterIn string) {
	acceptor, err := net.Listen("tcp", clusterIn)
	if err != nil {
		return
	}
	fmt.Printf("Waiting for connection at %s \n", clusterIn)
	// loop until signalled to stop
	for {
		ac, _ := acceptor.Accept()
		fmt.Printf("Accept connection %s->%s \n", ac.RemoteAddr().String(), ac.LocalAddr().String())
		go recvServiceData(ac, true)
	}
}

func testsendServiceData(clusterIn string, data []byte) {
	nodeConn, err := net.Dial("tcp", clusterIn)
	if err != nil {
		log.Fatalf("Failed to connect to socket %+v", err)
	}
	fmt.Printf("Connected to %s:%s \n", nodeConn.LocalAddr().String(), nodeConn.RemoteAddr().String())
	go recvServiceData(nodeConn, false)
	for {
		nbytes, err := nodeConn.Write(data)
		if err != nil || nbytes != len(data) {
			if err == io.EOF {
				break
			}
			fmt.Printf("failed to write data: %+v (%d bytes written)", err, nbytes)
		}
		time.Sleep(1 * time.Second)
	}
}

func recvServiceData(conn net.Conn, write bool) {
	bufData := make([]byte, maxDataBufferSize)
	for {
		numBytes, err := conn.Read(bufData)
		if err != nil {
			if err != io.EOF {
				log.Fatalf("Read error %v\n", err)
			}
			break
		}
		log.Printf("Received \"%s\" in Socket Connection", bufData[:numBytes])
		if write {
			_, err = conn.Write([]byte("Success from server"))
			if err != nil {
				if err != io.EOF {
					log.Fatalf("Read error %v\n", err)
				}
			}
		}
	}
}

func main() {
	flag.Parse()

	switch *ctype {
	case "server":
		testrecvServiceData(":" + *port)
	case "client":
		testsendServiceData("127.0.0.1:"+*port, []byte(*message))
	default:
		fmt.Printf("Wrong Type!")
	}
}
