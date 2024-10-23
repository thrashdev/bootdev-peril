package main

import (
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/thrashdev/bootdev-peril/internal/gamelogic"
	"github.com/thrashdev/bootdev-peril/internal/pubsub"
	"github.com/thrashdev/bootdev-peril/internal/routing"
)

func HandlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	return func(ps routing.PlayingState) pubsub.AckType {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
		return pubsub.Ack
	}
}

func HandlerMove(amqpCh *amqp.Channel, gs *gamelogic.GameState) func(move gamelogic.ArmyMove) pubsub.AckType {
	return func(move gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Print("> ")
		outcome := gs.HandleMove(move)
		switch outcome {
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack
		case gamelogic.MoveOutcomeMakeWar:
			routingKey := routing.WarRecognitionsPrefix + "." + gs.GetUsername()
			recWar := gamelogic.RecognitionOfWar{Attacker: move.Player, Defender: gs.GetPlayerSnap()}
			err := pubsub.PublishJSON(amqpCh, amqp.ExchangeTopic, routingKey, recWar)
			if err != nil {
				log.Println("Error during publishing war move: ", err)
				return pubsub.NackRequeue
			}
			return pubsub.NackRequeue
		case gamelogic.MoveOutcomeSamePlayer:
			return pubsub.Ack
		default:
			return pubsub.NackDiscard
		}
	}
}

func HandlerWar(gs *gamelogic.GameState) func(recWar gamelogic.RecognitionOfWar) pubsub.AckType {
	return func(recWar gamelogic.RecognitionOfWar) pubsub.AckType {
		defer fmt.Print("> ")
		outcome, _, _ := gs.HandleWar(recWar)
		log.Println("War outcome: ", outcome)
		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard
		case gamelogic.WarOutcomeOpponentWon:
			return pubsub.Ack
		case gamelogic.WarOutcomeYouWon:
			return pubsub.Ack
		case gamelogic.WarOutcomeDraw:
			return pubsub.Ack
		default:
			log.Println("Error: Unexpected war outcome")
			return pubsub.NackDiscard
		}
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

	publishCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("Could not create channel: %v", err)
	}
	fmt.Println("Successfully connected to RabbitMQ")
	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Println("Crashed on Welcome")
		log.Fatal(err)
	}
	queueName := fmt.Sprintf("%s.%s", routing.PauseKey, username)
	gameState := gamelogic.NewGameState(username)
	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilDirect, queueName, routing.PauseKey, pubsub.TransientQueue, HandlerPause(gameState))
	if err != nil {
		log.Println("Crashed at subscribing to:", queueName)
		log.Fatal(err)
	}

	armyMovesWildcard := routing.ArmyMovesPrefix + ".*"
	armyQueueName := routing.ArmyMovesPrefix + "." + username
	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, armyQueueName, armyMovesWildcard, pubsub.TransientQueue, HandlerMove(publishCh, gameState))
	if err != nil {
		log.Println("Crashed at subscribing to:", armyQueueName)
		log.Fatal(err)
	}

	warQueueName := "war"
	warRoutingKey := routing.WarRecognitionsPrefix + ".*"
	// _, _, err = pubsub.DeclareAndBind(conn, routing.ExchangePerilTopic, warQueueName, warRoutingKey, pubsub.DurableQueue)
	// if err != nil {
	// 	log.Println("Crashed at declaring:", warQueueName)
	// 	log.Fatal(err)
	// }
	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, warQueueName, warRoutingKey, pubsub.DurableQueue, HandlerWar(gameState))
	if err != nil {
		log.Println("Crashed at subscribing to:", warQueueName)
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
			err = pubsub.PublishJSON(publishCh, routing.ExchangePerilTopic, armyQueueName, armyMove)
			if err != nil {
				log.Println("Error encountered during publishing of army move")
				log.Println(err)
			}
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
