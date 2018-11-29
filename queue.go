package main

import (
	"container/list"
	"sync"
)

//
// StringQueue is an an in-memory, thread-safe queue
//
// Note: We're using a linked list as a queue. This could be made more efficient using a more complex data structure
// such as a list of arrays or a single array working as a ring buffer (with re-allocations as required)
//
type StringQueue struct {
	queue		list.List
	mutex		sync.Mutex
}

// Push pushes a new item onto the end of the queue
func (q *StringQueue) Push(item string) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	q.queue.PushBack(item)
}

//
// Pop removes the top item from the queue (if present)
// Returns the top item if present and a flag to indicate success
func (q *StringQueue) Pop() (string, bool) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	if q.queue.Len() == 0 {
		return "", false
	}
	f := q.queue.Front()
	q.queue.Remove(f)
	return f.Value.(string), true
}

func (q *StringQueue) Len() int {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	return q.queue.Len()
}
