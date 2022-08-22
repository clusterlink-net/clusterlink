package main

import (
	"flag"
	"fmt"
	"net"
	"os"
)

var (
	listener    = flag.String("listen", ":5001", "listen host:port for client")
	servicenode = flag.String("sn", "", "listen host:port of server side service node")
)

func main() {
	flag.Parse()

	if *listener == "" || *servicenode == "" {
		fmt.Println("missing listener or service")
		os.Exit(-1)
	}

	acceptLoop(*listener, *servicenode) // missing channel for signal handler
}

func acceptLoop(listener, servicenode string) error {
	// open listener
	// loop until signalled to stop
	// clientConn := ln.Accept()
	// if no error {
	//		go dispatch(clientConn, servicenode)
	// }
	return nil
}

func dispatch(c net.Conn, servicenode string) error {
	// nodeConn := dial(servicenode)
	// if ok { send setup frame; ioLoop(clientConn, nodeConn) }
	// else fail and return error
	return nil
}

func ioLoop(cl, sn net.Conn) error {
	defer cl.Close()
	defer sn.Close()
	// wg(2) + two go routines: go clientToServer, go serverToClient
	// wg.Wait(2)
	return nil
}

func clientToServer(cl, sn net.Conn) error {
	// allocate 64KB buffer
	// forever {
	// 	  numBytes = read from client to buffer
	//    create frame with data+numBytes
	//    send(sn, frame + buffer)
	// }
	return nil
}

func serverToClient(cl, sn net.Conn) error {
	// allocate 4B frame-buffer and 64KB payload buffer
	// forever {
	//    read 4B into frame-buffer
	//    if frame.Type == control { // not expected yet, except for error returns from SN
	// 	     read and process control frame
	//    } else {
	// 	 	 read(sn, payload, frame.Len) // might require multiple reads and need a timeout deadline set
	//	     send(cl, payload)
	//    }
	// }
	return nil
}
