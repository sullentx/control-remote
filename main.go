package main

import (
	"log"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rabbitmq/amqp091-go"
)

type RabbitMQAdapter struct {
	conn *amqp091.Connection
	ch   *amqp091.Channel
}

func InitRabbitMQ() *RabbitMQAdapter {
	conn, err := amqp091.Dial("amqp://uriel:eduardo117@3.228.81.226:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}

	_, err = ch.QueueDeclare(
		"remote-control", // name
		true,             // durable
		false,            // delete when unused
		false,            // exclusive
		false,            // no-wait
		nil,              // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	return &RabbitMQAdapter{conn: conn, ch: ch}
}

func (r *RabbitMQAdapter) PublishMessage(message string) error {
	err := r.ch.Publish(
		"",               // exchange
		"remote-control", // routing key
		false,            // mandatory
		false,            // immediate
		amqp091.Publishing{
			ContentType: "text/plain",
			Body:        []byte(message),
		})
	if err != nil {
		log.Printf("Failed to publish message: %v", err)
		return err
	}

	log.Printf(" [x] Sent message: %s", message)
	return nil
}

func (r *RabbitMQAdapter) Close() {
	r.ch.Close()
	r.conn.Close()
}

// Estructura para recibir comandos desde el frontend
type CommandRequest struct {
	Command string `json:"command"`
}

func main() {
	rabbitMQ := InitRabbitMQ()
	defer rabbitMQ.Close()

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	router.POST("/control/command", func(c *gin.Context) {
		var request CommandRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
			return
		}

		if err := rabbitMQ.PublishMessage(request.Command); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send command"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "Command sent", "command": request.Command})
	})

	log.Println("Server running on port 8080...")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
