// Copyright (c) The ClusterLink Authors.
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

package http

import (
	"context"
	"crypto/tls"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/sirupsen/logrus"

	"github.com/clusterlink-net/clusterlink/pkg/util/tcp"
)

// Server is a wrapper of an HTTP server.
type Server struct {
	tcp.Listener

	name   string
	router chi.Router
	server *http.Server

	logger    *logrus.Entry
	logWriter *io.PipeWriter
}

// Router returns the server (chi-)router.
func (s *Server) Router() chi.Router {
	return s.router
}

// Start the server.
func (s *Server) Start() error {
	defer func() {
		s.server.ErrorLog = nil
		if err := s.logWriter.Close(); err != nil {
			s.logger.Warnf("unable to close http server log writer: %v", err)
		}
	}()

	err := s.server.ServeTLS(s.GetListener(), "", "")
	if err == http.ErrServerClosed {
		s.logger.Info("Server closed by demand.")
		return nil
	}
	return err
}

// Stop the server.
func (s *Server) Stop() error {
	return s.server.Close()
}

// GracefulStop does a graceful stop of the server.
func (s *Server) GracefulStop() error {
	return s.server.Shutdown(context.Background())
}

// NewServer returns a new server.
func NewServer(name string, tlsConfig *tls.Config) *Server {
	logger := logrus.WithFields(logrus.Fields{
		"component": "http-server",
		"name":      name,
	})
	logWriter := logger.WriterLevel(logrus.ErrorLevel)

	router := chi.NewRouter()
	if logrus.GetLevel() >= logrus.DebugLevel {
		router.Use(middleware.RequestLogger(&middleware.DefaultLogFormatter{
			Logger:  logger,
			NoColor: true,
		}))
	}
	router.Use(middleware.Recoverer)

	return &Server{
		Listener: tcp.NewListener(name),
		name:     name,
		router:   router,
		server: &http.Server{
			Handler:           router,
			TLSConfig:         tlsConfig,
			ErrorLog:          log.New(logWriter, "", 0),
			ReadHeaderTimeout: time.Second,
		},
		logger:    logger,
		logWriter: logWriter,
	}
}
