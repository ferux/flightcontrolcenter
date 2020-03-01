package pubsub

import "sync"

type Topic string

type Handler func(args ...interface{})

// Core stores subscribers for each event
type Core struct {
	subs map[Topic][]Handler

	mu sync.RWMutex
}

func New() *Core {
	return &Core{}
}

// Subscribe handler to specified topic.
func (c *Core) Subscribe(topic Topic, h Handler) {
	c.mu.Lock()
	defer c.mu.RUnlock()

	hs, ok := c.subs[topic]
	if !ok {
		hs = []Handler{h}
		c.subs[topic] = hs

		return
	}

	hs = append(hs, h)
	c.subs[topic] = hs
}

// Notify subscribers.
func (c *Core) Notify(topic Topic, args ...interface{}) {
	hs := c.subs[topic]
	for _, h := range hs {
		h(args...)
	}
}
