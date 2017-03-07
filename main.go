// A simple web server that counts and returns the number of requests the past X seconds.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

var (
	requestsCountDuration = flag.Duration("requests-last-seconds", 60*time.Second, "The number of requests last X seconds")
	jsonOutputFile        = flag.String("output-file", "counter.json", "Filename of JSON that stores request data for restarts")
)

func main() {
	http.Handle("/counter", &CounterHandler{counter: newCounter(true)})
	http.ListenAndServe(":8080", nil)
}

// newCounter returns a Counter struct with either existing data, loaded from `counter.json` if it is present, or
// a new instantiated Counter struct with no existing data.
func newCounter(useJSON bool) *Counter {
	if useJSON {
		return getCounterFromJSON()
	}

	return &Counter{TimeEntries: make(map[string]*TimeEntry), useJSON: false}
}

// getCounterFromJSON tries to use the json data to create a Counter struct with
// existing data. If it fails, it will return a new Counter struct with no initial data.
func getCounterFromJSON() *Counter {
	counter := &Counter{}
	rawJSON := readCounterDataFromJSON()

	if rawJSON != nil {
		err := json.Unmarshal(rawJSON, counter)

		if err != nil {
			fmt.Errorf("Error. Could not Unmarshal JSON: %v", err)
		} else {
			return counter
		}
	}

	return &Counter{TimeEntries: make(map[string]*TimeEntry), useJSON: true}
}

// CounterHandler is the handler for the `/counter` endpoint
type CounterHandler struct {
	counter *Counter
}

// Counter counts and keeps track of all requests
type Counter struct {
	TimeEntries map[string]*TimeEntry `json:"time_enries"`
	useJSON     bool
}

// A TimeEntry tracks number of requests for a given time.
type TimeEntry struct {
	Value      int       `json:"value"`
	LastUpdate time.Time `json:"last_update"`
}

// Increments the value of a TimeEntry by 1. It will not increment a stale TimeEntry,
// it will set the value to 1 and update the timestamp to make it fresh again.
func (t *TimeEntry) increment() {
	if t.isStale() {
		t.Value, t.LastUpdate = 1, time.Now()
	} else {
		t.Value++
	}
}

// Update the entry times for a give time (0-59 second timestamp). It will
// increment an existing TimeEntry struct or initialize a new with value 1
// if it was not already in the map.
func (c *Counter) updateTimeEntries(time time.Time) {
	second := strconv.Itoa(time.Second())

	entry, ok := c.TimeEntries[second]
	if !ok {
		c.TimeEntries[second] = &TimeEntry{Value: 1, LastUpdate: time}
	} else {
		entry.increment()
	}
}

// Returns the number of request in the past X seconds, where X is the flag
// `requestsCountDuration`. Default is 60 seconds.
func (c *Counter) numberOfRequests() int {
	sum := 0

	for _, entry := range c.TimeEntries {
		if !entry.isStale() {
			sum += entry.Value
		}
	}
	return sum
}

// Returns true if the TimeEntry was not modified during the last X seconds ago,
// where X is the flag `requestsCountDuration`, the default is 60 seconds.
func (t *TimeEntry) isStale() bool {
	return time.Since(t.LastUpdate) > *requestsCountDuration
}

// Handler updating and returning the number of requests.
func (h *CounterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.counter.updateTimeEntries(time.Now())

	if h.counter.useJSON {
		go h.counter.WriteToJSON()
	}

	fmt.Fprintf(w, "There has been %d requests in the last %d seconds.", h.counter.numberOfRequests(), *requestsCountDuration/time.Second)
}

// WriteToJSON reads the current Counter and time entries data to a JSON
// file, `counter.json` is default.
func (c *Counter) WriteToJSON() {
	counterJSON, err := json.Marshal(c)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}

	err = ioutil.WriteFile(*jsonOutputFile, counterJSON, 0664)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}
}

// readCounterDataFromJSON reads the file `counter.json` in the current directory and returns
// the raw data if the file is available, otherwise it returns `nil`.
func readCounterDataFromJSON() []byte {
	json, err := ioutil.ReadFile(*jsonOutputFile)

	if err != nil {
		fmt.Printf("Could not load JSON. Using new counter data.")
		return nil
	}

	return json
}
