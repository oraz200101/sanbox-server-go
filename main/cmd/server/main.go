package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/oraz200101/sandbox-server/main/internal/broker"
	"github.com/oraz200101/sandbox-server/main/internal/config"
	"github.com/oraz200101/sandbox-server/main/internal/database"
)

func main() {
	// 1. Загрузи конфиг СНАЧАЛА
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Error loading config:", err)
	}

	// 2. Инициализируй все клиенты
	pgClient, err := database.NewPostgresClient(cfg)
	if err != nil {
		log.Fatal("postgres connection failed:", err)
	}
	defer pgClient.Close()
	log.Println("✓ PostgreSQL connected")

	mongoClient, err := database.NewMongoDBClient(cfg)
	if err != nil {
		log.Fatal("mongodb connection failed:", err)
	}
	defer mongoClient.Close()
	log.Println("✓ MongoDB connected")

	kafkaClient, err := broker.NewKafkaClient(cfg)
	if err != nil {
		log.Fatal("kafka connection failed:", err)
	}
	defer kafkaClient.Close()
	log.Println("✓ Kafka connected")

	esClient, err := database.NewElasticsearchClient(cfg)
	if err != nil {
		log.Fatal("elasticsearch connection failed:", err)
	}
	defer esClient.Close()
	log.Println("✓ Elasticsearch connected")

	log.Println("All services ready!")

	// 3. Создай routes
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from Exam Platform")
	})

	// 4. Запусти сервер в горутине
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: mux,
	}

	go func() {
		log.Printf("Server starting on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// 5. Обрабатывай shutdown сигналы
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("Shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}

	log.Println("Server stopped gracefully")
}
