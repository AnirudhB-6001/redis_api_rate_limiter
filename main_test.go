package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestAllowsFirstFiveRequestsAndRejectsSixth(t *testing.T) {
	rdb = redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})

	clientID := "test-limit"
	key := "rate:" + clientID

	rdb.Del(context.Background(), key)
	t.Cleanup(func() {
		rdb.Del(context.Background(), key)
	})

	for i := 1; i <= 6; i++ {
		req := httptest.NewRequest(http.MethodGet, "/hello", nil)
		req.Header.Set("X-Client-ID", clientID)

		rec := httptest.NewRecorder()

		helloHandler(rec, req)

		if i <= 5 && rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, rec.Code)
		}

		if i == 6 && rec.Code != http.StatusTooManyRequests {
			t.Fatalf("request 6: expected 429, got %d", rec.Code)
		}
	}
}

func TestRejectsMissingClientID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	rec := httptest.NewRecorder()

	helloHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestClientsHaveIndependentLimits(t *testing.T) {
	rdb = redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})

	ctx := context.Background()
	aliceKey := "rate:test-alice"
	bobKey := "rate:test-bob"

	rdb.Del(ctx, aliceKey, bobKey)
	t.Cleanup(func() {
		rdb.Del(ctx, aliceKey, bobKey)
	})

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/hello", nil)
		req.Header.Set("X-Client-ID", "test-alice")
		rec := httptest.NewRecorder()

		helloHandler(rec, req)
	}

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	req.Header.Set("X-Client-ID", "test-bob")
	rec := httptest.NewRecorder()

	helloHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected bob's first request to return 200, got %d", rec.Code)
	}
}
