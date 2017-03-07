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
	"sync/atomic"
	"time"
)

var (
	jsonOutputFile = flag.String("output-file", "buffer_counter.json", "Filename of JSON that stores request data for restarts")
	requestBuffer  int64
)

type Counter struct {
	TimeRequests  []int64   `json:"time_requests"`
	CurrentSecond int       `json:"current_second"`
	LastUpdate    time.Time `json:"last_update"`
}

func (c *Counter) Store(second int, value int64) {
	if c.CurrentSecond == second {
		c.TimeRequests[second] += value
	} else {
		c.CurrentSecond = second
		c.TimeRequests[second] = value
	}

	c.LastUpdate = time.Now()
}

func (c *Counter) Sum() int64 {
	sum := int64(0)

	for _, value := range c.TimeRequests {
		sum += value
	}

	return sum
}

func main() {
	counter := loadCounterFromJSON()

	flushTicker := time.NewTicker(10 * time.Millisecond)
	stopChan := make(chan os.Signal)

	signal.Notify(stopChan, os.Interrupt)

	http.HandleFunc("/counter", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&requestBuffer, 1)
		fmt.Fprintf(w, "There has been %d requests in the last 60 seconds.", counter.Sum())
	})

	go http.ListenAndServe(":8080", nil)

	for {
		select {
		case <-stopChan:
			storeCounterToJSON(counter)
			return
		case <-flushTicker.C:
			counter.Store(time.Now().Second(), requestBuffer)
			atomic.StoreInt64(&requestBuffer, 0)
		}
	}
}

func storeCounterToJSON(c *Counter) {
	// Marshal JSON
	counterJSON, err := json.Marshal(c)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}

	// Write JSON
	err = ioutil.WriteFile(*jsonOutputFile, counterJSON, 0664)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}
}

func loadCounterFromJSON() *Counter {
	// Read JSON
	rawJSON, err := ioutil.ReadFile(*jsonOutputFile)
	if err != nil {
		fmt.Printf("Error. Could not find JSON. Using new Counter.")
		return &Counter{TimeRequests: make([]int64, 60), CurrentSecond: 0}
	}

	// Unmarshal JSON
	counter := &Counter{}
	err = json.Unmarshal(rawJSON, counter)

	if err != nil {
		fmt.Printf("Error. Could not Unmarshal JSON. Using new Counter.")
		return &Counter{TimeRequests: make([]int64, 60), CurrentSecond: 0}
	}

	return counter
}