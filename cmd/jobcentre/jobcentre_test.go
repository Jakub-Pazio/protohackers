package main

import (
	"bufio"
	"encoding/json"
	"io"
	"net"
	"testing"
)

func TestReadRequest(t *testing.T) {
	r, w := io.Pipe()

	go func() {
		w.Write([]byte("{\"request\":\"delete\",\"id\":42}\n"))
	}()

	br := bufio.NewReader(r)

	req, err := readRequest(br)
	if err != nil {
		t.Errorf("Unexpected error: %v\n", err)
	}

	if req.Request != "delete" {
		t.Errorf("Expected delete request, got %q\n", req.Request)
	}

	if req.Id != 42 {
		t.Errorf("Expected Id: 42, got: %d\n", req.Id)
	}
}

func TestReadInvalidRequest(t *testing.T) {
	r, w := io.Pipe()

	go func() {
		w.Write([]byte("{\"request\":\"merge\",\"id\":42}\n"))
		w.Write([]byte("{\"request\":\"abort\",\"id\":42}\n"))
	}()

	br := bufio.NewReader(r)

	req, _ := readRequest(br)
	err := validateRequest(req)

	if err == nil {
		t.Errorf("Expected \"unknown request\", but got no error\n")
	}

	req, err = readRequest(br)

	if err != nil {
		t.Errorf("Unexpected error: %v\n", err)
	}

	if req.Request != "abort" {
		t.Errorf("Expected \"abort\" got %q\n", req.Request)
	}

	err = validateRequest(req)

	if err != nil {

	}
}

// TODO: add shutdown mechanism for the handler, so we can finish the test after client read response
func TestScenarios(t *testing.T) {
	c, s := net.Pipe()

	done := make(chan struct{})

	server := &QueueServer{
		js: &JobService{
			jobmap:       make(JobMap),
			inprogresmap: make(JobMap),
			queuemap:     make(QueueMap),
			waitreqistry: make(WaitRegistry),

			ActionChan: make(chan func()),
			StopChan:   make(chan struct{}),
		},
	}

	go server.js.Initialize()

	go func(done chan struct{}) {
		put1 := putRequest("q1", 1)
		sendRequest(put1, c)
		response, _ := readResponse(c)

		if response.Id != 1 {
			t.Errorf("exprected id of 1, got %d\n", response.Id)
		}

		put10 := putRequest("q1", 10)
		sendRequest(put10, c)
		response, _ = readResponse(c)

		if response.Id != 2 {
			t.Errorf("exprected id of 2, got %d\n", response.Id)
		}

		put2 := putRequest("q1", 2)
		sendRequest(put2, c)
		response, _ = readResponse(c)

		if response.Id != 3 {
			t.Errorf("exprected id of 3, got %d\n", response.Id)
		}

		get1 := request{Request: "get", Queues: []string{"q1", "q2"}}
		sendRequest(get1, c)
		response, _ = readResponse(c)

		inprogress := len(server.js.inprogresmap)
		if inprogress != 1 {
			t.Errorf("1 job should be in progres, but got %d\n", inprogress)
		}

		t.Logf("%+v\n", response)
		get2 := request{Request: "get", Queues: []string{"q1", "q2"}}
		sendRequest(get2, c)
		response, _ = readResponse(c)

		inprogress = len(server.js.inprogresmap)
		if inprogress != 2 {
			t.Errorf("2 job should be in progres, but got %d\n", inprogress)
		}

		t.Logf("%+v\n", response)

		done <- struct{}{}
	}(done)

	go server.handleConnection(s)
	<-done
}

func TestScenarios2(t *testing.T) {
	// c, s := net.Pipe()
	//
	// done := make(chan struct{})
	//
	// server := &QueueServer{
	// 	js: &JobService{
	// 		jobmap:       make(JobMap),
	// 		inprogresmap: make(JobMap),
	// 		queuemap:     make(QueueMap),
	// 		waitreqistry: make(WaitRegistry),
	// 	},
	// }
	//
	// go func(done chan struct{}) {
	// 	get1 := request{Request: "get", Queues: []string{"q1", "q2"}, Wait: true}
	// 	sendRequest(get1, c)
	// 	response, _ := readResponse(c)
	// 	want := "no-job"
	// 	if response.Status != want {
	// 		t.Errorf("want %q, got: %q\n", want, response.Status)
	// 	}
	//
	// 	done <- struct{}{}
	// }(done)
	//
	// go func() {
	// 	put1 := putRequest("q1", 10)
	// 	sendRequest(put1, conn)
	// }()
	//
	// go server.handleConnection(s)
	// <-done
}

func putRequest(queue string, priority int) request {
	return request{
		Request: "put",
		Queue:   queue,
		Job: map[string]any{
			"work":    "add",
			"times":   2,
			"numbers": []int{1, 2, 3},
		},
		Pri: priority,
	}
}

func sendRequest(r request, conn net.Conn) {
	b, _ := json.Marshal(r)
	conn.Write(b)
	conn.Write([]byte("\n"))
}

func readResponse(conn net.Conn) (response, error) {
	var r response

	br := bufio.NewReader(conn)
	bytes, err := br.ReadBytes('\n')

	if err != nil {
		return response{}, err
	}

	err = json.Unmarshal(bytes, &r)

	if err != nil {
		return response{}, err
	}

	return r, nil
}
