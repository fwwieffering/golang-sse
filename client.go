package sse

type Client struct {
	events chan *Event
}

func newClient() Client {
	return Client{
		events: make(chan *Event),
	}
}

func (c *Client) close() {
	close(c.events)
}
