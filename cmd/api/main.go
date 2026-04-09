package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	infrastructurepostgres "example.com/taskservice/internal/infrastructure/postgres"
	postgresrepo "example.com/taskservice/internal/repository/postgres"
	transporthttp "example.com/taskservice/internal/transport/http"
	swaggerdocs "example.com/taskservice/internal/transport/http/docs"
	httphandlers "example.com/taskservice/internal/transport/http/handlers"
	"example.com/taskservice/internal/usecase/task"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg := loadConfig()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := infrastructurepostgres.Open(ctx, cfg.DatabaseDSN)
	if err != nil {
		logger.Error("open postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	taskRepo := postgresrepo.New(pool)
	taskUsecase := task.NewService(taskRepo)
	taskHandler := httphandlers.NewTaskHandler(taskUsecase)
	docsHandler := swaggerdocs.NewHandler()
	router := transporthttp.NewRouter(taskHandler, docsHandler)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go startDailyBackgroundWorker(ctx, logger, taskUsecase)

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown http server", "error", err)
		}
	}()

	logger.Info("http server started", "addr", cfg.HTTPAddr)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("listen and serve", "error", err)
		os.Exit(1)
	}
}

func startDailyBackgroundWorker(ctx context.Context, logger *slog.Logger, taskUsecase *task.Service) {
	runTaskUpdate(logger, taskUsecase)

	logger.Info("daily background worker started, will run at 00:00 every day")

	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if now.After(next) {
		next = next.Add(24 * time.Hour)
	}

	select {
	case <-ctx.Done():
		logger.Info("daily background worker stopped")
		return
	case <-time.After(next.Sub(now)):
		runTaskUpdate(logger, taskUsecase)

		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				logger.Info("daily background worker stopped")
				return
			case <-ticker.C:
				runTaskUpdate(logger, taskUsecase)
			}
		}
	}
}

func runTaskUpdate(logger *slog.Logger, taskUsecase *task.Service) {
	logger.Info("starting daily task update")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	tasks, err := taskUsecase.GetTasksForUpdate(ctx)
	if err != nil {
		logger.Error("failed to get tasks for update", "error", err)
		return
	}

	if len(tasks) == 0 {
		logger.Info("no tasks to update")
		return
	}

	logger.Info("found tasks to update", "count", len(tasks))

	updatedCount := 0
	for _, t := range tasks {
		if err := taskUsecase.CreatePeriodicTask(ctx, &t); err != nil {
			logger.Error("failed to update task", "task_id", t.ID, "error", err)
			continue
		}
		updatedCount++
	}

	logger.Info("daily task update completed",
		"updated", updatedCount)
}

type config struct {
	HTTPAddr    string
	DatabaseDSN string
}

func loadConfig() config {
	cfg := config{
		HTTPAddr:    envOrDefault("HTTP_ADDR", ":8080"),
		DatabaseDSN: envOrDefault("DATABASE_DSN", "postgres://postgres:postgres@postgres:1111/taskservice?sslmode=disable"),
	}

	if cfg.DatabaseDSN == "" {
		panic(fmt.Errorf("DATABASE_DSN is required"))
	}

	return cfg
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
