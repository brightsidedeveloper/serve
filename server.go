package serve

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/brightsidedeveloper/serve/router"
)

type Server struct {
	server *http.Server
}

func NewServer(service *router.Router) *Server {
	port := os.Getenv("PORT")
	if port == "" {
		log.Println("No port set... defaulting to 8080")
		port = "8080"
	}

	s := &http.Server{
		Addr:    ":" + port,
		Handler: service.Handler,
	}
	return &Server{
		server: s,
	}
}

func NewKillChannel() chan os.Signal {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	return stop
}

func (s *Server) Listen() {
	log.Printf("Listening on %s...", s.server.Addr)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}

func (s *Server) Shutdown() context.CancelFunc {
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	if err := s.server.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown failed: %v", err)
	}

	log.Println("Server gracefully shutdown")

	return cancel
}
