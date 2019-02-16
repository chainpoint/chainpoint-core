package rabbitmq

import (
	"log"

	"github.com/streadway/amqp"
)

type Session struct {
	Conn   *amqp.Connection
	Ch     *amqp.Channel
	Queue  amqp.Queue
	Msgs   <-chan amqp.Delivery
	Notify chan *amqp.Error
}

func LogError(err error, msg string) {
	if err != nil {
		log.Printf("%s: %s", msg, err)
	}
}

/* Dial AMQP provider, create a channel, and declare a queue object*/
func Dial(amqpUrl string, queue string) (Session, error) {
	var session Session
	var err error
	session.Conn, err = amqp.Dial(amqpUrl)
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
		false, // delete when usused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		LogError(err, "Problem with queue declare")
		return session, err
	} else {
		return session, nil
	}
}

/* Begins consumption of queue by returning a delivery channel via the session struct*/
func ConnectAndConsume(rabbitmqConnectUri string, queue string) (session Session, err error) {
	//Connect to Queue
	session, err = Dial(rabbitmqConnectUri, queue)
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
