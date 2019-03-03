package sse

import (
	"fmt"
	"log"
	"net/http"
	"sync"
)

// Server will create the http handlers
type Server struct {
	EventStreams map[string]*EventStream
	lock         sync.Mutex
}

// NewServer creates a new server isntance
func NewServer() *Server {
	return &Server{
		EventStreams: make(map[string]*EventStream, 0),
	}
}

func (s *Server) streamExists(topic string) bool {
	return s.EventStreams[topic] != nil
}

func (s *Server) createStream(topic string) *EventStream {
	s.lock.Lock()
	defer s.lock.Unlock()
	e := newEventStream(topic)
	s.EventStreams[topic] = e
	e.Run()
	return e
}

// getCreateStream returns a stream for a topic. The stream is created if it does not exist
func (s *Server) getCreateStream(topic string) *EventStream {
	if s.streamExists(topic) {
		return s.EventStreams[topic]
	}
	return s.createStream(topic)
}

// Publish sends an event to a stream of the specified topic
func (s *Server) Publish(topic string, event *Event) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.streamExists(topic) {
		s.EventStreams[topic].events <- event
	}
}

// MakeHandler generates a http handler function
// eventTopic is a function that is used to determine the topic of the event stream from an http request
func (s *Server) MakeHandler(eventTopic func(*http.Request) (string, error)) func(http.ResponseWriter, *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		// make sure that the writer supports flushing
		flusher, supported := w.(http.Flusher)
		if !supported {
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// try to get event topic from eventTopic function passed
		topic, err := eventTopic(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		// check if we have an event stream for the topic
		stream := s.getCreateStream(topic)
		// get our client
		client := stream.AddClient()

		// Listen to connection close and un-register client
		notify := w.(http.CloseNotifier).CloseNotify()
		go func() {
			<-notify
			stream.unsubscribe <- client
		}()

		// push events to client
		for {
			event, ok := <-client.events
			if !ok {
				return
			}
			log.Printf("%s", event.Data)
			if len(event.Data) == 0 {
				break
			}

			fmt.Fprint(w, event.FormatEvent())
			flusher.Flush()
		}
	}
}
