# 07 - Message Queue

A message queue decouples services by letting producers publish messages without knowing who consumes them or when. Consumers read at their own pace, which absorbs traffic spikes and lets services fail independently.

## What was built

**Broker** — owns all topics, the single entry point for publishing and subscribing.

**Topic** — a named, ordered log of messages. Like a TV channel: producers broadcast on it, consumers tune in.

**Producer** — calls `Publish(topic, payload)` to append a message to a topic.

**Consumer** — has its own offset (bookmark) into the topic. Calls `Poll()` to get the next unread message. Two consumers on the same topic each get every message independently (pub/sub).

**ConsumerGroup** — a group of consumers that share one offset and split the work. Messages are dispatched round-robin across consumers in the group. Used when you want parallel workers processing the same stream without duplicating work.

**Ack/Requeue** — when a message is dispatched it becomes "in-flight" with a deadline. The consumer must call `Ack(msgID)` before the deadline. If it doesn't (crash, timeout), `Requeue()` puts the message back into the topic for redelivery. This gives **at-least-once delivery** — a message may be delivered more than once if a consumer fails after receiving but before acking.

## Key concepts

- **Temporal decoupling** — producer and consumer don't need to be alive at the same time
- **Load leveling** — queue absorbs bursts; consumers process at their own rate
- **Offset** — each consumer/group tracks where it is in the log independently
- **Round-robin dispatch** — same modulo trick as consistent hashing, spreads work evenly
- **At-least-once vs exactly-once** — at-least-once is achievable here; exactly-once requires coordination on both sides and is much harder

## Demo

```
go run main.go
```

Scenario A shows two independent consumers both reading all messages from the same topic.
Scenario B shows a consumer group splitting 4 jobs between 2 workers, then simulates a crash on the last job and demonstrates redelivery via `Requeue()`.
