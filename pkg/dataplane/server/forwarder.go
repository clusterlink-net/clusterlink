package server

import (
	"io"
	"net"
	"sync"

	"github.com/sirupsen/logrus"
)

const (
	dataBufferSize = 64 * 1024
)

type forwarder struct {
	appConn  net.Conn
	peerConn net.Conn
	logger   *logrus.Entry
}

type connDialer struct {
	c net.Conn
}

func (cd connDialer) Dial(_, _ string) (net.Conn, error) {
	return cd.c, nil
}

func (f *forwarder) peerToApp() error {
	bufData := make([]byte, dataBufferSize)
	for {
		numBytes, err := f.peerConn.Read(bufData)
		if err != nil {
			if err != io.EOF { // don't log EOF
				return err
			}
			break
		}
		_, err = f.appConn.Write(bufData[:numBytes]) // TODO: track actually written byte count
		if err != nil {
			if err != io.EOF { // don't log EOF
				return err
			}
			break
		}
	}
	return nil
}

func (f *forwarder) appToPeer() error {
	bufData := make([]byte, dataBufferSize)
	for {
		numBytes, err := f.appConn.Read(bufData)
		if err != nil {
			if err != io.EOF { // don't log EOF
				return err
			}
			break
		}

		_, err = f.peerConn.Write(bufData[:numBytes]) // TODO: track actually written byte count
		if err != nil {
			if err != io.EOF { // don't log EOF
				return err
			}
			break
		}
	}
	return nil
}

func (f *forwarder) closeConnections() {
	if f.peerConn != nil {
		f.peerConn.Close()
	}
	if f.appConn != nil {
		f.appConn.Close()
	}
}

func (f *forwarder) run() {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := f.appToPeer()
		if err != nil {
			f.logger.Errorf("End of app to peer connection %v.", err)
		}
	}()

	wg.Add(1)
	go func() {
		wg.Done()
		err := f.peerToApp()
		if err != nil {
			f.logger.Errorf("End of peer to app connection %v.", err)
		}
	}()

	wg.Wait()
	f.closeConnections()
}

func newForwarder(appConn net.Conn, peerConn net.Conn) *forwarder {
	return &forwarder{appConn: appConn,
		peerConn: peerConn,
		logger:   logrus.WithField("component", "dataplane.forwarder"),
	}
}
