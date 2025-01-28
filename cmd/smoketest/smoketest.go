package main

import (
	"io"
	"log"
	"net"
)

func main() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("cannot bind to tcp socket: %v\n", err)
	}
	log.Println("server started successfully")
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("cannot accept connection: %v\n", err)
		}
		log.Println("accepted connection")
		go handleConnection(conn)
	}
}

// This code does not work as expected
func hConn(conn net.Conn) {
	defer conn.Close()
	for {
		_, err := io.Copy(conn, conn)
		if err != nil {
			if err == io.EOF {
				log.Println("client disconnected gracefully")
			} else {
				log.Println("error reading from socket:", err)
			}
			return
		}
	}
}

func handleConnection(conn net.Conn) {
	defer func(conn net.Conn) {
		log.Println("closing connection")
		err := conn.Close()
		if err != nil {
			log.Printf("error when closing connection: %v\n", err)
		}
		log.Println("connection closed")
	}(conn)
	buffer := make([]byte, 1024)
	for {
		n, err := conn.Read(buffer)
		if n == 0 {
			break
		}
		if err != nil {
			log.Printf("error reading from sokcet: %v\n", err)
			break
		}
		wn, err := conn.Write(buffer[:n])
		log.Printf("read and send: %s", buffer[:n])
		if err != nil {
			log.Printf("error writing to sokcet: %v\n", err)
		}
		if wn != n {
			log.Printf("read %d bytes but wrote %d bytes\n", n, wn)
		}
	}
}
