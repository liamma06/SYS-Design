# How Each Component Could Be Better

A review of all 9 components — what's missing, how to improve them, and what they map to in production.

---

## 00 — TCP Server

**What you built:** Echo server, one goroutine per connection, raw byte reads.

**What's missing / how to improve:**
- **Message framing** — TCP is a stream, not packets. The 1024-byte buffer will silently truncate messages. Fix: length-prefix each message (`[4-byte len][payload]`) or use a delimiter + `bufio.Scanner`.
- **Backpressure** — if a client is slow, the goroutine blocks forever. Production servers use a worker pool with bounded channels.
- **Graceful shutdown** — `os.Signal` + `listener.Close()` so in-flight connections finish cleanly.

**Real-world counterparts:** This is the foundation of every server. Redis, PostgreSQL, and Nginx all do exactly this — `accept()` in a loop, route to workers. NGINX uses an event loop (epoll) instead of goroutines to handle 10k+ connections without the memory overhead of one goroutine per connection.

---

## 01 — HTTP Server

**What you built:** Manual HTTP/1.1 parser over TCP. Handles GET with 2 routes.

**What's missing / how to improve:**
- **Request body / POST** — parse `Content-Length` header and read that many bytes from the connection after the blank line.
- **Connection keep-alive** — HTTP/1.1 defaults to persistent connections. Closing after every response is slower.
- **Error resilience** — a malformed request panics or silently drops. Validate method, check for CRLF, return `400 Bad Request` on invalid input.
- **MIME types** — `Content-Type: text/html` should be set on responses so browsers render correctly.

**Real-world counterparts:** Go's `net/http` package does exactly what you wrote, just battle-hardened. For raw performance, **Nginx** and **Caddy** are written this way. **HTTP/2** and **HTTP/3 (QUIC)** replace the TCP stream with multiplexed streams — worth understanding once HTTP/1.1 is solid.

---

## 02 — WebSocket

**What you built:** RFC 6455 handshake, frame parsing, XOR unmasking, text echo.

**What's missing / how to improve:**
- **Extended payload length** — frames with `>125` byte payloads use a 16-bit or 64-bit extended length field. Current code corrupts those frames silently.
- **FIN bit** — must set bit 7 of byte 0 in response frames. (`0x81` is correct for text, but the logic should be explicit.)
- **Fragmentation** — browsers can split large messages into continuation frames (opcode `0x00`). Need to buffer and reassemble them.
- **Ping/pong** — browsers send heartbeat pings; if the server doesn't respond with pong, the browser drops the connection.

**Real-world counterparts:** **Socket.io** (Node.js), **Gorilla WebSocket** (Go), and **uWebSockets** (C++) all implement this exact spec. For high-scale real-time systems (Slack, Discord), a dedicated WebSocket gateway fans messages out via an internal pub/sub bus — the same pattern as component 07.

---

## 03 — Rate Limiter

**What you built:** Token bucket per-IP with mutex-protected map.

**What's missing / how to improve:**
- **`sync.Map` instead of `map + Mutex`** — for many IPs hitting simultaneously, a sharded or `sync.Map` approach reduces lock contention.
- **Cleanup of stale buckets** — the map grows forever. A background goroutine should evict buckets not seen in >N minutes.
- **Multiple strategies** — token bucket is one of three common algorithms. **Sliding window** is more accurate (higher memory). **Leaky bucket** smooths bursts. The tradeoff is precision vs. memory vs. complexity.
- **Per-endpoint limits** — `/login` should have tighter limits than `/static`.

**Real-world counterparts:** **Nginx's `limit_req`** module uses a leaky bucket. **Cloudflare's rate limiter** uses a sliding window stored in a distributed key-value store. **Redis + Lua scripts** are the standard backend for distributed rate limiting across multiple app servers — an in-process map doesn't work when you have 3 instances.

---

## 04 — Consistent Hashing

**What you built:** Virtual nodes on a sorted ring, clockwise lookup.

**What's missing / how to improve:**
- **Thread safety** — `AddNode`/`RemoveNode` are not safe to call concurrently with `GetNode`. Add a `sync.RWMutex`.
- **Higher vnode count** — 10 vnodes per server gives uneven distribution in practice. 150–300 is the industry standard (used by Cassandra).
- **Replication** — in real systems, `GetNode` returns the *next N nodes clockwise* for replication. The current API only returns one.
- **`O(log n)` lookup** — use `sort.Search` (binary search) instead of a linear scan for large rings.

**Real-world counterparts:** **Amazon DynamoDB**, **Apache Cassandra**, and **Riak** all use consistent hashing with vnodes for shard routing. **memcached** client libraries (like `ketama`) implement exactly this algorithm. **Redis Cluster** uses a variant called *hash slots* (16,384 fixed slots) instead of a continuous ring.

---

## 05 — Load Balancer

**What you built:** Least-connections balancer with TCP health checks, reverse proxy.

**What's missing / how to improve:**
- **Health check response validation** — currently only checks TCP connectivity, not whether the backend returns `200`. A hung app could accept TCP but never respond.
- **Weighted backends** — some servers are more powerful. A `weight` field lets them receive proportionally more traffic.
- **Sticky sessions** — some apps require a client always hits the same backend (e.g., server-side sessions). Consistent hashing on client IP solves this — combine with component 04.
- **Circuit breaker** — if a backend fails 5 times in 30 seconds, stop routing to it immediately rather than waiting for the health check cycle.

**Real-world counterparts:** **HAProxy** is the gold standard for TCP/HTTP load balancing — this architecture mirrors its "least-connections" mode. **Nginx upstream** does the same at L7. **AWS ALB** and **GCP Load Balancer** operate at cloud scale. **Envoy Proxy** (used in Istio) adds observability, retries, and circuit breaking on top of what's here.

