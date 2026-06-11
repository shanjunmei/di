package di

import (
	"context"
)

type Option interface {
	apply(*container)
}

type provideOption struct{ constructors []any }

func (o provideOption) apply(c *container) {
	for _, ctor := range o.constructors {
		c.Provide(ctor)
	}

}

func Provide(constructors ...any) Option { return provideOption{constructors: constructors} }

type invokeOption struct{ fns []any }

func (o invokeOption) apply(c *container) {
	for _, fn := range o.fns {
		c.Invoke(fn)
	}

}

func Invoke(fns ...any) Option { return invokeOption{fns: fns} }

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
