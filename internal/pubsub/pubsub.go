package pubsub

import (
	"context"
	"encoding/json"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
)

const (
	TransientQueue = iota
	DurableQueue
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	marshalled, err := json.Marshal(val)
	if err != nil {
		return err
	}
	ctx := context.Background()
	err = ch.PublishWithContext(ctx, exchange, key, false, false, amqp.Publishing{ContentType: "application/json", Body: marshalled})

	return nil
}

func DeclareAndBind(conn *amqp.Connection, exchange, queueName, key string, simpleQueueType int) (*amqp.Channel, amqp.Queue, error) {
	amqpChan, err := conn.Channel()
	durable := false
	autoDelete := false
	exclusive := false
	noWait := false
	switch simpleQueueType {
	case DurableQueue:
		durable = true
		autoDelete = false
		exclusive = false
	case TransientQueue:
		durable = false
		autoDelete = true
		exclusive = true
	}
	if err != nil {
		log.Println("Crashed on creating AMQP Channel")
		return &amqp.Channel{}, amqp.Queue{}, err
	}
	queue, err := amqpChan.QueueDeclare(queueName, durable, autoDelete, exclusive, noWait, nil)
	if err != nil {
		log.Println("Crashed on Creating a Queue")
		return &amqp.Channel{}, amqp.Queue{}, err
	}
	amqpChan.QueueBind(queueName, key, exchange, noWait, nil)
	return amqpChan, queue, nil
}
