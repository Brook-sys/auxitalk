package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/Brook-sys/auxitalk/internal/config"
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

	r := runtime.New(runtime.Options{
		Name:    "auxitalkd",
		Version: "0.1.0-dev",
		Config:  cfg,
	})

	if err := r.Run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "auxitalkd: %v\n", err)
		os.Exit(1)
	}
}
