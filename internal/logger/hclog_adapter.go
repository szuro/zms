package logger

import (
	"io"
	"log"

	"github.com/hashicorp/go-hclog"
)

// HCLogAdapter adapts ZMSLogger to implement hashicorp/go-hclog.Logger interface.
// This is used to integrate HashiCorp go-plugin logging with ZMS logging system.
type HCLogAdapter struct {
	logger *ZMSLogger
	name   string
}

// NewHCLogAdapter creates a new HCLog adapter wrapping the default ZMS logger.
func NewHCLogAdapter() hclog.Logger {
	return &HCLogAdapter{
		logger: Default(),
		name:   "plugin",
	}
}

// Log implementation
func (h *HCLogAdapter) Log(level hclog.Level, msg string, args ...interface{}) {
	switch level {
	case hclog.Trace, hclog.Debug:
		h.logger.Debug(msg, args...)
	case hclog.Info:
		h.logger.Info(msg, args...)
	case hclog.Warn:
		h.logger.Warn(msg, args...)
	case hclog.Error:
		h.logger.Error(msg, args...)
	}
}

// Trace logs at trace level
func (h *HCLogAdapter) Trace(msg string, args ...interface{}) {
	h.logger.Debug(msg, args...)
}

// Debug logs at debug level
func (h *HCLogAdapter) Debug(msg string, args ...interface{}) {
	h.logger.Debug(msg, args...)
}

// Info logs at info level
func (h *HCLogAdapter) Info(msg string, args ...interface{}) {
	h.logger.Info(msg, args...)
}

// Warn logs at warn level
func (h *HCLogAdapter) Warn(msg string, args ...interface{}) {
	h.logger.Warn(msg, args...)
}

// Error logs at error level
func (h *HCLogAdapter) Error(msg string, args ...interface{}) {
	h.logger.Error(msg, args...)
}

// IsTrace returns true if trace level is enabled
func (h *HCLogAdapter) IsTrace() bool {
	return false
}

// IsDebug returns true if debug level is enabled
func (h *HCLogAdapter) IsDebug() bool {
	return true
}

// IsInfo returns true if info level is enabled
func (h *HCLogAdapter) IsInfo() bool {
	return true
}

// IsWarn returns true if warn level is enabled
func (h *HCLogAdapter) IsWarn() bool {
	return true
}

// IsError returns true if error level is enabled
func (h *HCLogAdapter) IsError() bool {
	return true
}

// ImpliedArgs returns implied args (not used)
func (h *HCLogAdapter) ImpliedArgs() []interface{} {
	return nil
}

// With creates a new logger with additional context
func (h *HCLogAdapter) With(args ...interface{}) hclog.Logger {
	return &HCLogAdapter{
		logger: h.logger,
		name:   h.name,
	}
}

// Name returns the logger name
func (h *HCLogAdapter) Name() string {
	return h.name
}

// Named creates a new logger with a name
func (h *HCLogAdapter) Named(name string) hclog.Logger {
	return &HCLogAdapter{
		logger: h.logger,
		name:   h.name + "." + name,
	}
}

// ResetNamed creates a new logger with the given name, clearing parent names
func (h *HCLogAdapter) ResetNamed(name string) hclog.Logger {
	return &HCLogAdapter{
		logger: h.logger,
		name:   name,
	}
}

// SetLevel sets the log level (no-op in this adapter)
func (h *HCLogAdapter) SetLevel(level hclog.Level) {}

// GetLevel returns the current log level
func (h *HCLogAdapter) GetLevel() hclog.Level {
	return hclog.Info
}

// StandardLogger returns a standard library logger
func (h *HCLogAdapter) StandardLogger(opts *hclog.StandardLoggerOptions) *log.Logger {
	return log.Default()
}

// StandardWriter returns a writer for standard logging
func (h *HCLogAdapter) StandardWriter(opts *hclog.StandardLoggerOptions) io.Writer {
	return io.Discard
}
