package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var rdb *redis.Client

func helloHandler(w http.ResponseWriter, r *http.Request) {
	clientID := r.Header.Get("X-Client-ID")

	if clientID == "" {
		http.Error(w, "missing X-Client-ID", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	key := "rate:" + clientID

	pipe := rdb.TxPipeline()

	countCmd := pipe.Incr(ctx, key)
	pipe.ExpireNX(ctx, key, 60*time.Second)

	if _, err := pipe.Exec(ctx); err != nil {
		http.Error(w, "Redis error", http.StatusInternalServerError)
		return
	}

	count := countCmd.Val()

	if count > 5 {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	fmt.Fprintf(w, "hello %s, request %d\n", clientID, count)
}

func main() {
	redisAddr := os.Getenv("REDIS_ADDR")

	if redisAddr == "" {
		redisAddr = "127.0.0.1:6379"
	}

	rdb = redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("connected to Redis")

	http.HandleFunc("/hello", helloHandler)

	fmt.Println("server listening on :8080")
	http.ListenAndServe(":8080", nil)
}
