package main

import (
	"context"

	"github.com/lowplane/sevro/internal/config"
)

// colorPolicyKey is the context key under which the resolved
// colour-policy boolean is stashed by the root command. Using a
// distinct unexported type prevents collisions with anything else
// the cobra command tree might tuck into the context.
type colorPolicyKey struct{}

// withColorPolicy returns a context that carries the resolved
// "should we emit ANSI?" decision.
func withColorPolicy(ctx context.Context, useColor bool) context.Context {
	return context.WithValue(ctx, colorPolicyKey{}, useColor)
}

// colorPolicyFrom recovers the colour-policy decision; defaults to
// false (plain) so subcommands that bypass the persistent pre-run
// fall back to safe-by-default plain output.
func colorPolicyFrom(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, ok := ctx.Value(colorPolicyKey{}).(bool)
	if !ok {
		return false
	}
	return v
}

// configKey is the context key for the loaded .sevro.yaml.
type configKey struct{}

// withConfig stashes the loaded Config in ctx.
func withConfig(ctx context.Context, c config.Config) context.Context {
	return context.WithValue(ctx, configKey{}, c)
}

// configFrom recovers the Config; returns the zero Config when none
// is set so callers don't need to nil-check.
func configFrom(ctx context.Context) config.Config {
	if ctx == nil {
		return config.Config{}
	}
	v, ok := ctx.Value(configKey{}).(config.Config)
	if !ok {
		return config.Config{}
	}
	return v
}
