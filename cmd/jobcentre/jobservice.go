package main

import (
	"container/heap"
	"errors"
	"fmt"
	"log"
	"slices"
)

type JobService struct {
	// jobmap contains all maps in queue
	jobmap JobMap
	// inprogresmap contains all jobs that handled by clients
	inprogresmap JobMap
	queuemap     QueueMap
	waitreqistry WaitRegistry

	// https://youtu.be/LHe1Cb_Ud_M?si=bQHiCCxIlEp3LTnA&t=410
	ActionChan chan func()
	StopChan   chan struct{}
}

func (js *JobService) Initialize() {
	log.Println("Started")
	for {
		log.Println("Waiting for request")
		select {
		case f := <-js.ActionChan:
			log.Println("Func call")
			f()
		case <-js.StopChan:
			return
		}
	}
}

type rEntry struct {
	ch     chan *JobItem
	queues []string
}

type WaiterMap map[int]rEntry

type WaitRegistry map[string]WaiterMap

// TODO: of some of map if not created we need to dynamicaly create one or remove when size == 0
func (js *JobService) registerWait(queues []string, clientId int) chan *JobItem {
	ch := make(chan *JobItem)
	entry := rEntry{ch: ch, queues: queues}

	for _, queue := range queues {
		q, ok := js.waitreqistry[queue]
		if !ok {
			wmap := make(WaiterMap)
			js.waitreqistry[queue] = wmap
			q = wmap
		}
		q[clientId] = entry
	}

	return ch
}

func (js *JobService) checkWaitJob(job *JobItem, queues []string) bool {
	for _, queuename := range queues {
		entry, ok := js.waitreqistry[queuename]
		if !ok {
			continue
		}
		var clientId int
		var waitEntry rEntry
		for clientId, waitEntry = range entry {
			if clientId != 0 {
				break
			}
		}
		if clientId == 0 {
			continue
		}
		fmt.Printf("clientId: %v\n", clientId)
		fmt.Printf("waitEntry: %v\n", waitEntry)
		//TODO: for every queue name in the entry remove it from the waitqueue
		for _, qn := range waitEntry.queues {
			delete(js.waitreqistry[qn], clientId)
		}
		js.inprogresmap[job.Id] = job
		waitEntry.ch <- job
		return true
	}

	return false
}

func (js *JobService) removeWait(clientId int) {
	for _, wm := range js.waitreqistry {
		e, ok := wm[clientId]
		if !ok {
			continue
		}
		for _, registered := range e.queues {
			entry := js.waitreqistry[registered]
			delete(entry, clientId)
		}
		return
	}
}

func (js *JobService) HandleDisconnect(workingSlice []int, clientId int) {

	c := make(chan struct{})

	js.ActionChan <- func() {
		js.abortJobs(workingSlice)
		js.removeWait(clientId)
		c <- struct{}{}
	}

	<-c

}

func (js *JobService) HandleAbort(req request) bool {
	c := make(chan bool)

	js.ActionChan <- func() {
		job, ok := js.inprogresmap[req.Id]
		if !ok {
			// job was removed by some other user, we must send "no-job"
			c <- false
			return
		}

		if ok := js.checkWaitJob(job, []string{job.Queue}); ok {
			c <- true
			return
		}

		delete(js.inprogresmap, job.Id)
		js.jobmap[job.Id] = job

		queue := js.getQueue(job.Queue)
		heap.Push(queue, job)

		c <- true
	}

	return <-c
}

func (js *JobService) HandleDelete(req request) error {

	c := make(chan error)

	js.ActionChan <- func() {

		job, ok := js.jobmap[req.Id]
		if ok {
			delete(js.jobmap, req.Id)
			queue := js.queuemap[job.Queue]
			heap.Remove(queue, job.Index)
			c <- nil
			return
		}

		job, ok = js.inprogresmap[req.Id]
		if ok {
			delete(js.inprogresmap, req.Id)
			c <- nil
			return
		}
		c <- errors.New("no job to delete")
	}

	return <-c
}

func (js *JobService) HandlePut(req request) int {

	c := make(chan int)

	js.ActionChan <- func() {
		id := newId()
		qname := req.Queue

		job := &JobItem{
			Job:   req.Job,
			Id:    id,
			Pri:   req.Pri,
			Queue: qname,
		}

		if ok := js.checkWaitJob(job, req.Queues); ok {
			c <- id
			return
		}

		queue := js.getQueue(qname)
		if queue == nil {
			queue = js.createQueue(qname)
		}

		heap.Push(queue, job)
		js.jobmap[id] = job

		c <- id
	}

	return <-c

}

func (js *JobService) HandleGet(req request, clientId int) (*JobItem, chan *JobItem) {

	type result struct {
		item *JobItem
		ch   chan *JobItem
	}

	c := make(chan result)

	js.ActionChan <- func() {
		queues := req.Queues

		jobs := js.peekQueues(queues)
		if len(jobs) < 1 {
			if req.Wait == false {
				c <- result{nil, nil}
				return
			}
			ch := js.registerWait(req.Queues, clientId)
			c <- result{nil, ch}
			return
		}

		job := maxPrioJob(jobs)

		js.moveToInProgress(job)

		c <- result{job, nil}
	}

	r := <-c
	return r.item, r.ch
}

func (js *JobService) moveToInProgress(job *JobItem) {

	delete(js.jobmap, job.Id)
	js.inprogresmap[job.Id] = job

	q := js.queuemap[job.Queue]
	removed := heap.Pop(q)
	jobRemoved := removed.(*JobItem)
	log.Printf("Job removed: %+v\n", jobRemoved)
}

func maxPrioJob(jobs []*JobItem) *JobItem {
	return slices.MaxFunc(jobs, func(j1, j2 *JobItem) int { return j1.Pri - j2.Pri })
}

func (js *JobService) peekQueues(queues []string) []*JobItem {
	var top []*JobItem
	for _, qname := range queues {
		queue := js.queuemap[qname]
		if queue != nil && len(*queue) > 0 {
			first := (*queue)[0]
			top = append(top, first)
		}
	}
	return top
}

func (js *JobService) getQueue(name string) *JobsQueue {
	return js.queuemap[name]
}

func (js *JobService) createQueue(name string) *JobsQueue {
	queue := make(JobsQueue, 0, 10)
	heap.Init(&queue)

	js.queuemap[name] = &queue
	return &queue
}

func (js *JobService) abortJobs(ids []int) {
	for _, id := range ids {
		job, ok := js.inprogresmap[id]
		if !ok {
			log.Printf("No job id: %d found to remove\n", id)
			continue
		}
		log.Printf("Aborting job %+v\n", job)
		delete(js.inprogresmap, job.Id)
		js.jobmap[job.Id] = job
		queue := js.getQueue(job.Queue)
		log.Printf("Queue size before: %d\n", len(*queue))
		arr := []string{job.Queue}
		ok = js.checkWaitJob(job, arr)
		if !ok {
			heap.Push(queue, job)
		} else {
			log.Printf("Job %d has been assigned to other client\n", job.Id)
		}
		log.Printf("Queue size after: %d\n", len(*queue))
	}
}
