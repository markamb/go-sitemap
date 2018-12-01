package main

import (
	"container/list"
	"sync"
)

// Hyperlink is a type for storing a pages hyperlink and associated metadata on a queue for crawling
type Hyperlink struct {
	urlStr string
	depth  int
}

// HyperlinkQueue is an an in-memory, thread-safe queue of Hyperlink entries.
//
// Note: We're using a linked list as a queue. This could be made more efficient using a more complex data structure
// such as a list of arrays or a single array working as a ring buffer (with re-allocations as required)
type HyperlinkQueue struct {
	queue list.List
	mutex sync.Mutex
}

// Push pushes a new item onto the end of the queue
func (q *HyperlinkQueue) Push(item Hyperlink) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	q.queue.PushBack(item)
}

// Pop removes the top item from the queue (if present)
// Returns the top item if present and a flag to indicate success
func (q *HyperlinkQueue) Pop() (Hyperlink, bool) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	if q.queue.Len() == 0 {
		return Hyperlink{}, false
	}
	f := q.queue.Front()
	q.queue.Remove(f)
	return f.Value.(Hyperlink), true
}

// Len returns the number of items in the queue
func (q *HyperlinkQueue) Len() int {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	return q.queue.Len()
}
