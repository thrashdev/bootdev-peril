package main

import (
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/thrashdev/bootdev-peril/internal/gamelogic"
	"github.com/thrashdev/bootdev-peril/internal/pubsub"
	"github.com/thrashdev/bootdev-peril/internal/routing"
)

func HandlerPause(gs *gamelogic.GameState) func(routing.PlayingState) {
	return func(ps routing.PlayingState) {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
	}
}

func HandlerMove(gs *gamelogic.GameState) func(move gamelogic.ArmyMove) {
	return func(move gamelogic.ArmyMove) {
		defer fmt.Print("> ")
		gs.HandleMove(move)
	}
}

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
	amqpChan, _, err := pubsub.DeclareAndBind(conn, routing.ExchangePerilDirect, queueName, routing.PauseKey, pubsub.TransientQueue)
	if err != nil {
		log.Fatal("Crashed on declaring:", routing.ExchangePerilDirect)
	}
	gameState := gamelogic.NewGameState(username)

	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilDirect, queueName, routing.PauseKey, pubsub.TransientQueue, HandlerPause(gameState))
	if err != nil {
		log.Println("Crashed at subscribing to:", queueName)
		log.Fatal(err)
	}

	armyMovesWildcard := routing.ArmyMovesPrefix + ".*"
	armyQueueName := routing.ArmyMovesPrefix + "." + username
	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, armyQueueName, armyMovesWildcard, pubsub.TransientQueue, HandlerMove(gameState))
	if err != nil {
		log.Println("Crashed at subscribing to:", armyQueueName)
		log.Fatal(err)
	}

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
			armyMove, err := gameState.CommandMove(words)
			if err != nil {
				log.Println("Error encountered during army move")
				continue
			}
			pubsub.PublishJSON(amqpChan, routing.ExchangePerilTopic, armyQueueName, armyMove)
			fmt.Println("Successfully published move:", armyMove)
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
