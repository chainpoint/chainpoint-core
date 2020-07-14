package rabbitmq

import (
	"errors"
	"log"

	"github.com/streadway/amqp"
)

// Session : Session contains all data necessary to describe an RMQ connection, including the queue
type Session struct {
	Conn   *amqp.Connection
	Ch     *amqp.Channel
	Queue  amqp.Queue
	Msgs   <-chan amqp.Delivery
	Notify chan *amqp.Error
}

// LogError : Describes an RMQ error
func LogError(err error, msg string) error {
	if err != nil {
		log.Printf("RabbitMQ Error: %s; %s", msg, err)
		return err
	}
	return nil
}

// End : Closes and ends an RMQ session
func (session *Session) End() error {
	if session.Ch != nil && session.Conn != nil {
		if errCh := session.Ch.Close(); errCh != nil {
			return errCh
		}
		if errConn := session.Conn.Close(); errConn != nil {
			return errConn
		}
		return nil
	}
	return errors.New("RabbitMQ: Either channel or connection is nil")
}

// Dial AMQP provider, create a channel, and declare a queue object
func Dial(amqpURL string, queue string) (Session, error) {
	var session Session
	var err error
	session.Conn, err = amqp.Dial(amqpURL)
	if err != nil {
		LogError(err, "dialing connection error")
		return session, err
	}
	session.Notify = session.Conn.NotifyClose(make(chan *amqp.Error))
	session.Ch, err = session.Conn.Channel()
	if err != nil {
		LogError(err, "Channel error")
		return session, err
	}
	session.Queue, err = session.Ch.QueueDeclare(
		queue, // name
		true,  // durable
		false, // delete when used
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		LogError(err, "Problem with queue declare")
		return session, err
	}

	return session, nil
}

// ConnectAndConsume begins consumption of queue by returning a delivery channel via the session struct
func ConnectAndConsume(rabbitmqConnectURI string, queue string) (session Session, err error) {
	//Connect to Queue
	session, err = Dial(rabbitmqConnectURI, queue)
	if err != nil {
		return Session{}, err
	}
	session.Msgs, err = session.Ch.Consume(
		session.Queue.Name, // queue
		"",                 // consumer
		false,              // auto-ack
		false,              // exclusive
		false,              // no-local
		false,              // no-wait
		nil,                // args
	)
	LogError(err, "can't consume queue")
	return
}

// Publish sends messages to a particular queue
func Publish(rabbitmqConnectURI string, queue string, msgType string, msg []byte) error {
	pubSession, err := Dial(rabbitmqConnectURI, queue)
	if err != nil {
		LogError(err, "failed to dial for publishing")
		return err
	}
	defer pubSession.Conn.Close()
	defer pubSession.Ch.Close()
	var publish amqp.Publishing
	if msgType != "" {
		publish = amqp.Publishing{
			Type:         msgType,
			Body:         msg,
			DeliveryMode: 2, //persistent
			ContentType:  "application/json",
		}
	} else {
		publish = amqp.Publishing{
			Body:         msg,
			DeliveryMode: 2, //persistent
			ContentType:  "application/json",
		}
	}
	err = pubSession.Ch.Publish(
		"",
		pubSession.Queue.Name,
		false,
		false,
		publish)
	if err != nil {
		LogError(err, "rmq dial failure, is rmq connected?")
		return err
	}
	return nil
}
