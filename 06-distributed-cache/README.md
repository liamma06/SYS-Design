# 06 - Distributed Cache

A distributed in-memory key-value cache built from scratch in Go, similar to the core of Redis.

## How it works

Three cache server nodes run independently. A client uses consistent hashing to route every key to the correct node. The same key always hashes to the same node, so `Set` and `Get` always find each other.

```
go run . server 6379   ← node 1: Cache + TCP listener
go run . server 6380   ← node 2: Cache + TCP listener
go run . server 6381   ← node 3: Cache + TCP listener

go run . client        ← routes Set/Get across all three nodes
```

When the client calls `Set("user:1", "alice", 0)`:

```
1. hash("user:1") → picks node-2
2. opens TCP to localhost:6380
3. sends "SET user:1 alice\n"
4. node-2 stores it in its local cache
5. reads back "OK"
```

When the client calls `Get("user:1")`:

```
1. hash("user:1") → picks node-2 (same hash = same node every time)
2. opens TCP to localhost:6380
3. sends "GET user:1\n"
4. reads back "alice"
```

## Files

| File | Job |
|---|---|
| `main.go` | Cache data structure + TCP server |
| `ring.go` | Consistent hashing — maps a key to a node name |
| `client.go` | Routes Set/Get to the right node over TCP |

## TCP Protocol

Commands are plain text, one per line:

```
SET key value [ttl]   → OK
GET key               → value  OR  NULL
DELETE key            → OK
```

TTL is optional and uses Go duration syntax: `5s`, `1m`, `2h`.

## Key concepts

**Why consistent hashing?**
With simple `hash(key) % 3`, adding a 4th node remaps almost every key. Consistent hashing only moves ~25% of keys when a node is added or removed.

**Why RWMutex?**
Reads (Get) happen far more often than writes (Set/Delete). `RWMutex` allows unlimited concurrent reads but only one writer at a time, which is much faster than a plain `Mutex` under read-heavy load.

**Active + lazy eviction**
- Lazy: expired keys are caught on `Get` and silently returned as not found
- Active: a background goroutine sweeps the map every second and deletes expired keys that nobody reads

**One connection per command**
The client opens a new TCP connection for every command. Simple and correct, but a production client would use a connection pool to avoid the overhead of re-connecting on every request.

## Running

```bash
# terminal 1-3
cd 06-distributed-cache
go run . server 6379
go run . server 6380
go run . server 6381

# terminal 4
go run . client
```
