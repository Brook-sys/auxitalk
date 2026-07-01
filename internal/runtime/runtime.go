package runtime

import (
	"context"
	"fmt"
)

type Options struct {
	Name    string
	Version string
}

type Runtime struct {
	options Options
}

func New(options Options) *Runtime {
	return &Runtime{options: options}
}

func (r *Runtime) Run(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		fmt.Printf("%s %s\n", r.options.Name, r.options.Version)
		return nil
	}
}
