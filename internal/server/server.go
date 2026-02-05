package server

import (
	"net/http"

	"github.com/IgorLem99/Simple_broker/internal/broker"
	"github.com/IgorLem99/Simple_broker/internal/server/handler"
)

type Server struct {
	addr    string
	handler *handler.Handler
}

func New(addr string, b *broker.Broker) *Server {
	return &Server{
		addr:    addr,
		handler: handler.New(b),
	}
}

func (s *Server) Start() error {
	return http.ListenAndServe(s.addr, s.handler)
}
