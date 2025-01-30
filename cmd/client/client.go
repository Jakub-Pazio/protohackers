package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	targetAddr := "127.0.0.1:4242" // Destination server
	localPort := "54321"           // Port to send from and listen on

	// Resolve target UDP address
	raddr, err := net.ResolveUDPAddr("udp", targetAddr)
	if err != nil {
		fmt.Println("Error resolving target address:", err)
		return
	}

	// Bind to the specified local port
	laddr, err := net.ResolveUDPAddr("udp", ":"+localPort)
	if err != nil {
		fmt.Println("Error resolving local address:", err)
		return
	}

	// Create a UDP connection
	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		fmt.Println("Error binding to local port:", err)
		return
	}
	defer conn.Close()

	// Send a UDP packet
	message := []byte("version")
	_, err = conn.WriteToUDP(message, raddr)
	if err != nil {
		fmt.Println("Error sending UDP packet:", err)
		return
	}
	fmt.Printf("Sent: %s → %s\n", message, targetAddr)

	// Set a timeout for receiving response
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Buffer to receive response
	buf := make([]byte, 1024)
	n, addr, err := conn.ReadFromUDP(buf)
	if err != nil {
		fmt.Println("No response received:", err)
		return
	}

	// Print received data
	fmt.Printf("Received from %s: %s\n", addr, string(buf[:n]))
}
