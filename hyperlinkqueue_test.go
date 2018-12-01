package main

import (
	"strconv"
	"sync"
	"testing"
)

func TestEmptyQueue(t *testing.T) {

	q := HyperlinkQueue{}
	if l := q.Len(); l != 0 {
		t.Errorf("Incorrect length on empty queue: expected %d, got %d", 0, l)
	}

	if top, found := q.Pop(); found || len(top.urlStr) > 0 {
		t.Errorf("Pop from empty queue returned incorrect result: expected (, false), got (%s, %v)", top.urlStr, found)
	}
}

func TestQueue(t *testing.T) {

	q := HyperlinkQueue{}

	for i := 0; i < 100; i++ {
		q.Push(Hyperlink{strconv.Itoa(i + 1), 0})
	}

	if l := q.Len(); l != 100 {
		t.Errorf("Incorrect length on queue: expected %d, got %d", 100, l)
	}

	if top, found := q.Pop(); !found || top.urlStr != "1" {
		t.Errorf(`Pop returned incorrect result: expected ("1", true), got (%s, %v)`, top.urlStr, found)
	}
	if l := q.Len(); l != 99 {
		t.Errorf("Incorrect length on queue: expected %d, got %d", 99, l)
	}

	if top, found := q.Pop(); !found || top.urlStr != "2" {
		t.Errorf(`Pop returned incorrect result: expected ("2", true), got (%s, %v)`, top.urlStr, found)
	}
	if l := q.Len(); l != 98 {
		t.Errorf("Incorrect length on queue: expected %d, got %d", 98, l)
	}

	for i := 97; i >= 0; i-- {
		if _, found := q.Pop(); !found {
			t.Errorf(`Pop failed for iteration %d`, i)
		}
		if l := q.Len(); l != i {
			t.Errorf("Incorrect length on queue: expected %d, got %d", i, l)
		}
	}

	// should now be empty
	if l := q.Len(); l != 0 {
		t.Errorf("Incorrect length: expected %d, got %d", 0, l)
	}
	if top, found := q.Pop(); found || len(top.urlStr) > 0 {
		t.Errorf("Pop from empty queue returned incorrect result: expected (, false), got (%s, %v)", top.urlStr, found)
	}

	// one more push and pop
	q.Push(Hyperlink{"TEST", 0})
	if l := q.Len(); l != 1 {
		t.Errorf("Incorrect length: expected %d, got %d", 1, l)
	}
	if top, found := q.Pop(); !found || top.urlStr != "TEST" {
		t.Errorf(`Pop returned incorrect result: expected ("TEST", true), got (%s, %v)`, top.urlStr, found)
	}

	// should now be empty again
	if l := q.Len(); l != 0 {
		t.Errorf("Incorrect length: expected %d, got %d", 0, l)
	}
	if top, found := q.Pop(); found || len(top.urlStr) > 0 {
		t.Errorf("Pop from empty queue returned incorrect result: expected (, false), got (%s, %v)", top.urlStr, found)
	}
}

func TestConcurrentQueue(t *testing.T) {
	// very basic test to throw a lot of concurrent operations at a queue

	var wg sync.WaitGroup
	q := HyperlinkQueue{}

	t.Log("Starting concurrent queue population")
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(num int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				q.Push(Hyperlink{"TEST" + strconv.Itoa(num*100+j), 0})
			}
		}(i)
	}
	wg.Wait()

	// we should have 10,000 items in the queue in a random order
	if l := q.Len(); l != 10000 {
		t.Errorf("Incorrect length: expected %d, got %d", 10000, l)
	}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				if _, found := q.Pop(); !found {
					t.Errorf(`Pop returned incorrect result: expected true, got %v`, found)
				}
			}
		}()
	}
	wg.Wait()

	// should have an empty queue
	if l := q.Len(); l != 0 {
		t.Errorf("Incorrect length: expected %d, got %d", 0, l)
	}
	if top, found := q.Pop(); found || len(top.urlStr) > 0 {
		t.Errorf("Pop from empty queue returned incorrect result: expected (, false), got (%s, %v)", top.urlStr, found)
	}
}

func TestConcurrentQueueInterleave(t *testing.T) {
	// very basic test to throw a lot of concurrent operations at a queue

	var wg sync.WaitGroup
	q := HyperlinkQueue{}

	// random selection of push, pop and len operations
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(num int) {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				q.Push(Hyperlink{"TEST", 0})
			}
		}(i)
	}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(num int) {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				q.Pop()
			}
		}(i)
	}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(num int) {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				q.Len()
			}
		}(i)
	}

	wg.Wait()
}
