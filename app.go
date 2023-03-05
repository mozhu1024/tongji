package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("ADDR") + ":6379",
		Password: os.Getenv("PASSWORD"),
		DB:       0,
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_id := r.URL.Query().Get("id")
		if _id == "" {
			w.Write([]byte("Err"))
			return
		}
		// 计数
		_ = rdb.Incr(ctx, _id).Err()
		// 保存信息
		infoKey := fmt.Sprintf("info_%s", _id)
		if rdb.Exists(ctx, infoKey).Val() == 0 {
			_ = rdb.Set(ctx, infoKey,
				// ver-os-arch-ln-name
				fmt.Sprintf("%s-%s-%s-%s-%s",
					r.URL.Query().Get("ver"),
					r.URL.Query().Get("os"),
					r.URL.Query().Get("arch"),
					r.URL.Query().Get("ln"),
					r.URL.Query().Get("name"),
				),
				0,
			)
		}

		w.Write([]byte("OK"))
	})

	log.Printf("Listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
