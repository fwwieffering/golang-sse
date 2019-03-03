package sse

import "log"

// EventStream must:
// register / deregister clients
// keep track of clients to publish messages to
// close clients that have been closed
type EventStream struct {
	// Topic is a way to group events
	Topic string
	// Events are pushed to this channel by event producers
	events chan *Event
	// client subscription events are pushed to this channel
	subscribe chan Client
	// client removal events are pushed to this channel
	unsubscribe chan Client
	// clients registry. Each client is a channel of byte arrays
	clients map[Client]bool
	// stream history stores the history of event streams published to MessageStream
	streamHistory []*Event
	// shutdown receives a shutdown signal
	shutdown chan bool
}

func newEventStream(topic string) *EventStream {
	return &EventStream{
		Topic:         topic,
		events:        make(chan *Event),
		subscribe:     make(chan Client),
		unsubscribe:   make(chan Client),
		clients:       make(map[Client]bool),
		streamHistory: make([]*Event, 0),
		shutdown:      make(chan bool),
	}
}

// AddClient registers a client with the event stream
func (e *EventStream) AddClient() Client {
	c := newClient()
	e.subscribe <- c
	// TODO: write streamHistory to client channel
	return c
}

// RemoveClient deregisters a client from the stream
func (e *EventStream) removeClient(c Client) {
	close(c.events)
	delete(e.clients, c)
}

func (e *EventStream) removeAllClients() {
	for c := range e.clients {
		e.removeClient(c)
	}
}

func (e *EventStream) close() {
	e.shutdown <- true
}

// BroadcastMessage publishes a message to all clients
func (e *EventStream) broadcast(msg *Event) {
	// write message to the stream history
	e.streamHistory = append(e.streamHistory, msg)
	for c := range e.clients {
		// TODO: format message for SSE `data:%s`
		c.events <- msg
	}
}

// Run listens to events and handles them
func (e *EventStream) Run() {
	go func(ev *EventStream) {
		for {
			select {
			// broadcast message to subscriber
			case msg := <-ev.events:
				ev.broadcast(msg)
				// remove subscriber
			case removeClient := <-ev.unsubscribe:
				ev.removeClient(removeClient)
			case addClient := <-ev.subscribe:
				ev.clients[addClient] = true
				log.Printf("Client added for topic %s: %d reglistered clients\n", e.Topic, len(ev.clients))
			case _ = <-ev.shutdown:
				ev.removeAllClients()
			}
		}
	}(e)
}
