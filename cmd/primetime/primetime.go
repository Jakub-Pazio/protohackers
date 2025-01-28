package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
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

func handleConnection(conn net.Conn) {
	defer func(conn net.Conn) {
		log.Println("closing connection")
		err := conn.Close()
		if err != nil {
			log.Printf("error when closing connection: %v\n", err)
		}
		log.Println("connection closed")
	}(conn)
	for {
		type Req struct {
			Method *string  `json:"method"`
			Number *big.Int `json:"number"`
		}
		type ReqF struct {
			Method *string  `json:"method"`
			Number *float64 `json:"number"`
		}
		type Res struct {
			Method string `json:"method"`
			Prime  bool   `json:"prime"`
		}
		reader := bufio.NewReaderSize(conn, 64*1024)
		for {
			line, err := reader.ReadSlice('\n')
			if err != nil {
				log.Printf("error when reading from socket: %v\n", err)
				conn.Close()
				return
			}
			log.Printf("got message: %v\n", string(line))

			var req Req
			err = json.Unmarshal(line, &req)
			if err != nil {
				//907679055587844667193521059264604184896872566615279722448
				//9223372036854775807
				log.Printf("error when unmarshaling message: %v\n", err)
				var reqF ReqF
				err = json.Unmarshal(line, &reqF)
				if err == nil {
					log.Println("here")
					if req.Method == nil || *req.Method != "isPrime" || req.Number == nil {
						log.Printf("method was not \"isPrime\" but: %v\n", req.Method)
						conn.Write(line)
						conn.Close()
						return
					}
					res := Res{
						Method: "isPrime",
						Prime:  false,
					}
					b, _ := json.Marshal(res)
					b = append(b, '\n')
					conn.Write(b)
					log.Printf("message contains floating point number: %f", reqF.Number)
					continue
				} else {
					fmt.Printf("could not unmarshal as float: %v", string(line))
				}
				conn.Write(line)
				conn.Close()
				return
			}
			if req.Method == nil || *req.Method != "isPrime" || req.Number == nil {
				log.Printf("method was not \"isPrime\" but: %v\n", req.Method)
				conn.Write(line)
				conn.Close()
				return
			}
			var res Res
			res.Method = "isPrime"
			if req.Number.ProbablyPrime(20) {
				res.Prime = true
			} else {
				res.Prime = false
			}
			b, err := json.Marshal(res)
			if err != nil {
				log.Printf("error when marshaling the response: %v\n", err)
			}
			b = append(b, '\n')
			n, err := conn.Write(b)
			log.Printf("responding with: %v\n", string(b))
			if n != len(b) {
				log.Printf("could not write whole response, had %d, but wrote %d\n", len(b), n)
			}
			if err != nil {
				log.Printf("error writing the response: %v\n", err)
			}
		}
	}
}
