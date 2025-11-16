package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"example.com/pz4-todo/internal/task"
	myMW "example.com/pz4-todo/pkg/middleware"
)

func main() {
	repo := task.NewRepo()
	if err := repo.LoadFromFile("tasks.json"); err != nil {
		log.Printf("Warning: could not load tasks from file: %v", err)
	}

	h := task.NewHandler(repo)

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.Recoverer)
	r.Use(myMW.Logger)
	r.Use(myMW.SimpleCORS)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	r.Route("/api/v1", func(api chi.Router) {
		api.Mount("/tasks", h.Routes())
	})

	server := &http.Server{
		Addr: ":8080",
		Handler: r,
		ReadTimeout: 15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Создаем канал для получения ошибок сервера
	serverErr := make(chan error, 1)
	
	// Запускаем сервер в горутине
	go func() {
		log.Printf("🚀 Server starting on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Создам канал для сингналов ОС
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	select {
	case <-stop:
		log.Println("📞 Received shutdown signal")
	case err := <-serverErr:
		log.Printf("❌ Server error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Останавливаем сервер
	log.Println("🛑 Shutting down server gracefully...")
	
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("⚠️ Graceful shutdown failed: %v", err)
		log.Println("🔄 Forcing server close...")
		if err := server.Close(); err != nil {
			log.Printf("❌ Force close failed: %v", err)
		}
	} else {
		log.Println("✅ Server stopped gracefully")
	}
}
