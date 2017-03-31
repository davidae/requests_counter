# Request Counter
A simple http web-server that updates and returns the number of requests the past 60 seconds.

## How to run
1. `go run buffer.go` or build it.
2. `curl http://localhost:8080/counter`
3. GOTO 2.

## Problem description
Create a Go HTTP server that on each request responds with a counter of the total number of requests that it has received during the last 60 seconds.

The server should continue to the return the correct numbers after restarting it, by persisting data to a file.