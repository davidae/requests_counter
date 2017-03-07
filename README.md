# Request Counter
A simple http web-server that updates and returns the number of requests the past 60 seconds.

## How to run
1. `go run main.go` or build it.
2. `curl http://localhost:8080/counter`
3. GOTO 2.

Also try out `go run buffer.go`

## Problem description
Create a Go HTTP server that on each request responds with a counter of the total number of requests that it has received during the last 60 seconds.

The server should continue to the return the correct numbers after restarting it, by persisting data to a file.

## Notes on the implementation
I chose to store the number of requests on a `map[string]*TimeEntry`, first the key needed to be a string and not an `int` (json).
The key are based on seconds, with a range from 0 to 59, which will keep the size of the datastrucure small and constant over a longer period of time. The "downside" was that I needed to validate that the value for a given key, that it was done THIS minute, hour and years, i.e. if the value was stale (which can f.ex. occur if there was requests at 18:22:22 but not at 18:23:22, we should not count the value at key `"22"` @ 18:23:23 for the total number of requests the past 60 seconds).

I chose to write the JSON file after each request ensuring persistence and as up-to-date copy as possible. IO operations can be costly, but this is only writes (and overwrites) and it is in a goroutine, so it doesn't take up time for a response to the user.

### Other implementation options I evaluated
#### Circular buffer and polling
Update: I added `buffer.go` as a proof-of-concept of this. It works just as `main.go` only it considers "last 60 seconds" as last 60 seconds the server was running, not real life as `main.go`.

I also considered using a [circular buffer](https://en.wikipedia.org/wiki/Circular_buffer) slice with a size of 60 (one for each second) and utilizing `time.newTicker`. Having a buffer temporarily holding for holding all requests coming in and flushing in them into the circular buffer each second using `time.NewTicker(10)` for timing. This would remove need to validate if it was stale or not. I chose not to due to the flush being only each second (or less), which could cause it not to be 100% accuracy, and the constant polling.

#### JSON write only on shutdown
Another option I considered was only writing to a JSON file when shutting down the server, by f.ex.
```
func main() {
  stopChan := make(chan os.Signal)
  signal.Notify(stopChan, os.Interrupt)
  ...
  ...
  ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
  server.Shutdown(ctx)
  counter.writeToJson()
```
I chose not to due to persistence of writing more, my solution already works fine with a goroutine (time/performance is not an issue) and a server/application isn't always "gracefully" shut down.

