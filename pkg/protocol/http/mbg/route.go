package handler

import (
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/go-chi/chi"
)

type MbgHandler struct{}

func (m MbgHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", m.mbgWelcome())

	r.Route("/hello", func(r chi.Router) {
		//r.Use(PostCtx)
		r.Get("/", m.helloGet)   //  GET  /hello - Get MBG id
		r.Post("/", m.helloPost) // Post /hello  - Post MBG Id
	})

	r.Route("/expose", func(r chi.Router) {
		r.Post("/", m.exposePost()) // Post /expose  - Expose mbg service
	})
	return r
}

func (m MbgHandler) mbgWelcome() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("Welcome to Multi-cloud Boarder Gateway"))
		if err != nil {
			log.Println(err)
		}
	}
}
