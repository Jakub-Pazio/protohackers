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
			break
		}
		fmt.Printf("clientId: %v\n", clientId)
		fmt.Printf("waitEntry: %v\n", waitEntry)
		//TODO: for every queue name in the entry remove it from the waitqueue
		for _, qn := range waitEntry.queues {
			delete(js.waitreqistry[qn], clientId)
		}
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

func (js *JobService) handleAbort(req request) bool {
	//TODO: check if current user has this job ID in its workslice

	job, ok := js.inprogresmap[req.Id]
	if !ok {
		// job was removed by some other user, we must send "no-job"
		return false
	}

	delete(js.inprogresmap, job.Id)
	js.jobmap[job.Id] = job

	queue := js.getQueue(job.Queue)
	heap.Push(queue, job)

	return true
}

func (js *JobService) handleDelete(req request) error {
	//TODO: Remove job from clients work slice
	job, ok := js.jobmap[req.Id]
	if ok {
		delete(js.jobmap, req.Id)
		queue := js.queuemap[job.Queue]
		heap.Remove(queue, job.Index)
		return nil
	}

	job, ok = js.inprogresmap[req.Id]
	if ok {
		delete(js.inprogresmap, req.Id)
		return nil
	}
	return errors.New("no job to delete")
}

func (js *JobService) handlePut(req request) int {
	id := newId()
	qname := req.Queue

	job := &JobItem{
		Job:   req.Job,
		Id:    id,
		Pri:   req.Pri,
		Queue: qname,
	}

	if ok := js.checkWaitJob(job, req.Queues); ok {
		return id
	}

	queue := js.getQueue(qname)
	if queue == nil {
		queue = js.createQueue(qname)
	}

	heap.Push(queue, job)
	js.jobmap[id] = job

	return id
}

// TODO: if job is not found we should and we wait for job we need to return a channel
func (js *JobService) handleGet(req request) *JobItem {
	queues := req.Queues

	jobs := js.peekQueues(queues)
	if len(jobs) < 1 {
		return nil
	}

	job := maxPrioJob(jobs)

	js.moveToInProgress(job)

	return job
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