---

## 06 — Distributed Cache

**What you built:** Multi-node in-memory cache, consistent hash routing from client, TTL eviction.

**What's missing / how to improve:**
- **Replication** — if a node goes down, all its keys are gone. Real caches write each key to a primary + N replicas. Use the consistent hashing ring returning N nodes instead of one.
- **Binary protocol** — the text protocol (`SET key value`) is fine for learning but slow to parse at scale. Redis uses RESP (Redis Serialization Protocol) — a simple binary framing that's worth implementing.
- **LRU eviction policy** — when cache is full, TTL-only eviction means cold keys live forever if set without TTL. Add an LRU or LFU eviction layer.
- **Hot key problem** — if one key gets 90% of traffic, one node gets hammered. Solutions: local client-side cache for hot keys, or replicate hot keys to all nodes.

**Real-world counterparts:** This is essentially a minimal **Redis** (single-threaded key-value store over TCP with TTL). **Memcached** is even closer — purely in-memory, no persistence. **Redis Cluster** adds the consistent hashing + replication layer that's missing here. Facebook's Memcached paper (TAO) describes the same hot-key and replication challenges at hyperscale.

---

## 07 — Message Queue

**What you built:** In-memory broker with pub/sub and consumer groups, round-robin dispatch, redelivery on deadline.

**What's missing / how to improve:**
- **Persistence** — on restart, all messages are lost. Real queues write to disk (WAL or append-only log) before acknowledging the producer.
- **Continuous requeue sweep** — `Requeue()` is called once in the demo. It should run as a background goroutine on a timer to continuously redeliver expired in-flight messages.
- **Dead-letter queue (DLQ)** — if a message fails N times, move it to a separate topic instead of redelivering forever. Critical for poison-pill messages.
- **Backpressure** — if producers publish faster than consumers consume, the in-memory slice grows unbounded. Add a max-size with blocking or dropping semantics.
- **Offset persistence** — consumer group offsets are lost on restart. Kafka stores offsets in a special internal topic; simpler queues store them in a database.

**Real-world counterparts:** This is a hybrid of **RabbitMQ** (ack/nack, redelivery, routing) and early **Apache Kafka** (consumer groups, offsets). RabbitMQ uses AMQP and pushes messages to consumers. Kafka is pull-based, stores messages durably, and lets consumers replay from any offset — making it far more powerful for event sourcing. **AWS SQS** is the managed version closest to this implementation (at-least-once delivery, visibility timeout = the deadline concept here).

---

## 08 — Log Aggregator

**What you built:** TCP log ingestion, ring buffer storage, HTTP query API with level/service/time filters.

**What's missing / how to improve:**
- **Ring buffer size is 10** — even in the demo, half the logs are already overwritten before the query runs. Should be 10,000+ or configurable.
- **Structured ingestion from multiple sources** — real aggregators handle varied formats (syslog, JSON, nginx access logs). Add format parsers or strict JSON validation.
- **Indexing** — every query scans all N entries. For large buffers, index by level and service into separate sorted lists, or use time-based buckets.
- **Persistent storage** — logs must survive restarts. Write entries to append-only files (one per day), keep the ring buffer as a hot cache.
- **Alerting** — when `ERROR` log count exceeds a threshold in a time window, fire a webhook. This is the primary value of a log aggregator.

**Real-world counterparts:** This is a minimal **Logstash** (from the ELK stack). The full pipeline is: **Filebeat** (collects from servers) → **Logstash** (parses/enriches) → **Elasticsearch** (indexes for fast queries) → **Kibana** (UI). **Grafana Loki** is a lighter alternative that indexes only labels (level, service) and stores raw log text — similar to this approach. **Splunk** does the full pipeline commercially.

---

## 09 — Task Scheduler

**What you built:** Fixed-interval job runner, one goroutine per job execution, ticker-based polling.

**What's missing / how to improve:**
- **Cron expression support** — fixed intervals can't express "run at 2am every Sunday". Add a cron parser (`*/5 * * * *` format). Implement a minimal one from scratch as an exercise.
- **Job result tracking** — tasks run fire-and-forget. Track `LastRun`, `LastDuration`, `LastError` on the `Job` struct.
- **Concurrency limit** — if a job takes longer than its interval, the next run starts while the previous is still running. Add a `running bool` flag to skip or queue the overlap.
- **Persistence** — jobs are defined in code. A real scheduler persists job definitions to a database and exposes an API to add/remove them at runtime.
- **Missed run detection** — if the process was down for 2 hours and a job runs every 30 minutes, should those 4 missed runs execute on startup? Kubernetes CronJob has a `concurrencyPolicy` for exactly this.

**Real-world counterparts:** This is a simplified **cron daemon** (Unix `cron` / `crond`). For distributed systems, **Celery Beat** (Python) and **Quartz Scheduler** (Java) handle persistent, distributed job scheduling. **Kubernetes CronJob** schedules containers on a cron schedule across a cluster. **Temporal.io** and **Inngest** are modern takes — they add durable execution, retry policies, and workflow orchestration on top of the same core idea.

---

## The Common Pattern

Looking across all 9 components, the happy path is solid but each is missing the same three production concerns:

| Gap | Affects |
|-----|---------|
| **Persistence** — state survives restarts | Cache, Queue, Logs, Scheduler |
| **Failure handling** — what happens mid-write when a connection drops | All of them |
| **Observability** — metrics, structured logging, tracing | None emit anything queryable |

These aren't things to fix now — they're exactly the problems Part 2 (the URL shortener cluster) is designed to force you to solve with real tools: PostgreSQL for persistence, Redis for caching, Nginx for load balancing, Prometheus + Grafana for observability.
