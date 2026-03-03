package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Log headers once for debugging
		if r.Header.Get("Authorization") != "" {
			fmt.Printf("Received Header Authorization: %s\n", r.Header.Get("Authorization"))
		}

		fmt.Printf("Received Request URL: %s\n", r.URL.String())

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	fmt.Println("Test server listening on :8080")
	http.ListenAndServe(":8080", nil)
}
