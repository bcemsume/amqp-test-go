package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	_ "github.com/gorilla/mux"
	"github.com/streadway/amqp"
	_ "github.com/valyala/fasthttp"
	"io/ioutil"
	"log"
	"net/http"
	_"github.com/json-iterator/go"
)

type RabbitMQ struct {
	ConnStr, Port, QueueName string
	Channel                  *amqp.Channel
	Connection *amqp.Connection
	Queue *amqp.Queue
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
	To string
	Cc []string
	Bcc []string
}

type MailModel struct {
	RefId, FromName, FromAddress,Description, SystemName string
	Attachments []Attachment
	Message Message
	Receivers Receivers
}
var instantiated *RabbitMQ = nil

func GetRabbitMQInstance(connStr string, QueueName string) *RabbitMQ{
	if instantiated == nil{
		instantiated = NewRabbitMQ(connStr,QueueName)
	}
	return  instantiated
}

func NewRabbitMQ(connStr string, QueueName string) *RabbitMQ {

	var r = &RabbitMQ{ConnStr: connStr, QueueName: QueueName}

	conn, _ := amqp.Dial(r.ConnStr)
	r.Connection = conn
	fmt.Println("amqp connected.!")

	createChannel(r)

	fmt.Println("channel created.!")

	queueDeclare(r)

	return r
}

func (r *RabbitMQ) PublishMessage(message *MailModel) {

	m, e:= json.Marshal(&message)

	if e != nil{
		fmt.Println(e)
		return
	}

	err := r.Channel.Publish(
		"",     // exchange
		r.QueueName, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        m,
		})

	if err != nil{
		log.Println(err)
	}
	log.Printf("Message: %s", message.Message)

}

func (r* RabbitMQ) ConnectionClose() {
	r.Connection.Close()
}

func (r *RabbitMQ) ChannelClone(){
	r.Channel.Close()
}

func queueDeclare(r *RabbitMQ){

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

func createChannel(r *RabbitMQ){
	ch,_ :=r.Connection.Channel()
	r.Channel = ch
}

func SendRabbit(w http.ResponseWriter, r *http.Request) () {

	rmq := GetRabbitMQInstance("amqp://guest:guest@localhost:5672/","mail-test-queue")

	m:= new(MailModel)
	b, _ := ioutil.ReadAll(r.Body)
	err := json.Unmarshal(b,&m)
	if err != nil{
		fmt.Println(err)
		return
	}

	rmq.PublishMessage(m)
}

func main() {
	fmt.Println("hello kitty!")


	mux := mux.NewRouter()
	mux.HandleFunc("/api/send", SendRabbit)

	http.ListenAndServe(":3000",mux)
}