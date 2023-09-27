package server

import (
	"io"
	"net"
	"sync"

	"github.com/sirupsen/logrus"
)

const (
	maxDataBufferSize = 64 * 1024
)

type forwarder struct {
	listenerConn net.Conn
	peerConn     net.Conn
	logger       *logrus.Entry
}

type connDialer struct {
	c net.Conn
}

func (cd connDialer) Dial(_, _ string) (net.Conn, error) {
	return cd.c, nil
}

func (f *forwarder) peerToListener() error {
	bufData := make([]byte, maxDataBufferSize)
	for {
		numBytes, err := f.peerConn.Read(bufData)
		if err != nil {
			if err != io.EOF { // don't log EOF
				f.logger.Infof("peerToListener: Read error %v\n", err)
				return err
			}
			break
		}
		_, err = f.listenerConn.Write(bufData[:numBytes]) // TODO: track actually written byte count
		if err != nil {
			if err != io.EOF { // don't log EOF
				f.logger.Infof("peerToListener: Write error %v\n", err)
				return err
			}
			break
		}
	}
	f.closeConnections()
	return nil
}

func (f *forwarder) listenerToPeer() error {
	bufData := make([]byte, maxDataBufferSize)
	for {
		numBytes, err := f.listenerConn.Read(bufData)
		if err != nil {
			if err != io.EOF { // don't log EOF
				f.logger.Infof("listenerToPeer: Read error %v\n", err)
				return err
			}
			break
		}

		_, err = f.peerConn.Write(bufData[:numBytes]) // TODO: track actually written byte count
		if err != nil {
			if err != io.EOF { // don't log EOF
				f.logger.Infof("listenerToPeer: Write error %v\n", err)
				return err
			}
			break
		}
	}
	f.closeConnections()
	return nil
}

func (f *forwarder) closeConnections() {
	if f.peerConn != nil {
		f.peerConn.Close()
	}
	if f.listenerConn != nil {
		f.listenerConn.Close()
	}
}

func (f *forwarder) start() {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := f.listenerToPeer()
		if err != nil {
			f.logger.Error("End of listener to peer connection ", err)
		}
	}()

	wg.Add(1)
	go func() {
		wg.Done()
		err := f.peerToListener()
		if err != nil {
			f.logger.Error("End of peer to listerner connection ", err)
		}
	}()

	wg.Wait()
}
