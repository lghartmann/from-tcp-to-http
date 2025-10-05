package server

import (
	"fmt"
	"io"
	"net"

	"github.com/lghartmann/from-tcp-to-http/internal/request"
	"github.com/lghartmann/from-tcp-to-http/internal/response"
)

type Server struct {
	closed  bool
	handler Handler
}

type Handler func(w *response.Writer, req *request.Request)

type HandlerError struct {
	StatusCode response.StatusCode
	Message    string
}

func (s *Server) Close() error {
	return nil
}

func Serve(port uint16, handler Handler) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}

	server := &Server{closed: false, handler: handler}
	go runServer(server, listener)

	return server, nil
}

func runServer(s *Server, listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}

		if s.closed {
			return
		}

		go runConnection(s, conn)
	}
}

func runConnection(s *Server, conn io.ReadWriteCloser) {
	defer conn.Close()

	responseWriter := response.NewWriter(conn)
	r, err := request.RequestFromReader(conn)
	if err != nil {
		responseWriter.WriteStatusLine(response.StatusBadRequest)
		responseWriter.WriteHeaders(response.GetDefaultHeaders(0))
		return
	}

	s.handler(responseWriter, r)
}
