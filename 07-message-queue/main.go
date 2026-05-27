/*
message queue: a system that allows different parts of a system to communicate with each other asynchronously by sending messages to a queue.
This allows for decoupling of different parts of the system and can improve scalability and reliability.
breaking up a task into smaller parts and processing each part separately but coordinating them through a message queue so another part of the system can pick up.
*/
package main

import (
	"fmt"
	"sync"
	"time"
)

type Message struct {
	ID       string
	Payload  []byte    //raw data to be processed
	AckedAt  time.Time //track when message was acknowledged by consumer
	Deadline time.Time //track when message should be re-delivered if not acknowledged (timeout message and redeliver to another consumer)
}

// log of messages for debugging and monitoring
// under a specific topic/subject
type Topic struct {
	name     string
	mu       sync.Mutex
	messages []*Message //store messages in local memory
}

// broker manages multiple topics and handles message routing
type Broker struct {
	mu     sync.Mutex
	topics map[string]*Topic //map of topic name to topic struct
}

func generateID() string {
	return fmt.Sprintf("msg-%d", time.Now().UnixNano())
}

// constructor for broker
func NewBroker() *Broker {
	return &Broker{
		topics: make(map[string]*Topic), //initialize empty map of topics
	}
}

// publish a message to a topic
func (b *Broker) Publish(topicName string, payload []byte) {
	b.mu.Lock()
	topic, ok := b.topics[topicName]

	//if topic doesn't exist, create it
	if !ok {
		topic = &Topic{ //pointer to topic so we can modify it in the broker
			name:     topicName,
			messages: make([]*Message, 0), //initialize empty slice of messages
		}
		b.topics[topicName] = topic //add topic to broker's map of topics
	}
	b.mu.Unlock()

	//new message with unique ID and payload
	msg := &Message{
		ID:      generateID(),
		Payload: payload,
	}
	topic.mu.Lock()
	topic.messages = append(topic.messages, msg) //add message to topic's list of messages
	topic.mu.Unlock()
}

// consumer is just something that reads messages out of a queue.
// we need to track which messages have been consumed
type Consumer struct {
	id     string
	topic  *Topic
	offset int //track which messages have been consumed in the topic
}

// subscribe a consumer to a topic
func (b *Broker) Subscribe(topicName string, consumerID string) *Consumer {
	b.mu.Lock()
	topic, ok := b.topics[topicName]

	//if topic doesn't exist, create it empty topic with no messages
	if !ok {
		topic = &Topic{
			name:     topicName,
			messages: make([]*Message, 0),
		}
		b.topics[topicName] = topic
	}
	b.mu.Unlock()

	//create consumer subscribed to the topic (inital connection to the topic)
	return &Consumer{ //pointer
		id:     consumerID,
		topic:  topic,
		offset: 0,
	}
}

// read next message for this consumer from the topic.
func (c *Consumer) Poll() *Message {
	c.topic.mu.Lock()
	defer c.topic.mu.Unlock()

	//check if any new message to consume
	if c.offset >= len(c.topic.messages) {
		return nil //no new messages to consume
	}
	msg := c.topic.messages[c.offset]
	c.offset++ //move offset to next message for next poll
	return msg
}

// consumer group so it syncs message across multiple consumers
type ConsumerGroup struct {
	id        string
	topic     *Topic
	mu        sync.Mutex
	offset    int //tracks which message consumed across the grou p
	next      int //index of next consumer to receive message
	consumers []*Consumer
	inFlight  map[string]*Message //track messages that have been dispatched but not yet acknowledged
}

// subscribe a consumer group to a topic
func (b *Broker) SubscribeGroup(topicName string, groupID string) *ConsumerGroup {
	b.mu.Lock()
	topic, ok := b.topics[topicName]

	//if topic doesn't exist, create it empty topic with no messages
	if !ok {
		topic = &Topic{
			name:     topicName,
			messages: make([]*Message, 0),
		}
		b.topics[topicName] = topic
	}
	b.mu.Unlock()

	//create consumer group subscribed to the topic (inital connection to the topic)
	return &ConsumerGroup{
		id:        groupID,
		topic:     topic,
		offset:    0,
		consumers: make([]*Consumer, 0),      //initialize empty slice of consumers
		inFlight:  make(map[string]*Message), //initialize empty map of in-flight messages
	}
}

