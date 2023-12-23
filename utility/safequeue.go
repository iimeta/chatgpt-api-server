package utility

import "sync"

type SafeQueue struct {
	queue map[string]struct{}
	order []string
	mu    sync.Mutex
}

func NewSafeQueue() *SafeQueue {
	return &SafeQueue{
		queue: make(map[string]struct{}),
		order: make([]string, 0),
	}
}

func (sq *SafeQueue) Add(value string) {
	sq.mu.Lock()
	defer sq.mu.Unlock()

	// 如果元素已经存在，直接返回
	if _, ok := sq.queue[value]; ok {
		return
	}

	sq.queue[value] = struct{}{}
	sq.order = append(sq.order, value)
}
func (sq *SafeQueue) Pop() (value string, ok bool) {
	sq.mu.Lock()
	defer sq.mu.Unlock()

	if len(sq.order) == 0 {
		return "", false
	}

	value = sq.order[0]
	if len(sq.order) == 1 {
		sq.order = []string{}
	} else {
		sq.order = sq.order[1:]
	}
	delete(sq.queue, value)

	return value, true
}
func (sq *SafeQueue) Remove(value string) {
	sq.mu.Lock()
	defer sq.mu.Unlock()

	// 如果元素不存在，直接返回
	if _, ok := sq.queue[value]; !ok {
		return
	}

	// 从map中移除元素
	delete(sq.queue, value)

	// 从order中移除元素
	for i, v := range sq.order {
		if v == value {
			sq.order = append(sq.order[:i], sq.order[i+1:]...)
			break
		}
	}
}
func (sq *SafeQueue) Size() int {
	sq.mu.Lock()
	defer sq.mu.Unlock()

	return len(sq.order)
}
