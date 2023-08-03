package rest

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"reflect"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"

	"github.ibm.com/mbg-agent/pkg/store"
	utilhttp "github.ibm.com/mbg-agent/pkg/util/http"
)

// Server for handling REST-JSON requests.
type Server struct {
	utilhttp.Server

	logger *logrus.Entry
}

// Handler for object operations.
type Handler interface {
	// Decode and validate an object.
	Decode(data []byte) (any, error)
	// Create an object.
	Create(object any) error
	// Update an object.
	Update(object any) error
	// Get an object.
	Get(name string) (any, error)
	// Delete an object.
	Delete(object any) (any, error)
	// List all objects.
	List() (any, error)
}

// ServerObjectSpec specifies a set of server handlers for a specific object type.
type ServerObjectSpec struct {
	// BasePath is the server HTTP path for manipulating a specific type of objects.
	BasePath string
	// Handler interface for object operations.
	Handler Handler
	// DeleteByValue is true for object types which are deletable by sending their value, instead of their name.
	DeleteByValue bool
}

func (s *Server) create(spec *ServerObjectSpec, w http.ResponseWriter, r *http.Request) {
	requestLogger := s.logger.WithFields(logrus.Fields{"method": "create", "path": r.URL.Path})
	requestLogger.WithField("body-length", r.ContentLength).Infof("Handling request.")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		requestLogger.Errorf("Cannot read request body: %v.", err)
		return
	}

	requestLogger.Debugf("Body: %v.", body)

	object, err := spec.Handler.Decode(body)
	if err != nil {
		requestLogger.Errorf("Cannot decode object: %v.", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := spec.Handler.Create(object); err != nil {
		if _, ok := err.(*store.ObjectExistsError); ok {
			requestLogger.Errorf("Object already exists.")
			http.Error(w, "object already exists", http.StatusBadRequest)
			return
		}

		requestLogger.Errorf("Cannot create object: %v.", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Location", r.URL.String())
}

func (s *Server) update(spec *ServerObjectSpec, w http.ResponseWriter, r *http.Request) {
	requestLogger := s.logger.WithFields(logrus.Fields{"method": "update", "path": r.URL.Path})
	requestLogger.WithField("body-length", r.ContentLength).Infof("Handling request.")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		requestLogger.Errorf("Cannot read request body: %v.", err)
		return
	}

	requestLogger.Debugf("Body: %v.", body)

	object, err := spec.Handler.Decode(body)
	if err != nil {
		requestLogger.Errorf("Cannot decode object: %v.", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := spec.Handler.Update(object); err != nil {
		if _, ok := err.(*store.ObjectNotFoundError); ok {
			requestLogger.Errorf("Object not found.")
			http.Error(w, "object not found", http.StatusNotFound)
			return
		}

		requestLogger.Errorf("Cannot create object: %v.", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	w.Header().Set("Location", r.URL.String())
}

func (s *Server) get(spec *ServerObjectSpec, w http.ResponseWriter, r *http.Request) {
	requestLogger := s.logger.WithFields(logrus.Fields{"method": "get", "path": r.URL.Path})
	requestLogger.Infof("Handling request.")

	name := chi.URLParam(r, "name")

	result, err := spec.Handler.Get(name)
	if err != nil {
		requestLogger.Errorf("Cannot get object: %v.", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if result == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		requestLogger.Errorf("Cannot encode object: %v.", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if _, err := w.Write(encoded); err != nil {
		s.logger.Errorf("Cannot write http response: %v.", err)
	}
}

func (s *Server) deleteObject(spec *ServerObjectSpec, w http.ResponseWriter, r *http.Request) {
	requestLogger := s.logger.WithFields(logrus.Fields{"method": "deleteObject", "path": r.URL.Path})
	requestLogger.WithField("body-length", r.ContentLength).Infof("Handling request.")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		requestLogger.Errorf("Cannot read request body: %v.", err)
		return
	}

	requestLogger.Debugf("Body: %v.", body)

	object, err := spec.Handler.Decode(body)
	if err != nil {
		requestLogger.Errorf("Cannot decode object: %v.", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := spec.Handler.Delete(object)
	if err != nil {
		requestLogger.Errorf("Cannot delete object: %v.", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if reflect.ValueOf(result).IsNil() {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) delete(spec *ServerObjectSpec, w http.ResponseWriter, r *http.Request) {
	requestLogger := s.logger.WithFields(logrus.Fields{"method": "delete", "path": r.URL.Path})
	requestLogger.Infof("Handling request.")

	name := chi.URLParam(r, "name")

	result, err := spec.Handler.Delete(name)
	if err != nil {
		requestLogger.Errorf("Cannot delete object: %v.", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if reflect.ValueOf(result).IsNil() {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) list(spec *ServerObjectSpec, w http.ResponseWriter, r *http.Request) {
	requestLogger := s.logger.WithFields(logrus.Fields{"method": "list", "path": r.URL.Path})
	requestLogger.Infof("Handling request.")

	result, err := spec.Handler.List()
	if err != nil {
		requestLogger.Errorf("Cannot list objects: %v.", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		requestLogger.Errorf("Cannot encode objects: %v.", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if _, err := w.Write(encoded); err != nil {
		s.logger.Errorf("Cannot write http response: %v.", err)
	}
}

// AddObjectHandlers adds the server a handlers for managing a specific object type.
func (s *Server) AddObjectHandlers(spec *ServerObjectSpec) {
	r := s.Router()

	r.Route(spec.BasePath, func(r chi.Router) {
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			s.create(spec, w, r)
		})
		r.Put("/", func(w http.ResponseWriter, r *http.Request) {
			s.update(spec, w, r)
		})
		r.Get("/{name}", func(w http.ResponseWriter, r *http.Request) {
			s.get(spec, w, r)
		})
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			s.list(spec, w, r)
		})

		if spec.DeleteByValue {
			r.Delete("/", func(w http.ResponseWriter, r *http.Request) {
				s.deleteObject(spec, w, r)
			})
		} else {
			r.Delete("/{name}", func(w http.ResponseWriter, r *http.Request) {
				s.delete(spec, w, r)
			})
		}
	})
}

// NewServer returns a new empty REST-JSON server.
func NewServer(name string, tlsConfig *tls.Config) Server {
	return Server{
		Server: utilhttp.NewServer(name, tlsConfig),
		logger: logrus.WithFields(logrus.Fields{
			"component": "rest-server",
			"name":      name}),
	}
}
