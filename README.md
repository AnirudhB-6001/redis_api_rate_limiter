# Redis API Rate Limiter

A small Go HTTP service that limits each client to 5 requests per 60-second window, using Redis to store temporary request counters.

## How it works

Each request includes an `X-Client-ID` header.

The Go service asks Redis to increment a temporary counter for that client. Redis also gives the counter a 60-second expiry.

- Requests 1–5 are allowed.
- Request 6 and later return HTTP `429 Too Many Requests`.
- Different client IDs have independent counters.
- When the Redis key expires, the next request starts a fresh window.

## Architecture

```text
client
  ↓
Go HTTP service
  ↓
Redis
```

- Go receives the HTTP request, reads X-Client-ID, and decides whether to allow or reject the request.
- Redis stores the temporary per-client counters and their expiry.
- Docker packages the Go service and lets the Go and Redis containers run and communicate separately.

## Endpoint

### `GET /hello`

Each request must include an `X-Client-ID` header.

Example:

```bash
curl -H "X-Client-ID: alice" http://127.0.0.1:8080/hello
```

Example response:

```text
hello alice, request 1
```

A request without `X-Client-ID` returns HTTP `400 Bad Request`.

Once the client exceeds 5 requests within the current window, the service returns HTTP `429 Too Many Requests`.

## Running the tests

The tests call the Go HTTP handler directly, but use a real Redis instance for the request counters.

Start Redis in Docker:

```bash
docker run -d \
  --name rate-limiter-redis \
  -p 127.0.0.1:6379:6379 \
  redis:8-alpine
```

Then run:

```bash
go test
```

The tests cover:
- the first `5` requests being allowed;
- the `6th` request returning HTTP `429`;
- rejection of requests without `X-Client-ID`;
- independent limits for different client IDs.


When the tests are finished, stop and remove the test Redis container:

```bash
docker stop rate-limiter-redis
docker rm rate-limiter-redis
```

## Building the API image

Build the Go service into a Docker image:

```bash
docker build -t rate-limiter-api .
```

## Running the complete system

First, create a private Docker network for the API and Redis containers:

```bash
docker network create rate-limiter-net
```

This network allows the two containers to communicate with each other without exposing Redis directly to the host.

Start Redis on the private network:

```bash
docker run -d \
  --name rate-limiter-redis \
  --network rate-limiter-net \
  redis:8-alpine
```

Redis is now reachable by the other container at:

```text
rate-limiter-redis:6379
```

Its port is not published to the host.


Start the API container:
```bash
docker run -d \
  --name rate-limiter-api \
  --network rate-limiter-net \
  -e REDIS_ADDR=rate-limiter-redis:6379 \
  -p 127.0.0.1:8080:8080 \
  rate-limiter-api
```

`REDIS_ADDR` tells the Go service where Redis is on the private Docker network.

The port mapping makes the API available only on the host's localhost:

`127.0.0.1:8080` -> API container port `8080`.

## Demonstrating the rate limit

Send 6 requests using the same client ID:

```bash
for i in {1..6}; do
  curl -s -w "HTTP %{http_code}\n" \
    -H "X-Client-ID: alice" \
    http://127.0.0.1:8080/hello
done
```

The first 5 requests should return `HTTP 200`.
The 6th request should return `HTTP 429` Too Many Requests.


## Reset after the window expires

Redis removes the client's counter after 60 seconds.
Wait for the current window to expire:
```bash
sleep 61
```

Then send another request with the same client ID:
```bash
curl -H "X-Client-ID: alice" http://127.0.0.1:8080/hello
```

The counter should start again at:
```text
hello alice, request 1
```

This proves that the same client gets a fresh rate-limit window after its Redis key expires.

## Shutting down the system

The containers were started in detached mode, so they continue running in the background until they are stopped.

When finished, stop both project containers:

```bash
docker stop rate-limiter-api rate-limiter-redis
```

Confirm that no project containers are still running:
```bash
docker ps
```

The stopped containers still exist and can be viewed with:
```bash
docker ps -a
```

## Replacing the API container

The rate-limit counters are stored in Redis, not in the Go API container.

If the API container is removed and recreated while the Redis container keeps running, the current counters remain available until their 60-second expiry.

For example, if a client reaches request 3, replacing only the API container should make that client's next request return request 4, provided the Redis key has not expired.

## Limitations

This is a small learning project, not a production rate limiter.

- Client identity comes directly from `X-Client-ID` and is not authenticated.
- The limiter uses a simple fixed 60-second window.
- Blocked requests still increment the Redis counter.
- Redis persistence and backup are outside the scope of this project.
- There is no TLS, authentication, reverse proxy, CI, or cloud deployment.
- The setup is intended for local Docker use.