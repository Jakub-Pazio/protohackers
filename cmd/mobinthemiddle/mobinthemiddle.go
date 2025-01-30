package main

import (
	"bean/pkg/pserver"
	"bufio"
	"flag"
	"github.com/dlclark/regexp2"
	"log"
	"net"
)

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
	fConn, err := net.Dial("tcp", "chat.protohackers.com:16963")
	//fConn, err := net.Dial("tcp", "localhost:4243")
	if err != nil {
		log.Printf("could not connect to upstream: %v", err)
		_ = conn.Close()
		return
	}
	go BogCoinProxy(fConn, conn)
	go BogCoinProxy(conn, fConn)
}

func BogCoinProxy(input, output net.Conn) {
	defer pserver.HandleConnShutdown(input)
	defer pserver.HandleConnShutdown(output)
	//re := regexp.MustCompile(`\b7[a-zA-Z0-9]{25,34}\b`)
	//re := regexp.MustCompile(`(?:^|\s)(7[a-zA-Z0-9]{25,34})(?=\s|\n)`)
	re := regexp2.MustCompile(`(?<=^|\s)7[a-zA-Z0-9]{25,34}(?=\s|\n)`, 0)
	reader := bufio.NewReader(input)
	for {
		message, err := reader.ReadString('\n')
		if err != nil || message == "" {

			return
		}
		log.Printf("reading message: %q\n", message)
		result, _ := re.Replace(message, "7YWHMfk9JZe0LM0g1ZauHuiSxhI", -1, -1)

		log.Printf("replaced all adresses")
		if result[len(result)-1] != '\n' {
			result += "\n"
		}
		output.Write([]byte(result))
		log.Printf("send: %q\n", result)
	}
}
