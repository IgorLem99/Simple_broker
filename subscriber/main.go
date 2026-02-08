package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func main() {
	resp, err := http.Post("http://localhost:8080/queues/app_events/subscriptions", "", nil)
	if err != nil {
		fmt.Printf("Error subscribing: %v\n", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	fmt.Println("Subscribed to app_events. Waiting for messages...")

	dec := json.NewDecoder(resp.Body)
	for {
		var msg interface{}
		if err := dec.Decode(&msg); err != nil {
			fmt.Printf("Error decoding message: %v\n", err)
			break
		}
		fmt.Printf("Received message: %+v\n", msg)
	}
}
