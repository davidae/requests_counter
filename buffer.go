// A simple web server that counts and returns the number of requests the past 60 seconds.
// by using a circular buffer.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"time"
)

var (
	output = flag.String("output-file", "buffer_counter.json", "Filename of JSON that stores request data for restarts.")
	debug  = flag.Bool("debug", false, "More verbose information about the buffer cycle.")
)

// Counter is counter struct
type Counter struct {
	TimeRequests  []int64   `json:"time_requests"`
	CurrentSecond int       `json:"current_second"`
	LastUpdate    time.Time `json:"last_update"`
}

// Store number of requests in the right position in the cycle buffer TimeRequests
func (c *Counter) Store(second int, value int64) {
	if c.CurrentSecond == second {
		c.TimeRequests[second] += value
	} else {
		c.CurrentSecond = second
		c.TimeRequests[second] = value
	}

	c.LastUpdate = time.Now()
}

// Sum is the number of requests the past 60 seconds
func (c *Counter) Sum() int64 {
	sum := int64(0)

	for _, value := range c.TimeRequests {
		sum += value
	}

	return sum
}

// To ensure that the last 60 seconds of requests adhere to "real life"
// time and not server uptime.
func refreshCounter(c *Counter) {
	currentTime := time.Now()
	diff := currentTime.Sub(c.LastUpdate) // delta in Nanoseconds

	// Past 60 seconds in real time, no need to store current values in Counter
	if diff >= 60*time.Second {
		for i := 0; i < len(c.TimeRequests); i++ {
			c.TimeRequests[i] = 0
		}
		return
	}

	// Reset a certain time range `diff` seconds of the rolling window
	indexs := int(diff / time.Second) // nanosec to second
	fromBack := len(c.TimeRequests) - 1

	for i := 0; i < indexs; i++ { // Remove i Seconds "behind"" current second
		if currentTime.Second()-i >= 0 { // Since it's a rolling window, jumping from c.TimeRequests[0] to c.TimeRequests[59] might happen.
			c.TimeRequests[currentTime.Second()-i] = 0
		} else {
			c.TimeRequests[fromBack] = 0
			fromBack--
		}
	}
}

func debuBuffer(c *Counter) string {
	return strings.Join(strings.Fields(fmt.Sprint(c.TimeRequests)), ",")
}

func main() {
	flag.Parse()

	requestsBuffer := int64(0)
	counter, err := loadCounterFromJSON()

	if err != nil {
		fmt.Printf("Could not load counter from JSON: %v", err)
		counter = &Counter{TimeRequests: make([]int64, 60), CurrentSecond: 0}
	}

	flushTicker := time.NewTicker(10 * time.Millisecond)
	stopChan := make(chan os.Signal)

	signal.Notify(stopChan, os.Interrupt)

	http.HandleFunc("/counter", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&requestsBuffer, 1)
		fmt.Fprintf(w, "There has been %d requests in the last 60 seconds.\n", counter.Sum())

		if *debug {
			fmt.Fprintf(w, "Buffer: %s", debuBuffer(counter))
		}
	})

	go http.ListenAndServe(":8080", nil)

	for {
		select {
		case <-stopChan:
			if err := storeCounterToJSON(counter); err != nil {
				fmt.Printf("Could not store request data to JSON: %v", err)
			}
			return
		case <-flushTicker.C:
			counter.Store(time.Now().Second(), requestsBuffer)
			atomic.StoreInt64(&requestsBuffer, 0)
		}
	}
}

func storeCounterToJSON(c *Counter) error {
	// Marshal JSON
	counterJSON, err := json.Marshal(c)
	if err != nil {
		return err
	}
	// Write JSON
	return ioutil.WriteFile(*output, counterJSON, 0664)
}

func loadCounterFromJSON() (*Counter, error) {
	// Read JSON
	rawJSON, err := ioutil.ReadFile(*output)
	if err != nil {
		return nil, err
	}

	// Unmarshal JSON
	counter := &Counter{}
	if err := json.Unmarshal(rawJSON, counter); err != nil {
		return nil, err
	}
	refreshCounter(counter)

	return counter, nil
}
