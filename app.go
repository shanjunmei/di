package di

import (
	"context"
)

type Option interface {
	apply(*container)
}

type provideOption struct{ constructor any }

func (o provideOption) apply(c *container) { c.Provide(o.constructor) }

func Provide(constructor any) Option { return provideOption{constructor: constructor} }

type invokeOption struct{ fn any }

func (o invokeOption) apply(c *container) { c.Invoke(o.fn) }

func Invoke(fn any) Option { return invokeOption{fn: fn} }

type moduleOption struct{ opts []Option }

func (o moduleOption) apply(c *container) {
	for _, opt := range o.opts {
		opt.apply(c)
	}
}
func Module(opts ...Option) Option { return moduleOption{opts: opts} }

type App struct {
	container *container
	opts      []Option
}

func New(opts ...Option) *App {
	return &App{
		container: newContainer(),
		opts:      opts,
	}
}

func (app *App) Run(ctx context.Context) error {
	for _, opt := range app.opts {
		opt.apply(app.container)
	}
	return app.container.Run(ctx)
}
