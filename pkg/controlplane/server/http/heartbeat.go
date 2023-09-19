package http

import (
	"net/http"

	"github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
)

func (s *Server) addHeartbeatHandler() {
	r := s.Router()

	r.Post(api.HeartbeatPath, s.Heartbeat)
}

// Heartbeat returns a response for heartbeat checks from remote peers.
func (s *Server) Heartbeat(w http.ResponseWriter, _ *http.Request) {
	// Response
	w.WriteHeader(http.StatusOK)
}