// read next message for this consumer from the topic.
func (g *ConsumerGroup) Poll() *Message {
	g.mu.Lock()
	defer g.mu.Unlock()

	//check if any new message to consume
	if g.offset >= len(g.topic.messages) {
		return nil //no new messages to consume
	}
	msg := g.topic.messages[g.offset]
	g.offset++ //move offset to next message for next poll
	return msg
}

func (g *ConsumerGroup) AddConsumer(id string) *Consumer {
	g.mu.Lock()
	defer g.mu.Unlock()

	c := &Consumer{
		id:     id,
		topic:  g.topic,
		offset: 0,
	}
	g.consumers = append(g.consumers, c) //add new consumer
	return c                             //return pointer to new consumer so we can poll it
}

// dispatch next message to consumer using round robin and track it as in-flight until acknowledged
func (g *ConsumerGroup) Dispatch() (*Consumer, *Message) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.offset >= len(g.topic.messages) || len(g.consumers) == 0 {
		return nil, nil //no new messages to consume or no consumers in the group
	}

	msg := g.topic.messages[g.offset]
	consumer := g.consumers[g.next%len(g.consumers)] //round robin dispatch to consumers in the group
	g.next++                                         //move to next consumer for next dispatch
	g.offset++                                       //move to next message for next dispatch

	msg.Deadline = time.Now().Add(30 * time.Second) //set deadline for message acknowledgment (e.g. 30 seconds)
	g.inFlight[msg.ID] = msg                        //track message as in-flight until acknowledged

	return consumer, msg
}

// acknowledge a message as process by consumer and remove from in flight
func (g *ConsumerGroup) Ack(msgID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.inFlight, msgID) //remove message from in-flight tracking when acknowledged
}

// add back to topic if message not acknolwedged in time for deadline so it can be redelivered to another consumer
func (g *ConsumerGroup) Requeue() {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now()
	for id, msg := range g.inFlight {
		if now.After(msg.Deadline) {
			delete(g.inFlight, id) //remove from in-flight tracking
			g.topic.mu.Lock()
			g.topic.messages = append(g.topic.messages, msg) //requeue message back to topic for redelivery
			g.topic.mu.Unlock()
		}
	}
}

func main() {
	broker := NewBroker()

	//Scenario A: pub/sub
	fmt.Println("Scenario A: independent consumers")
	broker.Publish("orders", []byte("order-1")) //publish message to topic "orders"
	broker.Publish("orders", []byte("order-2"))
	broker.Publish("orders", []byte("order-3"))

	c1 := broker.Subscribe("orders", "consumer-1") //subscribe consumer-1 to topic "orders"
	c2 := broker.Subscribe("orders", "consumer-2")

	for {
		msg := c1.Poll() //poll next message for consumer-1
		if msg == nil {
			break
		}
		fmt.Printf("consumer-1 got: %s\n", string(msg.Payload))
	}
	for {
		msg := c2.Poll()
		if msg == nil {
			break
		}
		fmt.Printf("consumer-2 got: %s\n", string(msg.Payload))
	}

	//Scenario B: consumer group with redelivery
	fmt.Println("\n Scenario B: consumer group + redelivery")
	broker.Publish("jobs", []byte("job-1")) //message to topic "jobs"
	broker.Publish("jobs", []byte("job-2"))
	broker.Publish("jobs", []byte("job-3"))
	broker.Publish("jobs", []byte("job-4"))

	group := broker.SubscribeGroup("jobs", "workers") //subscribe consumer group "workers" to topic "jobs"
	w1 := group.AddConsumer("worker-1")               //add consumer "worker-1" to the group
	w2 := group.AddConsumer("worker-2")
	_ = w1
	_ = w2

	var lastMsg *Message
	for i := 0; i < 4; i++ {
		consumer, msg := group.Dispatch() //dispatch next message to a consumer in the group
		if msg == nil {
			break
		}
		fmt.Printf("dispatched %s to %s\n", string(msg.Payload), consumer.id)
		if i < 3 {
			group.Ack(msg.ID) //acknowledge first 3 messages as processed
		} else {
			lastMsg = msg
			lastMsg.Deadline = time.Now().Add(-1 * time.Second) // force expired to simulate crash
		}
	}

	fmt.Println("calling Requeue...")
	group.Requeue() //requeue any messages that were not acknowledged in time (e.g. due to worker crash) (Last message since we ended it early)

	consumer, msg := group.Dispatch() //dispatch next message to a consumer in the group (should get the requeued message)
	if msg != nil {
		fmt.Printf("redelivered %s to %s\n", string(msg.Payload), consumer.id)
	}
}
