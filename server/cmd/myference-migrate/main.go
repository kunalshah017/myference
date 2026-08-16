package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"

	"github.com/kunalshah017/myference/server/internal/store"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	if err := run(); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	databaseURL := os.Getenv("MYFERENCE_DATABASE_URL")
	if databaseURL == "" {
		return errors.New("MYFERENCE_DATABASE_URL is required")
	}
	paths, err := filepath.Glob("migrations/*.sql")
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return errors.New("no SQL migrations found in migrations/")
	}
	sort.Strings(paths)

	ctx := context.Background()
	repository, err := store.Open(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer repository.Close()
	for _, path := range paths {
		if err := repository.ApplyMigration(ctx, path); err != nil {
			return fmt.Errorf("apply %s: %w", filepath.Base(path), err)
		}
		slog.Info("applied migration", "file", filepath.Base(path))
	}
	return nil
}
