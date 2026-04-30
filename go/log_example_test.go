package golog_test

import (
	core "dappco.re/go"
	golog "dappco.re/go/log"
)

func ExampleLevel_String() {
	name := golog.LevelInfo.String()
	if name == "" {
		panic("missing level")
	}
}

func ExampleNew() {
	buf := core.NewBuffer()
	logger := golog.New(golog.Options{Level: golog.LevelInfo, Output: buf})
	logger.Info("ready")
	if !core.Contains(buf.String(), "ready") {
		panic("missing entry")
	}
}

func ExampleLogger_SetLevel() {
	logger := golog.New(golog.Options{Level: golog.LevelInfo, Output: core.Discard})
	logger.SetLevel(golog.LevelDebug)
	if logger.Level() != golog.LevelDebug {
		panic("missing level")
	}
}

func ExampleLogger_Level() {
	logger := golog.New(golog.Options{Level: golog.LevelWarn, Output: core.Discard})
	if logger.Level() != golog.LevelWarn {
		panic("missing level")
	}
}

func ExampleLogger_SetOutput() {
	buf := core.NewBuffer()
	logger := golog.New(golog.Options{Level: golog.LevelInfo, Output: core.Discard})
	logger.SetOutput(buf)
	logger.Info("redirected")
	if !core.Contains(buf.String(), "redirected") {
		panic("missing output")
	}
}

func ExampleLogger_SetRedactKeys() {
	buf := core.NewBuffer()
	logger := golog.New(golog.Options{Level: golog.LevelInfo, Output: buf})
	logger.SetRedactKeys("token")
	logger.Info("entry", "token", "secret")
	if !core.Contains(buf.String(), "[REDACTED]") {
		panic("missing redaction")
	}
}

func ExampleLogger_Debug() {
	buf := core.NewBuffer()
	logger := golog.New(golog.Options{Level: golog.LevelDebug, Output: buf})
	logger.Debug("debug")
	if !core.Contains(buf.String(), "[DBG]") {
		panic("missing debug")
	}
}

func ExampleLogger_Info() {
	buf := core.NewBuffer()
	logger := golog.New(golog.Options{Level: golog.LevelInfo, Output: buf})
	logger.Info("info")
	if !core.Contains(buf.String(), "[INF]") {
		panic("missing info")
	}
}

func ExampleLogger_Warn() {
	buf := core.NewBuffer()
	logger := golog.New(golog.Options{Level: golog.LevelWarn, Output: buf})
	logger.Warn("warn")
	if !core.Contains(buf.String(), "[WRN]") {
		panic("missing warn")
	}
}

func ExampleLogger_Error() {
	buf := core.NewBuffer()
	logger := golog.New(golog.Options{Level: golog.LevelError, Output: buf})
	logger.Error("failed", "err", core.NewError("boom"))
	if !core.Contains(buf.String(), "[ERR]") {
		panic("missing failure")
	}
}

func ExampleLogger_Security() {
	buf := core.NewBuffer()
	logger := golog.New(golog.Options{Level: golog.LevelError, Output: buf})
	logger.Security("entry")
	if !core.Contains(buf.String(), "[SEC]") {
		panic("missing security")
	}
}

func ExampleUsername() {
	name := golog.Username()
	if name == "" {
		panic("missing user")
	}
}

func ExampleDefault() {
	logger := golog.Default()
	if logger == nil {
		panic("missing default")
	}
}

func ExampleSetDefault() {
	original := golog.Default()
	buf := core.NewBuffer()
	golog.SetDefault(golog.New(golog.Options{Level: golog.LevelInfo, Output: buf}))
	defer golog.SetDefault(original)
	golog.Info("default")
	if !core.Contains(buf.String(), "default") {
		panic("missing default")
	}
}

func ExampleSetLevel() {
	original := golog.Default()
	logger := golog.New(golog.Options{Level: golog.LevelInfo, Output: core.Discard})
	golog.SetDefault(logger)
	defer golog.SetDefault(original)
	golog.SetLevel(golog.LevelDebug)
	if logger.Level() != golog.LevelDebug {
		panic("missing level")
	}
}

func ExampleSetRedactKeys() {
	original := golog.Default()
	buf := core.NewBuffer()
	golog.SetDefault(golog.New(golog.Options{Level: golog.LevelInfo, Output: buf}))
	defer golog.SetDefault(original)
	golog.SetRedactKeys("token")
	golog.Info("entry", "token", "secret")
	if !core.Contains(buf.String(), "[REDACTED]") {
		panic("missing redaction")
	}
}

func ExampleDebug() {
	original := golog.Default()
	buf := core.NewBuffer()
	golog.SetDefault(golog.New(golog.Options{Level: golog.LevelDebug, Output: buf}))
	defer golog.SetDefault(original)
	golog.Debug("debug")
	if !core.Contains(buf.String(), "[DBG]") {
		panic("missing debug")
	}
}

func ExampleInfo() {
	original := golog.Default()
	buf := core.NewBuffer()
	golog.SetDefault(golog.New(golog.Options{Level: golog.LevelInfo, Output: buf}))
	defer golog.SetDefault(original)
	golog.Info("info")
	if !core.Contains(buf.String(), "[INF]") {
		panic("missing info")
	}
}

func ExampleWarn() {
	original := golog.Default()
	buf := core.NewBuffer()
	golog.SetDefault(golog.New(golog.Options{Level: golog.LevelDebug, Output: buf}))
	defer golog.SetDefault(original)
	golog.Warn("warn")
	if !core.Contains(buf.String(), "[WRN]") {
		panic("missing warn")
	}
}

func ExampleError() {
	original := golog.Default()
	buf := core.NewBuffer()
	golog.SetDefault(golog.New(golog.Options{Level: golog.LevelDebug, Output: buf}))
	defer golog.SetDefault(original)
	golog.Error("failed", "err", core.NewError("boom"))
	if !core.Contains(buf.String(), "[ERR]") {
		panic("missing failure")
	}
}

func ExampleSecurity() {
	original := golog.Default()
	buf := core.NewBuffer()
	golog.SetDefault(golog.New(golog.Options{Level: golog.LevelDebug, Output: buf}))
	defer golog.SetDefault(original)
	golog.Security("entry")
	if !core.Contains(buf.String(), "[SEC]") {
		panic("missing security")
	}
}
