// SPDX-License-Identifier: Apache-2.0

package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/go-hclog"

	"github.com/charmbracelet/lipgloss"
	charmlog "github.com/charmbracelet/log"
)

// Initializing the colors for the charm logger.
var (
	DebugColor = lipgloss.AdaptiveColor{Light: "63", Dark: "63"}
	InfoColor  = lipgloss.AdaptiveColor{Light: "74", Dark: "86"}
	WarnColor  = lipgloss.AdaptiveColor{Light: "214", Dark: "192"}
	ErrorColor = lipgloss.AdaptiveColor{Light: "203", Dark: "203"}
	FatalColor = lipgloss.AdaptiveColor{Light: "134", Dark: "134"}
)

func defaultOptions() *charmlog.Options {
	return &charmlog.Options{
		ReportCaller:    false,
		ReportTimestamp: false,
	}
}

// Setup charm styles to use for logger.
func defaultStyles() *charmlog.Styles {
	styles := charmlog.DefaultStyles()

	styles.Levels = map[charmlog.Level]lipgloss.Style{
		charmlog.DebugLevel: lipgloss.NewStyle().
			SetString(strings.ToUpper(charmlog.DebugLevel.String())).
			Foreground(DebugColor).
			Faint(true),
		charmlog.InfoLevel: lipgloss.NewStyle().
			SetString(strings.ToUpper(charmlog.InfoLevel.String())).
			Foreground(InfoColor),
		charmlog.WarnLevel: lipgloss.NewStyle().
			SetString(strings.ToUpper(charmlog.WarnLevel.String())).
			Foreground(WarnColor),
		charmlog.ErrorLevel: lipgloss.NewStyle().
			SetString(strings.ToUpper(charmlog.ErrorLevel.String())).
			Foreground(ErrorColor),
		charmlog.FatalLevel: lipgloss.NewStyle().
			SetString(strings.ToUpper(charmlog.FatalLevel.String())).
			Foreground(FatalColor).
			Bold(true),
	}

	// Add custom format for keys.
	styles.Keys["err"] = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	styles.Values["err"] = lipgloss.NewStyle().Bold(true)

	styles.Keys["plugin"] = lipgloss.NewStyle().Foreground(DebugColor)
	styles.Values["plugin"] = lipgloss.NewStyle()

	return styles
}

// NewLogger initializes a new wrapped logger with default styles
func NewLogger(o io.Writer) hclog.Logger {
	c := charmlog.NewWithOptions(o, *defaultOptions())
	l := &CharmHclog{c}
	l.logger.SetStyles(defaultStyles())
	return l
}

// CharmHclog adapts the charm logger to the hashicorp logger.
type CharmHclog struct {
	logger *charmlog.Logger
}

// CharmHclog will implement the hclog.Logger.
var _ hclog.Logger = &CharmHclog{}

// Declaring hclogCharmLevels to map key: value pairs for adapting go-hclog to charmlog.
var hclogCharmLevels = map[hclog.Level]charmlog.Level{
	hclog.NoLevel: charmlog.InfoLevel,  // There is no "NoLevel" equivalent in charm, use info
	hclog.Trace:   charmlog.DebugLevel, // There is no "Trace" equivalent in charm, use debug
	hclog.Debug:   charmlog.DebugLevel,
	hclog.Info:    charmlog.InfoLevel,
	hclog.Warn:    charmlog.WarnLevel,
	hclog.Error:   charmlog.ErrorLevel,
	hclog.Off:     charmlog.FatalLevel, // There is no "Off" level equivalent in charm logger
}

// Declaring charmHclogLevels to map key: value pairs for adapting go-hclog to charmlog.
var charmHclogLevels = map[charmlog.Level]hclog.Level{
	charmlog.DebugLevel: hclog.Debug,
	charmlog.InfoLevel:  hclog.Info,
	charmlog.WarnLevel:  hclog.Warn,
	charmlog.ErrorLevel: hclog.Error,
	charmlog.FatalLevel: hclog.Error, // There is no "fatal" equivalent in go-hclog
}

func (c *CharmHclog) Log(level hclog.Level, msg string, args ...interface{}) {
	c.logger.Log(hclogCharmLevels[level], fmt.Sprintf(msg, args...))
}
func (c *CharmHclog) Trace(msg string, args ...interface{}) {
	c.logger.Debug(msg, args...)
}
func (c *CharmHclog) Debug(msg string, args ...interface{}) {
	c.logger.Debug(msg, args...)
}
func (c *CharmHclog) Info(msg string, args ...interface{}) {
	c.logger.Info(msg, args...)
}
func (c *CharmHclog) Warn(msg string, args ...interface{}) {
	c.logger.Warn(msg, args...)
}
func (c *CharmHclog) Error(msg string, args ...interface{}) {
	c.logger.Error(msg, args...)
}

// Methods of go-hclog interface.
func (c *CharmHclog) IsTrace() bool { return false }

func (c *CharmHclog) IsDebug() bool { return false }

func (c *CharmHclog) IsInfo() bool { return false }

func (c *CharmHclog) IsWarn() bool { return false }

func (c *CharmHclog) IsError() bool { return false }

func (c *CharmHclog) ImpliedArgs() []interface{} { return nil }

func (c *CharmHclog) With(args ...interface{}) hclog.Logger {
	return &CharmHclog{c.logger.With(args...)}
}

// The GetPrefix() method will return the prefix of the logger.
func (c *CharmHclog) Name() string { return c.logger.GetPrefix() }

// The Named() method appends to the current logger prefix.
func (c *CharmHclog) Named(name string) hclog.Logger {
	return &CharmHclog{c.logger.WithPrefix(name)}
}

// The ResetNamed() method creates the logger with only the prefix passed.
func (c *CharmHclog) ResetNamed(name string) hclog.Logger {
	return &CharmHclog{c.logger.WithPrefix(name)}
}

// The SetLevel() method enables setting logger level.
func (c *CharmHclog) SetLevel(level hclog.Level) {
	c.logger.SetLevel(hclogCharmLevels[level])
}

// The GetLevel() method returns the current level.
func (c *CharmHclog) GetLevel() hclog.Level {
	return charmHclogLevels[c.logger.GetLevel()]
}

// The StandardLog() of CharmHclog is returned, wrapping go-hclog.
func (c *CharmHclog) StandardLogger(opts *hclog.StandardLoggerOptions) *log.Logger {
	return (c.logger.StandardLog())
}

func (c *CharmHclog) StandardWriter(opts *hclog.StandardLoggerOptions) io.Writer { return os.Stdout }
