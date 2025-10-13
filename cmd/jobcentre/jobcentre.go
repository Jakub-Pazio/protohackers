package main

import (
	"bean/pkg/pserver"
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"sync/atomic"
)

var portNumber = flag.Int("port", 4242, "Port number of server")

var idGen atomic.Int32

func newId() int {
	return int(idGen.Add(1))
}

var idGenClient atomic.Int32

func newClientId() int {
	return int(idGenClient.Add(1))
}

type QueueServer struct {
	js *JobService
}

func main() {
	flag.Parse()
	qs := &QueueServer{
		js: &JobService{
			jobmap:       make(JobMap),
			inprogresmap: make(JobMap),
			queuemap:     make(QueueMap),
			waitreqistry: make(WaitRegistry),
		},
	}
	handler := pserver.WithMiddleware(
		qs.handleConnection,
		pserver.LoggingMiddleware,
	)
	log.Fatal(pserver.ListenServe(handler, *portNumber))
}

type request struct {
	Request string `json:"request"`

	// fields for "put" request
	Queue string         `json:"queue,omitempty"`
	Job   map[string]any `json:"job,omitempty"`
	Pri   int            `json:"pri,omitempty"`

	// fields for "get" request
	Queues []string `json:"queues,omitempty"`
	Wait   bool     `json:"wait,omitempty"`

	// fields for "delete" and "abort" request
	Id int `json:"id,omitempty"`
}

type response struct {
	Status string `json:"status"`

	Id    int            `json:"id"`
	Job   map[string]any `json:"job"`
	Pri   int            `json:"pri"`
	Queue string         `json:"queue"`
}

func (s *QueueServer) handleConnection(conn net.Conn) {
	var workingSlice []int
	clientId := newClientId()
	log.Printf("Client %d connected\n", clientId)

	defer func() {
		log.Printf("Client %d disconecting\n", clientId)
		log.Printf("remocing jobs with id %+v\n", workingSlice)
		s.js.abortJobs(workingSlice)
		s.js.removeWait(clientId)
	}()

	defer pserver.HandleConnShutdown(conn)

	br := bufio.NewReader(conn)

	for {
		req, err := readRequest(br)
		if err == io.EOF {
			break
		} else if err != nil {
			log.Printf("Error reading request: %v\n", err)
			writeError(conn)
			continue
		}
		err = validateRequest(req)
		if err != nil {
			log.Printf("Invalid request: %v\n", err)
			writeError(conn)
			continue
		}
		log.Printf("Request: %+v\n", req)

		// Maybe here we could cast each request to its type that only has
		// the fields that are in the request
		switch req.Request {
		case "put":
			id := s.js.handlePut(req)
			writeIdResponse(conn, id)

		case "get":
			job := s.js.handleGet(req)
			if job == nil {
				if !req.Wait {
					writeNoJobResponse(conn)
					continue
				}
				ch := s.js.registerWait(req.Queues, clientId)
				job = <-ch
			}
			workingSlice = append(workingSlice, req.Id)
			writeJobResponse(conn, job)

		case "delete":
			s.js.handleDelete(req)

		case "abort":
			ok := s.js.handleAbort(req)
			if ok {
				writeStatusResponse(conn, "ok")
			} else {
				writeStatusResponse(conn, "no-job")
			}
		}
	}
}

func writeJobResponse(conn net.Conn, job *JobItem) {
	type response struct {
		Status string         `json:"status"`
		Id     int            `json:"id"`
		Job    map[string]any `json:"job"`
		Pri    int            `json:"pri"`
		Queue  string         `json:"queue"`
	}

	r := response{
		Status: "ok",
		Id:     job.Id,
		Job:    job.Job,
		Pri:    job.Pri,
		Queue:  job.Queue,
	}
	b, _ := json.Marshal(r)
	log.Printf("Response: %+v\n", r)

	conn.Write(b)
	conn.Write([]byte("\n"))
}

func writeNoJobResponse(conn net.Conn) {
	type response struct {
		Status string `json:"status"`
	}

	r := response{Status: "no-job"}
	log.Printf("Response: %+v\n", r)
	b, _ := json.Marshal(r)

	conn.Write(b)
	conn.Write([]byte("\n"))
}

type IdReponse struct {
	Status string `json:"status"`
	Id     int    `json:"id"`
}

type StatusResponse struct {
	Status string `json:"status"`
}

func writeIdResponse(conn net.Conn, id int) {

	r := IdReponse{Status: "ok", Id: id}
	log.Printf("Response: %+v\n", r)

	b, _ := json.Marshal(r)
	conn.Write(b)
	conn.Write([]byte("\n"))
}

func writeStatusResponse(conn net.Conn, status string) {
	r := StatusResponse{Status: status}
	log.Printf("Response: %+v\n", r)

	b, _ := json.Marshal(r)
	conn.Write(b)
	conn.Write([]byte("\n"))
}

func writeError(conn net.Conn) {
	j, _ := json.Marshal(response{Status: "error"})
	j = append(j, '\n')
	conn.Write(j)
}

func readRequest(br *bufio.Reader) (request, error) {
	line, err := br.ReadString('\n')
	if err != nil {
		return request{}, err
	}

	var req request
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		return request{}, err
	}

	return req, nil
}

func validateRequest(r request) error {
	switch r.Request {
	case "put":
		if r.Queue == "" {
			//TODO: Check if empty name is valid, if yes either use pointer to string
			// or 2nd version of JSON Go package
			return fmt.Errorf("empty queue name")
		}
		if r.Job == nil {
			return fmt.Errorf("job is empty")
		}
		if r.Pri < 0 {
			return fmt.Errorf("pri should be non-negative, but is %d", r.Pri)
		}
		return nil
	case "get":
		if len(r.Queues) < 1 {
			return fmt.Errorf("queues array is empty")
		}
		return nil
	case "delete":
		if r.Id < 1 {
			return fmt.Errorf("id must be positive, but is %d", r.Id)
		}
		return nil
	case "abort":
		if r.Id < 1 {
			return fmt.Errorf("id must be positive, but is %d", r.Id)
		}
		return nil
	default:
		return fmt.Errorf("unknown request: %q", r.Request)
	}
}
