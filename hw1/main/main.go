package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	dockerEnv := os.Getenv("DOCKER_ENV")

	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from docker env: %s\n", dockerEnv)
	})

	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
