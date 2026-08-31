package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
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

	count, err := rdb.Incr(context.Background(), "rate:"+clientID).Result()
	if err != nil {
		http.Error(w, "Redis error", http.StatusInternalServerError)
		return
	}

	if count == 1 {
		if err := rdb.Expire(context.Background(), "rate:"+clientID, 60*time.Second).Err(); err != nil {
			http.Error(w, "Redis error", http.StatusInternalServerError)
			return
		}
	}

	if count > 5 {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	fmt.Fprintf(w, "hello %s, request %d\n", clientID, count)
}

func main() {
	rdb = redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("connected to Redis")

	http.HandleFunc("/hello", helloHandler)

	fmt.Println("server listening on :8080")
	http.ListenAndServe(":8080", nil)
}
