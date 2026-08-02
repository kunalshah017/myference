package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/kunalshah017/myference/server/internal/store"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
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
		log.Printf("applied %s", filepath.Base(path))
	}
	return nil
}
