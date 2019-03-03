package sse

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type betterHttpRecorder struct {
	*httptest.ResponseRecorder
	closeNotifier chan bool
}

func (b betterHttpRecorder) CloseNotify() <-chan bool {
	return b.closeNotifier
}

func newRecorder() betterHttpRecorder {
	recorder := httptest.NewRecorder()
	better := betterHttpRecorder{recorder, make(chan bool)}
	return better
}

func TestNewServer(t *testing.T) {
	s := NewServer()
	if s == nil {
		t.Fatalf("didn't get a server")
	}
	if len(s.EventStreams) != 0 {
		t.Fatalf("expected a nil event stream")
	}
}

func TestCreateStream(t *testing.T) {
	s := NewServer()
	s.getCreateStream("foo")
	if len(s.EventStreams) != 1 {
		t.Fatalf("stream should be created when topic does not exist")
	}

	s.getCreateStream("foo")
	if len(s.EventStreams) != 1 {
		t.Fatalf("duplicate streams should not be created")
	}
}

func TestHTTP(t *testing.T) {
	s := NewServer()

	handler := s.MakeHandler(func(r *http.Request) (string, error) {
		streamID := r.URL.Query().Get("stream")
		if streamID == "" {
			return "", errors.New("Please specify a stream name as query param stream")
		}
		return streamID, nil
	})

	w := newRecorder()
	w2 := newRecorder()
	r := httptest.NewRequest("GET", "/?stream=foo", nil)

	// should add one subscriber and create stream foo
	go handler(w, r)
	time.Sleep(200 * time.Millisecond)
	if s.EventStreams["foo"] == nil {
		t.Fatalf("Should have created stream foo")
	}
	// should add a second client
	go handler(w2, r)
	time.Sleep(200 * time.Millisecond)
	if len(s.EventStreams["foo"].clients) != 2 {
		t.Fatalf("should be two")
	}
	// send a message
	s.Publish("foo", &Event{Data: "test data"})
	time.Sleep(200 * time.Millisecond)
	data1, err := ioutil.ReadAll(w.Body)
	if err != nil {
		t.Fatalf("Error reading response body: %s", err.Error())
	}
	data2, _ := ioutil.ReadAll(w2.Body)
	if string(data1) != "data: test data\n\n" || string(data1) != string(data2) {
		t.Fatalf("messages not equal expected value:\nexpected:test data\n1:%s\n2:%s", string(data1), string(data2))
	}
	// remove a client
	w2.closeNotifier <- true
	time.Sleep(200 * time.Millisecond)
	if len(s.EventStreams["foo"].clients) != 1 {
		t.Fatalf("should be one client")
	}
}
