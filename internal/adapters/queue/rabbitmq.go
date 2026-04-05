package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
	amqp "github.com/rabbitmq/amqp091-go"
)


const (
	ExchangeName = "kliops_ingestion"
	QueueName = "ingestion_jobs"
	DLQExchange = "kliops_dlx"
	DLQQueue = "ingestion_dlq"
	MaxRetries = 3
)

type RabbitMQAdapter struct {
	Conn *amqp.Connection 
	Channel *amqp.Channel
}

type JobMessage struct {
	JobID string `json:"job_id"`
	ProjectID string `json:"project_id"`
	Retry int `json:"retry"`
}

func NewRabbitMQAdapter(amqpUri string) (*RabbitMQAdapter,error) {
	conn, dialingRabbitError := amqp.Dial(amqpUri)
	if dialingRabbitError != nil {
		return nil, fmt.Errorf("failed to dial the rabbitmq instance : %v",dialingRabbitError)
	}

	ch, channelCreationError := conn.Channel()
	if channelCreationError != nil {
		return nil, fmt.Errorf("failed to establish a new channel on top of the established connection : %v",channelCreationError)
	}

	// let's setup the DLQ 
	dlqExchangeDeclarationError := ch.ExchangeDeclare(DLQExchange,"direct",true, false,false,false,nil)
	if dlqExchangeDeclarationError != nil {
		return nil, fmt.Errorf("failed to create the exchange %s, an error occured : %v",DLQExchange,dlqExchangeDeclarationError)
	}
	_,dqlQueueDeclarationError := ch.QueueDeclare(DLQQueue,true,false,false,false,nil)
	if dqlQueueDeclarationError != nil {
		return nil, fmt.Errorf("failed to create the dlq queue %s , an error occured : %v",DLQQueue,dqlQueueDeclarationError)
	}
	bindingDlqQueueAndExchangeError := ch.QueueBind(DLQQueue,"",DLQExchange,false,nil)
	if bindingDlqQueueAndExchangeError != nil {
		return nil,fmt.Errorf("failed to bind the Queue %s to the exchange %s , an error occured : %v",DLQQueue,DLQExchange,bindingDlqQueueAndExchangeError)
	}

	//Setup main queue with DLQ arguments 
	args := amqp.Table{
		"x-dead-letter-exchange":DLQExchange,
	}
	mainExchangeDeclarationError := ch.ExchangeDeclare(ExchangeName,"direct",true,false,false,false,nil)
	if mainExchangeDeclarationError != nil {
		return nil, fmt.Errorf("failed to create the exchange %s, an error occured : %v",ExchangeName,mainExchangeDeclarationError)
	}
	_,mainQueueDeclarationError := ch.QueueDeclare(QueueName,true,false,false,false,args)
	if mainQueueDeclarationError != nil {
		return nil, fmt.Errorf("failed to create the dlq queue %s , an error occured : %v",QueueName,mainQueueDeclarationError)
	}
	bindingQueueAndExchangeError := ch.QueueBind(QueueName,"",ExchangeName,false,nil)
	if bindingQueueAndExchangeError != nil {
		return nil,fmt.Errorf("failed to bind the Queue %s to the exchange %s , an error occured : %v",QueueName,ExchangeName,bindingQueueAndExchangeError)
	}

	return &RabbitMQAdapter{Conn:conn,Channel:ch,},nil
}

func (r *RabbitMQAdapter) PublishJob(ctx context.Context,jobID,projectID string) error {
	msg:= JobMessage{JobID:jobID,ProjectID:projectID,Retry:0}
	body, marshallingbodyError := json.Marshal(msg)
	if marshallingbodyError != nil {
		return fmt.Errorf("an error happened while marshalling the Job: %s , to be published over rabbitmq: %v ",jobID,marshallingbodyError)
	}

	return r.Channel.PublishWithContext(ctx,ExchangeName,"",false,false,amqp.Publishing{
		ContentType: "application/json",
		Body:	body,
		DeliveryMode: 	amqp.Persistent,
	})
}


func (r *RabbitMQAdapter) ConsumeJob(ctx context.Context, handler func(ctx context.Context,jobID,projectID string) error) error {
	msgs, consumingMessageError := r.Channel.Consume(QueueName,"",false,false,false,false,nil)
	if consumingMessageError != nil {
		return fmt.Errorf("an error occured while consuming the message : %v",consumingMessageError)
	}
	go func(){
		for d := range msgs {
			var msg JobMessage 
			json.Unmarshal(d.Body,&msg)

			handlingMessageError := handler(ctx,msg.JobID,msg.ProjectID)
			if handlingMessageError != nil {
				msg.Retry++ 
				if msg.Retry > MaxRetries {
					log.Printf("Job %s failed permanently. Moving to DLQ",msg.JobID)
					d.Nack(false,false)
				}else{
					log.Printf("Job %s failed. Retry %d/%d",msg.JobID,msg.Retry,MaxRetries)
					//TODO: Use RabbitMQ delayed Exchange here 
					time.Sleep(time.Duration(msg.Retry*2) * time.Second)
					d.Nack(false,true) // we requeue the Job
				}
				continue
			}
			d.Ack(false)
		}
	}()
	return nil
}