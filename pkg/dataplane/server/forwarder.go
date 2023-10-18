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

package server

import (
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	dataBufferSize = 64 * 1024
	readDeadline   = 10 * time.Millisecond
)

type forwarder struct {
	workloadConn net.Conn
	peerConn     net.Conn
	closeSignal  atomic.Bool
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
	var err error
	numBytes := 0
	for {
		if f.closeSignal.Load() {
			return nil
		}
		err = f.peerConn.SetReadDeadline(time.Now().Add(readDeadline))
		if err != nil {
			return err
		}
		numBytes, err = f.peerConn.Read(bufData)
		if err != nil {
			if err1, ok := err.(net.Error); ok && err1.Timeout() {
				continue
			}
			break
		}
		_, err = f.workloadConn.Write(bufData[:numBytes]) // TODO: track actually written byte count
		if err != nil {
			break
		}
	}
	f.closeSignal.Swap(true)
	if err != io.EOF { // don't log EOF
		return err
	}
	return nil
}

func (f *forwarder) workloadToPeer() error {
	bufData := make([]byte, dataBufferSize)
	var err error
	numBytes := 0
	for {
		if f.closeSignal.Load() {
			return nil
		}
		err = f.workloadConn.SetReadDeadline(time.Now().Add(readDeadline))
		if err != nil {
			return err
		}
		numBytes, err = f.workloadConn.Read(bufData)
		if err != nil {
			if err1, ok := err.(net.Error); ok && err1.Timeout() {
				continue
			}
			break
		}
		_, err = f.peerConn.Write(bufData[:numBytes]) // TODO: track actually written byte count
		if err != nil {
			break
		}
	}
	f.closeSignal.Swap(true)
	if err != io.EOF { // don't log EOF
		return err
	}
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
			f.logger.Errorf("Error in workload to peer connection: %v.", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := f.peerToWorkload()
		if err != nil {
			f.logger.Errorf("Error in peer to workload connection: %v.", err)
		}
	}()

	wg.Wait()
	f.closeConnections()
}

func newForwarder(workloadConn net.Conn, peerConn net.Conn) *forwarder {
	return &forwarder{workloadConn: workloadConn,
		peerConn: peerConn,
		logger:   logrus.WithField("component", "dataplane.forwarder"),
	}
}
