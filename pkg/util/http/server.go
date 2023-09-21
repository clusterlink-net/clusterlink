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

	"github.com/clusterlink-org/clusterlink/pkg/util/tcp"
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

// Serve starts the server.
func (s *Server) Serve() error {
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
func NewServer(name string, tlsConfig *tls.Config) Server {
	logger := logrus.WithFields(logrus.Fields{
		"component": "http-server",
		"name":      name})
	logWriter := logger.WriterLevel(logrus.ErrorLevel)

	router := chi.NewRouter()
	if logrus.GetLevel() >= logrus.DebugLevel {
		router.Use(middleware.RequestLogger(&middleware.DefaultLogFormatter{
			Logger:  logger,
			NoColor: true,
		}))
	}
	router.Use(middleware.Recoverer)

	return Server{
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
