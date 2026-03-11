package sse

// Client represents a single SSE client connection
type Client struct {
	hub      *Hub
	streamID string
	send     chan Message
	done     chan struct{}
}

// Message represents an SSE message to be sent
type Message struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

// NewClient creates a new SSE client.
func NewClient(hub *Hub, streamID string) *Client {
	return &Client{
		hub:      hub,
		streamID: streamID,
		send:     make(chan Message, 256),
		done:     make(chan struct{}),
	}
}

// Close signals the client to stop
func (c *Client) Close() {
	close(c.done)
}
