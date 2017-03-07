package main

import (
	"github.com/bouk/monkey"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Test<Function under test>ToReturn<Expected output><Conditions>
func TestCounterHandlerHealthCheck(t *testing.T) {
	handler, req := setUpHandlerAndRequest()

	doRequestsToHandler(req, 60, handler)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expectedBody := "There has been 61 requests in the last 60 seconds."
	if rr.Body.String() != expectedBody {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expectedBody)
	}
}

func TestCounterHandlerToReturn21RequestCountAfterLongerTimeWithMoreRequests(t *testing.T) {
	handler, req := setUpHandlerAndRequest()

	wayback := time.Date(2017, time.May, 17, 19, 20, 10, 0, time.UTC)
	monkey.Patch(time.Now, func() time.Time { return wayback })

	// Do 30 requests @ 2017-05-17 19:20:10.00
	doRequestsToHandler(req, 30, handler)

	wayback = time.Date(2017, time.May, 17, 19, 21, 11, 0, time.UTC)
	monkey.Patch(time.Now, func() time.Time { return wayback })

	// Do 20 requests @ 2017-05-17 19:21:11.00
	doRequestsToHandler(req, 20, handler)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	expectedBody := "There has been 21 requests in the last 60 seconds."
	if rr.Body.String() != expectedBody {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expectedBody)
	}
}

// setUpHandlerAndRequest is a helper method to return a request
// type to the correct endoint and a handler for the endpoint
func setUpHandlerAndRequest() (*CounterHandler, *http.Request) {
	handler := &CounterHandler{counter: newCounter(false)} // Do not write to json
	req, _ := http.NewRequest("GET", "/counter", nil)

	return handler, req
}

// doRequestsToHandler is a simple helper method to do X number of requests to a handler
func doRequestsToHandler(req *http.Request, noRequests int, handler *CounterHandler) {
	for i := 0; i < noRequests; i++ {
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}
}
