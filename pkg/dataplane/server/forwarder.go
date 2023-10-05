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
	workloadConn net.Conn
	peerConn     net.Conn
	close        bool
	logger       *logrus.Entry
}

type connDialer struct {
	c net.Conn
}

func (cd connDialer) Dial(_, _ string) (net.Conn, error) {
	return cd.c, nil
}

func (f *forwarder) peerToWorkload() error {
	bufData := make([]byte, dataBufferSize)
	for {
		numBytes, err := f.peerConn.Read(bufData)
		if err != nil {
			if err != io.EOF { // don't log EOF
				return err
			}
			break
		}
		_, err = f.workloadConn.Write(bufData[:numBytes]) // TODO: track actually written byte count
		if err != nil {
			if err != io.EOF { // don't log EOF
				return err
			}
			break
		}
	}
	f.closeConnections()
	return nil
}

func (f *forwarder) workloadToPeer() error {
	bufData := make([]byte, dataBufferSize)
	for {
		numBytes, err := f.workloadConn.Read(bufData)
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
	f.closeConnections()
	return nil
}

func (f *forwarder) closeConnections() {
	if f.peerConn != nil {
		f.peerConn.Close()
	}
	if f.workloadConn != nil {
		f.workloadConn.Close()
	}
}

func (f *forwarder) run() {
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := f.workloadToPeer()
		if err != nil {
			f.logger.Errorf("End of workload to peer connection %v.", err)
		}
	}()

	wg.Add(1)
	go func() {
		wg.Done()
		err := f.peerToWorkload()
		if err != nil {
			f.logger.Errorf("End of peer to workload connection %v.", err)
		}
	}()

	wg.Wait()
}

func newForwarder(workloadConn net.Conn, peerConn net.Conn) *forwarder {
	return &forwarder{workloadConn: workloadConn,
		peerConn: peerConn,
		logger:   logrus.WithField("component", "dataplane.forwarder"),
	}
}
