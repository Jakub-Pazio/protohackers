package pserver

import (
	"fmt"
	"log"
	"net"
)

type HandlerFunc func(conn net.Conn)

type Middleware func(next HandlerFunc) HandlerFunc

// ListenServe is responsible for starting the sever and listening on the given port
func ListenServe(handler HandlerFunc, port int) error {
	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to bind to port %d, %w", port, err)
	}
	log.Printf("server started successfully, running at port: %d\n", port)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("cannot accept connection: %v\n", err)
		}
		go handler(conn)
	}
}

func WithMiddleware(handler HandlerFunc, ms ...Middleware) HandlerFunc {
	for i := len(ms) - 1; i >= 0; i-- {
		handler = ms[i](handler)
	}
	return handler
}

func LoggingMiddleware(next HandlerFunc) HandlerFunc {
	return func(conn net.Conn) {
		log.Printf("new connection from: %s\n", conn.RemoteAddr().String())
		next(conn)
		log.Printf("end of connection from :%s\n", conn.RemoteAddr().String())
	}
}

func HandleConnShutdown(conn net.Conn) {
	log.Println("closing connection")
	err := conn.Close()
	if err != nil {
		log.Printf("error when closing connection: %v\n", err)
	}
	log.Println("connection closed")
}
