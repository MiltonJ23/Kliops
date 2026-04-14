package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ExchangeName = "kliops_ingestion"
	QueueName    = "ingestion_jobs"
	DLQExchange  = "kliops_dlx"
	DLQQueue     = "ingestion_dlq"
	MaxRetries   = 3
)

type RabbitMQAdapter struct {
	Conn      *amqp.Connection
	Channel   *amqp.Channel
	PublishMu sync.Mutex
}

type JobMessage struct {
	JobID     string `json:"job_id"`
	ProjectID string `json:"project_id"`
	Retry     int    `json:"retry"`
}

func NewRabbitMQAdapter(amqpUri string) (*RabbitMQAdapter, error) {
	conn, dialingRabbitError := amqp.Dial(amqpUri)
	if dialingRabbitError != nil {
		return nil, fmt.Errorf("failed to dial the rabbitmq instance : %v", dialingRabbitError)
	}

	ch, channelCreationError := conn.Channel()
	if channelCreationError != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to establish a new channel on top of the established connection : %v", channelCreationError)
	}

	// let's setup the DLQ
	dlqExchangeDeclarationError := ch.ExchangeDeclare(DLQExchange, "direct", true, false, false, false, nil)
	if dlqExchangeDeclarationError != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to create the exchange %s, an error occured : %v", DLQExchange, dlqExchangeDeclarationError)
	}
	_, dqlQueueDeclarationError := ch.QueueDeclare(DLQQueue, true, false, false, false, nil)
	if dqlQueueDeclarationError != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to create the dlq queue %s , an error occured : %v", DLQQueue, dqlQueueDeclarationError)
	}
	bindingDlqQueueAndExchangeError := ch.QueueBind(DLQQueue, "", DLQExchange, false, nil)
	if bindingDlqQueueAndExchangeError != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to bind the Queue %s to the exchange %s , an error occured : %v", DLQQueue, DLQExchange, bindingDlqQueueAndExchangeError)
	}

	//Setup main queue with DLQ arguments
	args := amqp.Table{
		"x-dead-letter-exchange": DLQExchange,
	}
	mainExchangeDeclarationError := ch.ExchangeDeclare(ExchangeName, "direct", true, false, false, false, nil)
	if mainExchangeDeclarationError != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to create the exchange %s, an error occured : %v", ExchangeName, mainExchangeDeclarationError)
	}
	_, mainQueueDeclarationError := ch.QueueDeclare(QueueName, true, false, false, false, args)
	if mainQueueDeclarationError != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to create the main queue %s , an error occured : %v", QueueName, mainQueueDeclarationError)
	}
	bindingQueueAndExchangeError := ch.QueueBind(QueueName, "", ExchangeName, false, nil)
	if bindingQueueAndExchangeError != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to bind the Queue %s to the exchange %s , an error occured : %v", QueueName, ExchangeName, bindingQueueAndExchangeError)
	}

	return &RabbitMQAdapter{Conn: conn, Channel: ch}, nil
}

func (r *RabbitMQAdapter) Close() error {
	var errs []error
	if r.Channel != nil {
		if err := r.Channel.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close channel: %w", err))
		}
	}
	if r.Conn != nil {
		if err := r.Conn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close connection: %w", err))
		}
	}
	return errors.Join(errs...)
}

func (r *RabbitMQAdapter) PublishJob(ctx context.Context, jobID, projectID string) error {
	msg := JobMessage{JobID: jobID, ProjectID: projectID, Retry: 0}
	body, marshallingbodyError := json.Marshal(msg)
	if marshallingbodyError != nil {
		return fmt.Errorf("an error happened while marshalling the Job: %s , to be published over rabbitmq: %v ", jobID, marshallingbodyError)
	}

	r.PublishMu.Lock()
	defer r.PublishMu.Unlock()

	return r.Channel.PublishWithContext(ctx, ExchangeName, "", false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent,
	})
}

func (r *RabbitMQAdapter) ConsumeJob(ctx context.Context, maxConcurrency int, handler func(ctx context.Context, jobID, projectID string) error) error {
	msgs, channelCreationError := r.Channel.Consume(QueueName, "", false, false, false, false, nil)
	if channelCreationError != nil {
		return fmt.Errorf("failed to consume from Queue : %v", channelCreationError)
	}

	// This will work more likely like a worker pool
	sem := make(chan struct{}, maxConcurrency)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case d, ok := <-msgs:
				if !ok {
					return
				}
				// let's launch the processing of a message in its own isolated goroutine
				go func(delivery amqp.Delivery) {
					acquired := false
					release := func() {
						if acquired {
							<-sem
							acquired = false
						}
					}
					select {
					case sem <- struct{}{}:
						acquired = true
						defer release()
					case <-ctx.Done():
						return
					}

					// Check context before processing
					if ctx.Err() != nil {
						delivery.Nack(false, true) // Requeue
						return
					}

					var msg JobMessage
					unmarshalingMessageError := json.Unmarshal(delivery.Body, &msg)
					if unmarshalingMessageError != nil {
						log.Printf("Unprocessable message format, dropping message : %v", unmarshalingMessageError)
						delivery.Nack(false, false) // we send it directly to DLQ so that i can look at it after
						return
					}
					handlingErr := handler(ctx, msg.JobID, msg.ProjectID)
					if handlingErr != nil {
						msg.Retry++
						if msg.Retry > MaxRetries {
							log.Printf("Job %s failed permanently after %d retries . Moving to DLQ.", msg.JobID, MaxRetries)
							delivery.Nack(false, false)
						} else {
							log.Printf("Job %s failed (Attempt %d/%d). Requeueing ", msg.JobID, msg.Retry, MaxRetries)

							// Perform retry synchronously instead of in a nested goroutine
							release()
							select {
							case <-time.After(time.Duration(msg.Retry*5) * time.Second):
								sem <- struct{}{}
								acquired = true

								// Check context before retrying
								if ctx.Err() != nil {
									delivery.Nack(false, true) // Requeue on context cancellation
									return
								}

								body, marshalErr := json.Marshal(msg)
								if marshalErr != nil {
									log.Printf("Error marshalling retry message: %v", marshalErr)
									delivery.Nack(false, false)
									return
								}

								r.PublishMu.Lock()
								publishErr := r.Channel.PublishWithContext(ctx, ExchangeName, "", false, false, amqp.Publishing{
									ContentType:  "application/json",
									Body:         body,
									DeliveryMode: amqp.Persistent,
								})
								r.PublishMu.Unlock()

								if publishErr != nil {
									log.Printf("Error publishing retry message: %v", publishErr)
									delivery.Nack(false, true) // Requeue
									return
								}

								delivery.Ack(false)
							case <-ctx.Done():
								delivery.Nack(false, true) // Requeue on context cancellation
								return
							}
						}
						return
					}
					delivery.Ack(false)
				}(d)
			}
		}
	}()
	return nil
}
