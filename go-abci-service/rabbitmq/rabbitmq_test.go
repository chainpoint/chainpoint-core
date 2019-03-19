package rabbitmq

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/chainpoint/chainpoint-core/go-abci-service/util"
)

func TestDial(t *testing.T) {
	assert := assert.New(t)
	time.Sleep(5 * time.Second) //sleep until rabbit comes online
	rabbitTestURI := util.GetEnv("RABBITMQ_URI", "amqp://chainpoint:chainpoint@rabbitmq:5672/")
	session, err := Dial(rabbitTestURI, "test")
	assert.NotEqual(session.Conn, nil, "RabbitMQ Connection should be non-nil and established")
	assert.Equal(err, nil, "RabbitMQ Connection err should be nil")
	err = session.End()
	assert.Equal(err, nil, "RabbitMQ Connection End() should return nil")
	session, err = Dial("", "test")
	assert.NotEqual(nil, err, "error for empty rabbitmquri should be non-nil")
}

func TestPublishConsume(t *testing.T) {
	assert := assert.New(t)
	time.Sleep(5 * time.Second) //sleep until rabbit comes online
	rabbitTestURI := util.GetEnv("RABBITMQ_URI", "amqp://chainpoint:chainpoint@rabbitmq:5672/")
	err := Publish(rabbitTestURI, "test", "test", []byte("msg"))
	assert.Equal(err, nil, "Err from successful publish should be nil")
	session, err := ConnectAndConsume(rabbitTestURI, "test")
	for m := range session.Msgs {
		assert.Equal(m.Body, []byte("msg"), "rabbitmq message body should be 'msg'")
		assert.Equal(m.Type, "test", "rabbitmq message type should be 'test'")
		m.Ack(false)
		break
	}
	err = session.End()
	assert.Equal(nil, err, "Session should have ended successfully")
	//test empty msgType
	err = Publish(rabbitTestURI, "test", "", []byte("msg"))
	assert.Equal(err, nil, "Err from successful publish should be nil")
	session, err = ConnectAndConsume(rabbitTestURI, "test")
	for m := range session.Msgs {
		assert.Equal(m.Body, []byte("msg"), "rabbitmq message body should be 'msg'")
		assert.Equal(m.Type, "", "rabbitmq message type should be empty")
		m.Ack(false)
		break
	}
	err = session.End()
	assert.Equal(nil, err, "Session should have ended successfully")
}
