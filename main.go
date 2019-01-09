package main

import (
	"fmt"
	"log"
	"math"

	"github.com/json-iterator/go"
	"github.com/streadway/amqp"

	"github.com/AdhityaRamadhanus/fasthttpcors"
	"github.com/qiangxue/fasthttp-routing"
	"github.com/valyala/fasthttp"
)

type RabbitMQ struct {
	ConnStr, Port, QueueName string
	Channel                  *amqp.Channel
	Connection               *amqp.Connection
	Queue                    *amqp.Queue
	errorChannel             chan *amqp.Error
	closed                   bool
}

type MQMessage struct {
	Message string
}

type Attachment struct {
	MimeType, FileName, Extension, Body string
}

type Message struct {
	Subject, Content string
}

type Receivers struct {
	To  string
	Cc  []string
	Bcc []string
}

type MailModel struct {
	RefId, FromName, FromAddress, Description, SystemName string
	Attachments                                           []Attachment
	Message                                               Message
	Receivers                                             Receivers
}

var instantiated *RabbitMQ = nil

func GetRabbitMQInstance(connStr string, QueueName string) *RabbitMQ {
	if instantiated == nil {
		instantiated = NewRabbitMQ(connStr, QueueName)
	}

	return instantiated
}

func NewRabbitMQ(connStr string, QueueName string) *RabbitMQ {

	var r = &RabbitMQ{ConnStr: connStr, QueueName: QueueName}
	r.connect()

	go r.reconnector()

	return r
}

func (r *RabbitMQ) PublishMessage(message *MailModel) {

	m, e := jsoniter.Marshal(&message)

	if e != nil {
		fmt.Println(e)
		return
	}

	err := r.Channel.Publish(
		"",          // exchange
		r.QueueName, // routing key
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        m,
		})

	if err != nil {
		log.Println(err)
	}
}

func (r *RabbitMQ) ConnectionClose() {
	r.Connection.Close()
}

func (r *RabbitMQ) ChannelClone() {
	r.Channel.Close()
}

func (r *RabbitMQ) close() {
	r.closed = true
	r.Channel.Close()
	r.Connection.Close()
}

func (r *RabbitMQ) connect() {

	conn, _ := amqp.Dial(r.ConnStr)
	r.Connection = conn
	fmt.Println("amqp connected.!")

	r.errorChannel = make(chan *amqp.Error)
	r.Connection.NotifyClose(r.errorChannel)

	createChannel(r)

	fmt.Println("channel created.!")

	QueueDeclare(r)
}

func (q *RabbitMQ) reconnector() {
	for {
		err := <-q.errorChannel
		if err != nil {
			fmt.Println("Reconnecting after connection closed", err)

			q.connect()
		} else {
			return
		}
	}
}

func QueueDeclare(r *RabbitMQ) {

	_, err := r.Channel.QueueDeclare(
		r.QueueName, // name of the queue
		false,       // should the message be persistent? also queue will survive if the cluster gets reset
		false,       // autodelete if there's no consumers (like queues that have anonymous names, often used with fanout exchange)
		false,       // exclusive means I should get an error if any other consumer subsribes to this queue
		false,       // no-wait means I don't want RabbitMQ to wait if there's a queue successfully setup
		nil,         // arguments for more advanced configuration
	)
	if err != nil {
		fmt.Println(err)
	}
}

func createChannel(r *RabbitMQ) {
	ch, _ := r.Connection.Channel()
	r.Channel = ch
}

func SendRabbit(ctx *routing.Context) error {

	rmq := GetRabbitMQInstance("amqp://guest:guest@localhost:5672/", "mail-test-queue")

	m := new(MailModel)

	err := jsoniter.Unmarshal(ctx.Request.Body(), &m)
	if err != nil {
		fmt.Println(err)
		return err
	}

	rmq.PublishMessage(m)

	return nil
}

func main() {
	fmt.Println("hello kitty!")

	withCors := fasthttpcors.NewCorsHandler(fasthttpcors.Options{
		AllowMaxAge: math.MaxInt32,
	})

	router := routing.New()

	router.Post("/api/send", SendRabbit)

	panic(fasthttp.ListenAndServe(":3000", withCors.CorsMiddleware(router.HandleRequest)))
}
