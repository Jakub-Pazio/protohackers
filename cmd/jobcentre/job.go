package main

type JobItem struct {
	Job   map[string]any
	Id    int
	Pri   int
	Index int
	Queue string
}

type MapItem struct {
	job   *JobItem
	queue *JobsQueue
}

type JobMap map[int]*JobItem

// JobsQueue is single queue with defined name
type JobsQueue []*JobItem

// When searching for highest priority Job we need to ask queue with provided name
type QueueMap map[string]*JobsQueue

func (jq JobsQueue) Len() int { return len(jq) }

func (jq JobsQueue) Less(i, j int) bool {
	return jq[i].Pri > jq[j].Pri
}

func (jq JobsQueue) Swap(i, j int) {
	jq[i], jq[j] = jq[j], jq[i]
	jq[i].Index = j
	jq[j].Index = i
}

func (jq *JobsQueue) Push(x any) {
	n := len(*jq)
	item := x.(*JobItem)
	item.Index = n
	*jq = append(*jq, item)
}

func (jq *JobsQueue) Pop() any {
	old := *jq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // don't stop the GC from reclaiming the item eventually
	item.Index = -1 // for safety
	*jq = old[0 : n-1]
	return item
}
