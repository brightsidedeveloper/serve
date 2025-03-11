package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type Service struct {
	Handler *http.ServeMux
}

func NewService() *Service {
	return &Service{
		Handler: http.NewServeMux(),
	}
}

type Context struct {
	W      http.ResponseWriter
	R      *http.Request
	String func(s string) error
	JSON   func(map[string]any) error
}

func Handle(h func(r *Context) error) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		req := &Context{
			W: w,
			R: r,
			String: func(s string) error {
				_, err := w.Write([]byte(s))
				return err
			},
			JSON: func(m map[string]any) error {
				w.Header().Set("Content-Type", "application/json")
				return json.NewEncoder(w).Encode(m)
			},
		}
		if err := h(req); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

type Route struct {
	Path   string
	GET    func(r *Context) error
	POST   func(r *Context) error
	PUT    func(r *Context) error
	PATCH  func(r *Context) error
	DELETE func(r *Context) error
}

func (s *Service) Mount(route *Route) {
	if route.Path == "" {
		log.Println("Path is required to Mount Route")
		return
	}
	handler := func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if route.GET != nil {
				Handle(route.GET)(w, r)
				return
			}
		case http.MethodPost:
			if route.POST != nil {
				Handle(route.POST)(w, r)
				return
			}
		case http.MethodPut:
			if route.PUT != nil {
				Handle(route.PUT)(w, r)
				return
			}
		case http.MethodPatch:
			if route.PATCH != nil {
				Handle(route.PATCH)(w, r)
				return
			}
		case http.MethodDelete:
			if route.DELETE != nil {
				Handle(route.DELETE)(w, r)
				return
			}
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}

	s.Handler.HandleFunc(route.Path, handler)
}
