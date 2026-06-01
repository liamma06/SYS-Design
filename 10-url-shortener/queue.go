/*
queue to store messages for analyzing traffic in asynchronous way.
*/
package main

type Message struct {
	ID        string
	Payload   []byte //raw data to be processed
	Timestamp int64  //track when message was published
}

type Queue struct {
	messages chan Message
}

func NewQueue() *Queue {
	return &Queue{
		messages: make(chan Message, 100), //buffered channel with capacity of 100 messages
	}
}

func (q *Queue) Publish(msg Message) {
	q.messages <- msg //send message to channel, will block if channel is full
}

func (q *Queue) Subscribe(handler func(Message)) {
	go func() {
		for msg := range q.messages { //continuously read messages from channel
			handler(msg) //process message with provided handler function passed in
		}
	}()
}
