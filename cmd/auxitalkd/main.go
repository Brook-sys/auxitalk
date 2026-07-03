package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Brook-sys/auxitalk/internal/config"
	"github.com/Brook-sys/auxitalk/internal/logger"
	"github.com/Brook-sys/auxitalk/internal/runtime"
)

func main() {
	configPath := flag.String("config", "", "path to auxitalk config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "auxitalkd: %v\n", err)
		os.Exit(1)
	}

	if err := logger.Init(cfg.Runtime.LogPath); err != nil {
		fmt.Fprintf(os.Stderr, "auxitalkd logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	r := runtime.New(runtime.Options{
		Name:    "auxitalkd",
		Version: "0.1.0-dev",
		Config:  cfg,
	})

	if err := r.Run(ctx); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "auxitalkd: %v\n", err)
		os.Exit(1)
	}
}
