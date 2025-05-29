package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"redis-clone/internal/server"
)

func main() {
	port := flag.String("port", "6379", "Port to run the Redis server on")
	config := flag.String("config", "redis.conf", "Configuration file path")
	flag.Parse()

	srv := server.NewServer(*config)

	// Set up graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		fmt.Printf("Redis server starting on port %s\n", *port)
		if err := srv.Start(*port); err != nil {
			serverErr <- err
		}
	}()

	// Wait for either shutdown signal or server error
	select {
	case <-c:
		fmt.Println("\nReceived shutdown signal...")
	case err := <-serverErr:
		log.Printf("Server error: %v", err)
	}

	fmt.Println("Shutting down Redis server...")
	srv.Shutdown()
}
