package handler

import (
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/go-chi/chi"
)

type ClusterHandler struct{}

func (c ClusterHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", c.clusterWelcome())

	r.Route("/expose", func(r chi.Router) {
		r.Post("/", c.exposePost()) // Post /expose  - Expose cluster service
	})
	return r
}

func (m ClusterHandler) clusterWelcome() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("Welcome to agent control for MBG"))
		if err != nil {
			log.Println(err)
		}
	}
}
