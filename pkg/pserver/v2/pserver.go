package v2

import (
	"context"
	"fmt"
	"log"
	"net"
)

type HandlerFunc func(ctx context.Context, conn net.Conn)

type UDPHandlerFunc func(msg string) string

type Middleware func(next HandlerFunc) HandlerFunc

// ListenServe is responsible for starting the sever and listening on the given port
func ListenServe(ctx context.Context, handler HandlerFunc, port int) error {
	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("Failed to bind to port %d, %w", port, err)
	}
	log.Printf("Server started successfully, running at port: %d\n", port)
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Cannot accept connection: %v\n", err)
		}
		go func() {
			handler(ctx, conn)
		}()
	}
}

func ListenServeUDP(handler UDPHandlerFunc, port int) error {
	addr := fmt.Sprintf(":%d", port)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	ln, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("failed to bind to port %d, %w", port, err)
	}
	log.Printf("server started successfully, running at port: %d\n", port)
	buffer := make([]byte, 1024)
	for {
		//n, err := ln.Read(buffer)
		n, remoteAddr, err := ln.ReadFromUDP(buffer)
		log.Printf("got message: %q\n", buffer[:n])
		if err != nil {
			return err
		}
		response := handler(string(buffer[:n]))
		if response == "" {
			continue
		}
		_, _ = ln.WriteToUDP([]byte(response), remoteAddr)
		log.Printf("send message: %q\n", response)
	}
}

func WithMiddleware(handler HandlerFunc, ms ...Middleware) HandlerFunc {
	for i := len(ms) - 1; i >= 0; i-- {
		handler = ms[i](handler)
	}
	return handler
}

func LoggingMiddleware(next HandlerFunc) HandlerFunc {
	return func(ctx context.Context, conn net.Conn) {
		log.Printf("New connection from: %s\n", conn.RemoteAddr().String())
		next(ctx, conn)
		log.Printf("End of connection from :%s\n", conn.RemoteAddr().String())
	}
}

func HandleConnShutdown(conn net.Conn) {
	log.Println("Closing connection")
	err := conn.Close()
	if err != nil {
		log.Printf("Error when closing connection: %v\n", err)
	}
	log.Println("Connection closed")
}
