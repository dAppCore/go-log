// SPDX-License-Identifier: EUPL-1.2

// Service registration for the go-log package. Wraps a Logger as a
// Core service so consumers can dispatch log writes through the same
// Action plumbing as every other domain.
//
//	c, _ := core.New(
//	    core.WithName("log", golog.NewService(golog.Options{
//	        Level: golog.LevelInfo,
//	    })),
//	)
//	r := c.Action("log.info").Run(ctx, core.NewOptions(
//	    core.Option{Key: "msg", Value: "ready"},
//	    core.Option{Key: "component", Value: "ax-10"},
//	))

package golog

import (
	"context"

	core "dappco.re/go"
)

// Service is the registerable handle for the go-log package — embeds
// *core.ServiceRuntime[Options] for typed options access and holds a
// live *Logger ready for direct method calls or action use.
//
//	svc := core.MustServiceFor[*golog.Service](c, "log")
//	svc.Logger.Info("ready", "component", "ax-10")
type Service struct {
	*core.ServiceRuntime[Options]
	// Logger is the live *Logger the service was constructed with.
	//
	//	svc.Logger.Warn("slow", "ms", 850)
	Logger        *Logger
	registrations core.Once
}

// NewService returns a factory that builds a Logger from Options and
// produces a *Service ready for c.Service() registration.
//
//	c, _ := core.New(
//	    core.WithName("log", golog.NewService(golog.Options{
//	        Level: golog.LevelDebug,
//	    })),
//	)
func NewService(opts Options) func(*core.Core) core.Result {
	return func(c *core.Core) core.Result {
		return core.Ok(&Service{
			ServiceRuntime: core.NewServiceRuntime(c, opts),
			Logger:         New(opts),
		})
	}
}

// Register builds the log service with default Options and returns
// the service Result directly — the imperative-style alternative to
// NewService for consumers wiring services without WithName options.
//
//	r := golog.Register(c)
//	svc := r.Value.(*golog.Service)
func Register(c *core.Core) core.Result {
	return NewService(Options{})(c)
}

// OnStartup registers the log action handlers on the attached Core.
// Implements core.Startable. Idempotent via core.Once.
//
//	r := svc.OnStartup(ctx)
func (s *Service) OnStartup(context.Context) core.Result {
	if s == nil {
		return core.Ok(nil)
	}
	s.registrations.Do(func() {
		c := s.Core()
		if c == nil {
			return
		}
		c.Action("log.debug", s.handleDebug)
		c.Action("log.info", s.handleInfo)
		c.Action("log.warn", s.handleWarn)
		c.Action("log.error", s.handleError)
		c.Action("log.security", s.handleSecurity)
		c.Action("log.set_level", s.handleSetLevel)
	})
	return core.Ok(nil)
}

// OnShutdown is a no-op — the Logger holds no closable resources;
// rotation writers are flushed on process exit. Implements
// core.Stoppable.
//
//	r := svc.OnShutdown(ctx)
func (s *Service) OnShutdown(context.Context) core.Result {
	return core.Ok(nil)
}

// keyvalsFromOptions extracts an optional single key/value pair from
// Options (Action shape: msg + optional key + value). Callers needing
// arbitrary keyvals invoke svc.Logger.Info(msg, k1, v1, k2, v2, ...)
// directly via the Service handle rather than through the Action
// surface.
func keyvalsFromOptions(opts core.Options) []any {
	if !opts.Has("key") {
		return nil
	}
	value := opts.Get("value")
	if !value.OK {
		return []any{opts.String("key"), ""}
	}
	return []any{opts.String("key"), value.Value}
}

// handleDebug — `log.debug` action handler. Reads opts.msg and emits
// it at debug level; remaining keys become structured keyvals.
//
//	r := c.Action("log.debug").Run(ctx, core.NewOptions(
//	    core.Option{Key: "msg", Value: "cache miss"},
//	    core.Option{Key: "key", Value: "user/42"},
//	))
func (s *Service) handleDebug(_ core.Context, opts core.Options) core.Result {
	if s == nil || s.Logger == nil {
		return core.Fail(core.E("log.debug", "service not initialised", nil))
	}
	s.Logger.Debug(opts.String("msg"), keyvalsFromOptions(opts)...)
	return core.Ok(nil)
}

// handleInfo — `log.info` action handler. Reads opts.msg and emits
// it at info level; remaining keys become structured keyvals.
//
//	r := c.Action("log.info").Run(ctx, core.NewOptions(
//	    core.Option{Key: "msg", Value: "ready"},
//	    core.Option{Key: "component", Value: "ax-10"},
//	))
func (s *Service) handleInfo(_ core.Context, opts core.Options) core.Result {
	if s == nil || s.Logger == nil {
		return core.Fail(core.E("log.info", "service not initialised", nil))
	}
	s.Logger.Info(opts.String("msg"), keyvalsFromOptions(opts)...)
	return core.Ok(nil)
}

// handleWarn — `log.warn` action handler. Reads opts.msg and emits
// it at warn level; remaining keys become structured keyvals.
//
//	r := c.Action("log.warn").Run(ctx, core.NewOptions(
//	    core.Option{Key: "msg", Value: "slow route"},
//	    core.Option{Key: "ms", Value: 850},
//	))
func (s *Service) handleWarn(_ core.Context, opts core.Options) core.Result {
	if s == nil || s.Logger == nil {
		return core.Fail(core.E("log.warn", "service not initialised", nil))
	}
	s.Logger.Warn(opts.String("msg"), keyvalsFromOptions(opts)...)
	return core.Ok(nil)
}

// handleError — `log.error` action handler. Reads opts.msg and emits
// it at error level; remaining keys become structured keyvals.
//
//	r := c.Action("log.error").Run(ctx, core.NewOptions(
//	    core.Option{Key: "msg", Value: "request failed"},
//	    core.Option{Key: "err", Value: err.Error()},
//	))
func (s *Service) handleError(_ core.Context, opts core.Options) core.Result {
	if s == nil || s.Logger == nil {
		return core.Fail(core.E("log.error", "service not initialised", nil))
	}
	s.Logger.Error(opts.String("msg"), keyvalsFromOptions(opts)...)
	return core.Ok(nil)
}

// handleSecurity — `log.security` action handler. Reads opts.msg and
// emits it at security level; remaining keys become structured
// keyvals.
//
//	r := c.Action("log.security").Run(ctx, core.NewOptions(
//	    core.Option{Key: "msg", Value: "suspicious entry"},
//	    core.Option{Key: "ip", Value: "127.0.0.1"},
//	))
func (s *Service) handleSecurity(_ core.Context, opts core.Options) core.Result {
	if s == nil || s.Logger == nil {
		return core.Fail(core.E("log.security", "service not initialised", nil))
	}
	s.Logger.Security(opts.String("msg"), keyvalsFromOptions(opts)...)
	return core.Ok(nil)
}

// handleSetLevel — `log.set_level` action handler. Reads opts.level
// (int) and updates the underlying Logger's level filter.
//
//	r := c.Action("log.set_level").Run(ctx, core.NewOptions(
//	    core.Option{Key: "level", Value: int(golog.LevelDebug)},
//	))
func (s *Service) handleSetLevel(_ core.Context, opts core.Options) core.Result {
	if s == nil || s.Logger == nil {
		return core.Fail(core.E("log.set_level", "service not initialised", nil))
	}
	s.Logger.SetLevel(Level(opts.Int("level")))
	return core.Ok(nil)
}
