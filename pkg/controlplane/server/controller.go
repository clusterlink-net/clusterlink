package server

import (
	"errors"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

// Server represents a controllable server.
type Server interface {
	Name() string
	Listen(address string) error
	Serve() error
	Stop() error
	GracefulStop() error
	Close() error
}

// Controller controls a set of servers.
type Controller struct {
	lock sync.Mutex

	// servers maps listen address to server.
	servers map[string]Server
	errors  map[Server]error

	logger *logrus.Entry
}

// Add adds a new server.
func (c *Controller) Add(listenAddress string, server Server) {
	c.servers[listenAddress] = server
}

// Run starts all servers.
func (c *Controller) Run() error {
	// start server listeners
	for listenAddress, server := range c.servers {
		if err := server.Listen(listenAddress); err != nil {
			return fmt.Errorf("unable to create listener for server '%s' on %s: %v",
				server.Name(), listenAddress, err)
		}
	}

	lock := &sync.Mutex{}
	stop := sync.NewCond(lock)

	// goroutine for stopping all servers if one fails
	go func(stop *sync.Cond) {
		stop.L.Lock()
		stop.Wait()
		stop.L.Unlock()

		c.lock.Lock()
		pending := len(c.errors) < len(c.servers)
		c.lock.Unlock()

		if pending {
			if err := c.Stop(); err != nil {
				c.logger.Warnf("Error stopping: %v.", err)
			} else {
				c.logger.Infof("Asked all servers to stop.")
			}
		}
	}(stop)

	// initialize wait group
	wg := &sync.WaitGroup{}
	wg.Add(len(c.servers))

	// start servers in goroutines
	for _, server := range c.servers {
		go func(srv Server) {
			defer wg.Done()

			c.logger.Infof("Starting server '%s'.", srv.Name())
			err := srv.Serve()
			c.logger.Infof("Server '%s' stopped: %v.", srv.Name(), err)

			c.lock.Lock()
			c.errors[srv] = err
			c.lock.Unlock()

			if err != nil {
				// signal to stop other servers
				stop.Signal()
			}
		}(server)
	}

	// wait for all servers to stop
	wg.Wait()

	// terminate error-waiting goroutine
	stop.Signal()

	// close all servers
	for _, server := range c.servers {
		if err := server.Close(); err != nil {
			c.logger.Warnf("Error closing server '%s': %v.", server.Name(), err)
		}
	}

	// collect and return errors
	var errs []error
	for server, err := range c.errors {
		if err != nil {
			errs = append(errs, fmt.Errorf(
				"error running server '%s': %v", server.Name(), err))
		}
	}
	return errors.Join(errs...)
}

// Stop stops all servers.
func (c *Controller) Stop() error {
	c.logger.Info("Stopping.")

	var errs []error
	for _, server := range c.servers {
		if err := server.Stop(); err != nil {
			errs = append(errs, fmt.Errorf(
				"unable to stop server '%s': %v", server.Name(), err))
		}
	}

	return errors.Join(errs...)
}

// GracefulStop gracefully stops all servers.
func (c *Controller) GracefulStop() error {
	c.logger.Info("Gracefully stopping.")

	var errs []error
	for _, server := range c.servers {
		if err := server.GracefulStop(); err != nil {
			errs = append(errs, fmt.Errorf(
				"unable to gracefully stop server '%s': %v", server.Name(), err))
		}
	}

	return errors.Join(errs...)
}

// NewController returns a new empty server controller.
func NewController() *Controller {
	return &Controller{
		servers: make(map[string]Server),
		errors:  make(map[Server]error),
		logger:  logrus.WithField("component", "server-controller"),
	}
}
