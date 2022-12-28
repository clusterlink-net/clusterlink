package handler

import (
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/go-chi/chi"
)

type MbgHandler struct{}

func (m MbgHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", m.mbgWelcome)

	r.Route("/hello", func(r chi.Router) {
		//r.Use(PostCtx)
		r.Get("/", m.helloGet)   //  GET  /hello - Get MBG id
		r.Post("/", m.helloPost) // Post /hello  - Post MBG Id
	})

	r.Route("/addservice", func(r chi.Router) {
		r.Post("/", m.addServicePost) // Post /expose  - Expose mbg service
	})

	r.Route("/expose", func(r chi.Router) {
		r.Post("/", m.exposePost) // Post /expose  - Expose mbg service
	})

	r.Route("/connect", func(r chi.Router) {
		r.Post("/", m.connectPost)      // Post /connect  - Connect mbg service
		r.Connect("/", m.handleConnect) // Connect /connect  - Connect mbg service
		r.Delete("/", m.connectDelete)  // Disconnect /connect  - Disconnect mbg service

	})

	return r
}

func (m MbgHandler) mbgWelcome(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("Welcome to Multi-cloud Border Gateway"))
	if err != nil {
		log.Println(err)
	}
}
