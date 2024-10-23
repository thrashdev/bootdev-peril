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

type AckType int

const (
	Ack AckType = iota
	NackRequeue
	NackDiscard
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
	table := amqp.Table{}
	table["x-dead-letter-exchange"] = "peril_dlx"
	queue, err := amqpChan.QueueDeclare(queueName, durable, autoDelete, exclusive, noWait, table)
	if err != nil {
		log.Println("Crashed on Creating a Queue")
		return &amqp.Channel{}, amqp.Queue{}, err
	}
	amqpChan.QueueBind(queueName, key, exchange, noWait, nil)
	return amqpChan, queue, nil
}

func SubscribeJSON[T any](conn *amqp.Connection, exchange, queueName, key string, simpleQueueType SimpleQueueType, handler func(T) AckType) error {
	amqpChan, _, err := DeclareAndBind(conn, exchange, queueName, key, simpleQueueType)
	if err != nil {
		return err
	}
	deliveryChan, err := amqpChan.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	go func(delChan <-chan amqp.Delivery) {
		for msg := range delChan {
			var payload T
			err := json.Unmarshal(msg.Body, &payload)
			if err != nil {
				log.Println(err)
			}
			ackType := handler(payload)
			switch ackType {
			case Ack:
				msg.Ack(false)
				log.Println("Acked msg: ", msg)
			case NackRequeue:
				msg.Nack(false, true)
				log.Println("NackRequeued msg: ", msg)
			case NackDiscard:
				msg.Nack(false, false)
				log.Println("NackDiscarded msg: ", msg)
			}
			msg.Ack(false)
		}
	}(deliveryChan)

	return nil
}
