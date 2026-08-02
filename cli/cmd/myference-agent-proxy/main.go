package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kunalshah017/myference/cli/internal/backend/command"
)

func main() {
	configuration := flag.String("config", "", "proxy configuration file")
	health := flag.Bool("health", false, "check the local proxy")
	flag.Parse()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	var err error
	if *health {
		err = command.ProxyHealth(ctx)
	} else if *configuration == "" {
		err = fmt.Errorf("--config is required")
	} else {
		err = command.ServeProxy(ctx, *configuration)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
