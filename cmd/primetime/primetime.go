package main

import (
	"bean/pkg/pserver"
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net"
)

const BufferSize = 1024 * 64

var portNumber = flag.Int("port", 4242, "Port number of server")

func main() {
	handler := pserver.WithMiddleware(
		handleConnection,
		pserver.LoggingMiddleware,
	)
	log.Fatal(pserver.ListenServe(handler, *portNumber))
}

func handleConnection(conn net.Conn) {
	defer pserver.HandleConnShutdown(conn)
	reader := bufio.NewReaderSize(conn, BufferSize)
	for {
		line, err := reader.ReadSlice('\n')
		if err != nil {
			log.Printf("error when reading from socket: %v\n", err)
			_, _ = conn.Write(line)
			return
		}

		log.Printf("request: %s", string(line))

		value, ok := getIntFromRequest(line)
		if ok {
			_ = writeResponse(conn, checkIsPrime(*value))
			continue
		}

		if checkIsValidFloat(line) {
			_ = writeResponse(conn, false)
			continue
		}
		fmt.Printf("malformed response: %s", line)
		_, _ = conn.Write(line)
		return
	}
}

type reqBigInt struct {
	Method *string  `json:"method"`
	Number *big.Int `json:"number"`
}
type reqFloat struct {
	Method *string  `json:"method"`
	Number *float64 `json:"number"`
}
type res struct {
	Method string `json:"method"`
	Prime  bool   `json:"prime"`
}

func writeResponse(conn net.Conn, value bool) error {
	response := res{
		Method: "isPrime",
		Prime:  value,
	}
	b, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("error when marshaling the response: %v", err)
	}
	b = append(b, '\n')
	n, err := conn.Write(b)
	if err != nil {
		return fmt.Errorf("error when writing to client: %v", err)
	}
	if n != len(b) {
		return fmt.Errorf("could not write the whole response: had %d wrote: %d", n, len(b))
	}
	log.Printf("response: %s", string(b))
	return nil
}

func getIntFromRequest(line []byte) (*big.Int, bool) {
	var req reqBigInt
	err := json.Unmarshal(line, &req)
	if err == nil && req.Method != nil && *req.Method == "isPrime" && req.Number != nil {
		return req.Number, true
	}
	return nil, false
}

func checkIsValidFloat(line []byte) bool {
	var req reqFloat
	err := json.Unmarshal(line, &req)
	if err == nil && req.Method != nil && *req.Method == "isPrime" && req.Number != nil {
		log.Printf("got float: %f\n", *req.Number)
		return true
	}
	return false
}

func checkIsPrime(b big.Int) bool {
	return b.ProbablyPrime(20)
}
