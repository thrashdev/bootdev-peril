package main

import (
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/thrashdev/bootdev-peril/internal/gamelogic"
	"github.com/thrashdev/bootdev-peril/internal/pubsub"
	"github.com/thrashdev/bootdev-peril/internal/routing"
)

func main() {
	fmt.Println("Starting Peril client...")
	conn_string := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(conn_string)
	if err != nil {
		log.Println("Crashed on startup")
		log.Fatal(err)
	}
	defer conn.Close()

	fmt.Println("Successfully connected to RabbitMQ")
	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Println("Crashed on Welcome")
		log.Fatal(err)
	}
	queueName := fmt.Sprintf("%s.%s", routing.PauseKey, username)
	_, _, err = pubsub.DeclareAndBind(conn, "peril_direct", queueName, routing.PauseKey, pubsub.TransientQueue)
	if err != nil {
		log.Fatal("Crashed on declaring the queue and binding")
	}
	gameState := gamelogic.NewGameState(username)

	stop := false
	for stop == false {
		words := gamelogic.GetInput()
		command := words[0]
		switch command {
		case "spawn":
			err := gameState.CommandSpawn(words)
			if err != nil {
				log.Fatal(err)
			}
		case "move":
			gameState.CommandMove(words)
		case "status":
			gameState.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			fmt.Println("Spamming not allowed yet!")
		case "quit":
			gamelogic.PrintQuit()
			stop = true
			break
		default:
			fmt.Printf("%s is not a recognized command.\n", command)
		}

	}

	fmt.Println("Press Enter to continue...")
	fmt.Scanln()

}
