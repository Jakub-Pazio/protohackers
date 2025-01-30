package main

import (
	"bean/pkg/pserver"
	"flag"
	"log"
	"net"
)

const BufferSize = 1024

var portNumber = flag.Int("port", 4242, "Port number of server")

func main() {
	flag.Parse()
	handler := pserver.WithMiddleware(
		handleConnection,
		pserver.LoggingMiddleware,
	)

	log.Fatal(pserver.ListenServe(handler, *portNumber))
}

func handleConnection(conn net.Conn) {
	defer pserver.HandleConnShutdown(conn)
	buffer := make([]byte, BufferSize)
	for {
		n, err := conn.Read(buffer)
		if n == 0 {
			log.Println("end of data from client")
			return
		}
		if err != nil {
			log.Printf("error reading from socket: %v\n", err)
			return
		}
		wn, err := conn.Write(buffer[:n])
		log.Printf("read and send: %s", buffer[:n])
		if err != nil {
			log.Printf("error writing to sokcet: %v\n", err)
		}
		if wn != n {
			log.Printf("read %d but wrote %d bytes\n", n, wn)
		}
	}
}
