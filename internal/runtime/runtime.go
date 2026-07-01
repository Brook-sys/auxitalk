package runtime

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/Brook-sys/auxitalk/internal/config"
	"github.com/Brook-sys/auxitalk/internal/plugins"
	"github.com/Brook-sys/auxitalk/internal/plugins/supervisor"
)

type Options struct {
	Name    string
	Version string
	Config  config.Config
}

type Runtime struct {
	options    Options
	supervisor *supervisor.Supervisor
}

func New(options Options) *Runtime {
	sup := supervisor.NewSupervisor(supervisor.ProcessOptions{
		CallTimeout:    options.Config.Runtime.RequestTimeout.Std(),
		HealthInterval: 30 * time.Second,
		RestartBackoff: time.Second,
		MaxRestarts:    3,
		MaxPayloadSize: int(options.Config.Runtime.MaxPayloadSize),
		OnLog: func(pluginID string, line string) {
			fmt.Printf("[%s] %s\n", pluginID, line)
		},
		OnRequest: func(req supervisor.ProcessRequest) {
			fmt.Printf("[%s] plugin request: %s\n", req.PluginID, req.Method)
		},
	})

	return &Runtime{options: options, supervisor: sup}
}

func (r *Runtime) Run(ctx context.Context) error {
	fmt.Printf("%s %s mode=%s\n", r.options.Name, r.options.Version, r.options.Config.Mode)

	if err := r.loadPlugins(ctx); err != nil {
		return err
	}

	<-ctx.Done()
	return r.shutdown()
}

func (r *Runtime) loadPlugins(ctx context.Context) error {
	for _, pluginConfig := range r.options.Config.Plugins {
		if !pluginConfig.Enabled {
			continue
		}
		if pluginConfig.Manifest == "" {
			continue
		}

		manifestFile, err := plugins.LoadManifest(pluginConfig.Manifest)
		if err != nil {
			return err
		}

		command := manifestFile.Manifest.Runtime
		args := []string{}

		if manifestFile.Manifest.Entry != "" {
			entry := manifestFile.Manifest.Entry
			if !filepath.IsAbs(entry) {
				entry = filepath.Join(manifestFile.Dir, entry)
			}
			args = append(args, entry)
		} else if !filepath.IsAbs(command) {
			command = filepath.Join(manifestFile.Dir, command)
		}

		spec := supervisor.ProcessSpec{
			ID:      manifestFile.Manifest.ID,
			Command: command,
			Args:    args,
			Dir:     manifestFile.Dir,
		}

		if err := r.supervisor.Register(spec); err != nil {
			return err
		}
		if err := r.supervisor.Start(ctx, manifestFile.Manifest.ID); err != nil {
			return err
		}

		fmt.Printf("plugin started: %s\n", manifestFile.Manifest.ID)
	}
	return nil
}

func (r *Runtime) shutdown() error {
	for _, id := range r.supervisor.List() {
		_, _ = r.supervisor.Call(context.Background(), id, "plugin.stop", nil)
		if err := r.supervisor.Stop(id); err != nil {
			fmt.Printf("plugin stop error %s: %v\n", id, err)
		}
	}
	return nil
}
