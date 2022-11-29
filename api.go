package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/stampede"
)

var (
	//go:embed template.html
	TemplateFS embed.FS

	//go:embed style.css
	StyleFS embed.FS
)

type Server struct {
	HttpClient *http.Client
	ATBBaseURL string

	httpServer *http.Server
	cache      *stampede.Cache
	templates  *template.Template
}

func (s *Server) Run(_ context.Context, addr string) error {
	s.ATBBaseURL = strings.TrimSuffix(s.ATBBaseURL, "/")

	s.templates = template.Must(template.ParseFS(TemplateFS, "template.html"))

	s.cache = stampede.NewCache(1, 100*time.Minute, 120*time.Minute)

	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           s.router(),
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	log.Printf("listening on %s\n", addr)
	err := s.httpServer.ListenAndServe()
	return err
}

func (s *Server) router() *chi.Mux {
	router := chi.NewRouter()
	router.Use(middleware.Recoverer)
	router.Use(middleware.Compress(5))
	router.Use(middleware.Heartbeat("/ping"))

	router.Get("/api", s.getDiscountItemsAPICtrl)
	router.Get("/", s.getDiscountItemsHTMLCtrl)

	fs := http.FileServer(http.FS(StyleFS))
	router.Mount("/", fs)

	return router
}

func (s *Server) getDiscountItemsAPICtrl(w http.ResponseWriter, r *http.Request) {
	items, err := s.GetDiscountItems(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("%s", err), 500)
		log.Printf("could not scrape discount items: %v", err)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(items)
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		log.Printf("could not encode discount items to json: %v", err)
	}
}

func (s *Server) getDiscountItemsHTMLCtrl(w http.ResponseWriter, r *http.Request) {
	items, err := s.GetDiscountItems(r.Context())
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		log.Printf("could not scrape discount items: %v", err)
		return
	}
	tmplData := struct{ Items []DiscountItem }{Items: items}
	w.Header().Add("Content-Type", "text/html")
	err = s.templates.ExecuteTemplate(w, "template.html", &tmplData)
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		log.Printf("could not render template: %v", err)
		return
	}
}
