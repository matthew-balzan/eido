package utils

import (
	"context"
	"errors"
	"math/rand/v2"
	"sync"
)

var (
	// QueueFullErr is returned when attempting to push an item in an already full queue.
	QueueFullErr = errors.New("queue is full")
)

// RingQueue is a FIFO queue implemented using a ring buffer.
//
// See https://en.wikipedia.org/wiki/Circular_buffer.
// All methods are concurrency safe.
type RingQueue[T any] struct {
	lock     sync.Mutex
	notify   chan struct{}
	buf      []T
	len, cap int
	r, w     int
}

// NewRingQueue creates a new queue with a capacity of size elements.
func NewRingQueue[T any](size int) *RingQueue[T] {
	return &RingQueue[T]{
		notify: make(chan struct{}),
		buf:    make([]T, size),
		cap:    size,
	}
}

// Len returns the current number of items in the queue.
func (q *RingQueue[T]) Len() int {
	return q.len
}

// Cap returns the maximum capacity of the queue.
func (q *RingQueue[T]) Cap() int {
	return q.cap
}

// Push pushes an item at the end of the queue.
//
// QueueFullErr is returned when the queue is full (Len == Cap).
func (q *RingQueue[T]) Push(item T) error {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.len >= q.cap {
		return QueueFullErr
	}

	q.buf[q.w] = item

	q.w = (q.w + 1) % q.cap
	q.len++

	// Notify subscribers of a new item in the queue.
	// If there are already items in the queue, nobody must be waiting for a new item. Skip notify.
	if q.len <= 1 {
		select {
		case q.notify <- struct{}{}:
			break
		default:
			// Ignore when nobody is waiting.
		}
	}

	return nil
}

// Pop retrieves and removes the first item from the queue.
//
// False is returned when no items are in the queue.
func (q *RingQueue[T]) Pop() (item T, ok bool) {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.len <= 0 {
		return
	}

	ok = true
	item = q.buf[q.r]

	// Zero the popped item in case it is a pointer, so the target object can be freed.
	q.buf[q.r] = *new(T)

	q.r = (q.r + 1) % q.cap
	q.len--

	return
}

// Accept blocks until an item is available in the queue, or the context is done.
//
// If an item is available, err is nil.
// Otherwise, err is the context termination reason.
func (q *RingQueue[T]) Accept(ctx context.Context) (item T, err error) {
	for {
		// Attempt to pop from queue.
		if i, ok := q.Pop(); ok {
			return i, nil
		}

		// TODO: Possible infinite accept.
		// If listening coroutine is here (after pop but before select),
		// and push on empty queue is executed completely, we are never notified.
		// This can be mitigated by always notify on push, or by introducing a timer on notify listen.
		// Race window is small enough to not worry about it.

		select {
		case <-q.notify:
			// We have been notified, retry.
			continue
		case <-ctx.Done():
			// Context is done, time to go.
			err = ctx.Err()
			return
		}
	}
}

// PeekAll returns a copy of all items in the queue.
func (q *RingQueue[T]) PeekAll() []T {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.len <= 0 {
		return nil
	}

	list := make([]T, q.len)

	for i := 0; i < q.len; i++ {
		list[i] = q.buf[(q.r+i)%q.cap]
	}

	return list
}

// Clear removes all items in the queue.
func (q *RingQueue[T]) Clear() {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.len <= 0 {
		return
	}

	q.buf = make([]T, q.cap)
	q.len = 0
	q.r = 0
	q.w = 0

	return
}

// Shuffle randomly shuffles the order of the items in the queue.
func (q *RingQueue[T]) Shuffle() {
	q.lock.Lock()
	defer q.lock.Unlock()

	if q.len < 2 {
		return
	}

	rand.Shuffle(q.len, func(i, j int) {
		ir := (q.r + i) % q.cap
		jr := (q.r + j) % q.cap

		q.buf[ir], q.buf[jr] = q.buf[jr], q.buf[ir]
	})
}
