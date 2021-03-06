package main

import (
	"flag"
	"github.com/emanueljoivo/arrebol/arrebol"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emanueljoivo/arrebol/api"
	"github.com/emanueljoivo/arrebol/storage"
	"github.com/joho/godotenv"
)

func main() {
	const ServerPort = "SERVER_PORT"
	const DefaultServerPort = "8080"

	var wait time.Duration
	flag.DurationVar(&wait, "graceful_timeout", time.Second*15, "the duration for which the server "+
		"gracefully wait for existing connections to finish - e.g. 15s or 1m")

	apiPort := flag.String(ServerPort, DefaultServerPort, "Service port")

	flag.Parse()

	err := godotenv.Load()

	if err != nil {
		log.Println("No .env file found")
	}

	s := storage.NewDB(os.Getenv("DATABASE_ADDRESS"), os.Getenv("DATABASE_PORT"),
		os.Getenv("DATABASE_NAME"), os.Getenv("DATABASE_PASSWORD"), os.Getenv("DATABASE_USER"))
	s.SetUp()
	defer s.Driver().Close()

	var jobDispatcher = arrebol.NewDispatcher(s)
	go jobDispatcher.Start()

	a := api.New(s, jobDispatcher)

	// Shutdown gracefully
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
		<-sigs
		log.Println("Shutting down service")

		if err := a.Shutdown(); err != nil {
			log.Printf("Failed to shutdown the server: %v", err)
		}
	}()

	if err := a.Start(*apiPort); err != nil {
		log.Fatal(err.Error())
	}
}
