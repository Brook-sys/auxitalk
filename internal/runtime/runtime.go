package runtime

import (
	"context"
	"fmt"

	"github.com/Brook-sys/auxitalk/internal/config"
)

type Options struct {
	Name    string
	Version string
	Config  config.Config
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
		fmt.Printf("%s %s mode=%s\n", r.options.Name, r.options.Version, r.options.Config.Mode)
		return nil
	}
}
