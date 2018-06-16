# Keen IO Golang Client SDK

### Community-Supported SDK
This is an _unofficial_ community supported SDK. If you find any issues or have a request please post an [issue](https://github.com/Secoura/go-keen/issues).

## Writing Events

Currently, only adding events to collections is supported.

A queue has been implemented for production batch sending of events. An example of how to do this is documented below:

```go
package main

import (
	"time"
	"github.com/secoura/go-keen"
)

const (
	KEEN_IO_PROJECT_ID = "XXX"
	KEEN_IO_WRITE_KEY = "YYY"
)

func main() {
	client := keen.New(KEEN_IO_PROJECT_ID, KEEN_IO_WRITE_KEY)
	client.Size = 10 // Send batch after 10 events in the queue
	client.Interval = time.Second // Send batch after 1 second (if it's not empty)

	client.Event("foo_collection", map[string]interface{}{
		"user_id": 123,
		"event": "viewed_product",
		"product_id": 91873,
	})

	// Don't do this in production; this is just to trigger the loop at client.Interval
	time.Sleep(time.Second * 5)
}
```

## TODO
Add support for all other Keen IO API endpoints, especially querying data.

## LICENSE
MIT
