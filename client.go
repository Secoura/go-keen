package keen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const (
	baseUrl = "https://api.keen.io/3.0/projects/"
)

type batchMessages struct {
	objectType string
	messages   map[string][]interface{}
}

func (b *batchMessages) AddEvent(collection string, properties interface{}) {
	list, ok := b.messages[collection]
	if !ok {
		list = make([]interface{}, 0)
	}
	b.messages[collection] = append(list, properties)
}

type message struct {
	objectType     string
	collectionName string
	properties     interface{}
}

type Client struct {
	Interval   time.Duration
	Size       int
	HttpClient http.Client

	writeKey  string
	projectID string
	msgs      chan message
	quit      chan struct{}
	shutdown  chan struct{}
	mu        sync.Mutex
	once      sync.Once
	wg        sync.WaitGroup
}

func New(projectID string, writeKey string) *Client {
	return &Client{
		Interval: 5 * time.Second,
		Size:     250,

		projectID: projectID,
		writeKey:  writeKey,
		msgs:      make(chan message, 100),
		quit:      make(chan struct{}),
		shutdown:  make(chan struct{}),
	}
}

func (c *Client) Event(collection string, event interface{}) {
	c.queue(message{objectType: "events", collectionName: collection, properties: event})
}

func (c *Client) startLoop() {
	go c.loop()
}

func (c *Client) loop() {
	var msgs []message
	tick := time.NewTicker(c.Interval)

	for {
		select {
		case msg := <-c.msgs:
			msgs = append(msgs, msg)
			if len(msgs) == c.Size {
				c.sendAsync(msgs)
				msgs = make([]message, 0, c.Size)
			}
		case <-tick.C:
			if len(msgs) > 0 {
				c.sendAsync(msgs)
				msgs = make([]message, 0, c.Size)
			}
		case <-c.quit:
			tick.Stop()
			// drain msg channel
			for msg := range c.msgs {
				msgs = append(msgs, msg)
			}
			c.sendAsync(msgs)
			c.wg.Wait()
			c.shutdown <- struct{}{}
			return
		}
	}
}

func (c *Client) queue(msg message) {
	c.once.Do(c.startLoop)
	c.msgs <- msg
}

func (c *Client) sendAsync(msgs []message) {
	c.mu.Lock()
	c.wg.Add(1)

	go func() {
		defer c.wg.Done()
		defer c.mu.Unlock()
		c.send(msgs)
	}()
}

func (c *Client) send(msgs []message) error {
	// Sort messages into groups
	msgMap := make(map[string]batchMessages, 0)
	for _, m := range msgs {
		object := m.objectType
		batch, ok := msgMap[object]
		if !ok {
			batch = batchMessages{objectType: object, messages: make(map[string][]interface{})}
			msgMap[object] = batch
		}
		batch.AddEvent(m.collectionName, m.properties)
	}

	// Send each group
	for object, batch := range msgMap {
		for collection, properties := range batch.messages {
			c.request("POST", fmt.Sprintf("/%s", object), map[string]interface{}{collection: properties})
		}
	}

	return nil
}

func (c *Client) request(method, path string, payload interface{}) (*http.Response, error) {
	// serialize payload
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// construct url
	url := baseUrl + c.projectID + path

	// new request
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	// add auth
	req.Header.Add("Authorization", c.writeKey)

	// set length/content-type
	if body != nil {
		req.Header.Add("Content-Type", "application/json")
		req.ContentLength = int64(len(body))
	}

	return c.HttpClient.Do(req)
}
