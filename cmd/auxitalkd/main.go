package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Brook-sys/auxitalk/internal/runtime"
)

func main() {
	r := runtime.New(runtime.Options{
		Name:    "auxitalkd",
		Version: "0.1.0-dev",
	})

	if err := r.Run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "auxitalkd: %v\n", err)
		os.Exit(1)
	}
}
