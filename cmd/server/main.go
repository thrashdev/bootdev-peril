package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/thrashdev/bootdev-peril/internal/gamelogic"
	"github.com/thrashdev/bootdev-peril/internal/pubsub"
	"github.com/thrashdev/bootdev-peril/internal/routing"
)

func main() {
	fmt.Println("Starting Peril server...")
	gamelogic.PrintServerHelp()
	conn_string := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(conn_string)
	if err != nil {
		log.Println("Crashed on startup")
		log.Fatal(err)
	}
	defer conn.Close()
	fmt.Println("Successfully connected to RabbitMQ")

	amqpChan, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	gameLogKey := routing.GameLogSlug + ".*"
	_, _, err = pubsub.DeclareAndBind(conn, routing.ExchangePerilTopic, routing.GameLogSlug, gameLogKey, pubsub.DurableQueue)
	if err != nil {
		log.Fatal(err)
	}
	stop := false
	for stop == false {
		input := gamelogic.GetInput()
		fmt.Println(input)
		switch input[0] {
		case "pause":
			fmt.Println("Pausing...")
			err = pubsub.PublishJSON(amqpChan, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: true})
			if err != nil {
				log.Println("Error during publishing JSON")
				log.Fatal(err)
			}
		case "resume":
			fmt.Println("Resuming...")
			err = pubsub.PublishJSON(amqpChan, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{IsPaused: false})
			if err != nil {
				log.Println("Error during publishing JSON")
				log.Fatal(err)
			}
		case "quit":
			fmt.Println("Exiting...")
			stop = true
			break
		}
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
	fmt.Println("Received interrupt, shutting down...")
}
