package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/ragwolf/tasksync"
)

var port = ":8080"
var timeout = time.Duration(500) * time.Millisecond

func main() {
	// GroupedTaskSync is used so all requests for the same ID use the same task
	gts := tasksync.NewGroupedTaskSync[string, string]()

	http.HandleFunc("GET /example/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		log.Printf("Received request for ID: %s", id)

		ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
		defer cancelFunc()

		output, err := gts.Run(ctx, id, func(ctx context.Context) (string, error) {
			log.Printf("Starting task for ID: %s", id)
			return calculateSomething(ctx, id)
		})
		if err != nil {
			log.Printf("Error processing request for ID %s: %v", id, err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		log.Printf("Task completed for ID: %s", id)
		w.Write([]byte(output))
	})

	// Start the HTTP server in a goroutine
	go func() {
		log.Printf("Server started on %s\n", port)
		log.Fatal(http.ListenAndServe(port, nil))
	}()

	// Give the server a moment to start
	time.Sleep(1 * time.Second)

	// Make requests to the server
	var wg sync.WaitGroup

	for id := 1; id <= 10; id++ {
		url := fmt.Sprintf("http://localhost%s/example/%d", port, id)

		// Make 10 requests for each ID6a
		for idx := 0; idx < 10; idx++ {
			wg.Add(1)

			go func(url string, id int) {
				defer wg.Done()

				// Make the HTTP request
				resp, err := http.Get(url)
				if err != nil {
					log.Printf("Error making request to %s: %v", url, err)
				}
				defer resp.Body.Close()

				// Read the response body
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					log.Printf("Error reading response body from %s: %v", url, err)
				}

				// Log the response
				log.Printf("Response from ID %d: %s", id, string(body))
			}(url, id)
		}
	}

	// Wait for all requests to complete
	wg.Wait()
}

func calculateSomething(ctx context.Context, id string) (string, error) {
	select {
	case <-time.After(time.Duration(400) * time.Millisecond):
		return fmt.Sprintf("%s response", id), nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}
