package pubsub

import (
	"context"
	"encoding/json"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
)

type SimpleQueueType int

const (
	TransientQueue SimpleQueueType = iota
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

func DeclareAndBind(conn *amqp.Connection, exchange, queueName, key string, simpleQueueType SimpleQueueType) (*amqp.Channel, amqp.Queue, error) {
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

func SubscribeJSON[T any](conn *amqp.Connection, exchange, queueName, key string, simpleQueueType SimpleQueueType, handler func(T)) error {
	amqpChan, _, err := DeclareAndBind(conn, exchange, queueName, key, simpleQueueType)
	if err != nil {
		return err
	}
	deliveryChan, err := amqpChan.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	go func(delChan <-chan amqp.Delivery) {
		for del := range delChan {
			var msg T
			err := json.Unmarshal(del.Body, &msg)
			if err != nil {
				log.Println(err)
			}
			handler(msg)
			del.Ack(false)
		}
	}(deliveryChan)

	return nil
}
