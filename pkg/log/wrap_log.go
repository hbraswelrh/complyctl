package log

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	charmlog "github.com/charmbracelet/log"
	"github.com/hashicorp/go-hclog"
)

var ErrMissingValue = fmt.Errorf("missing value")

// Need to initialize function
// Possibly use the LoggerOptions here to configure new logger
// Must indicate the outputs, etc
// HClog Logger Options will interact with the formatter
func init() {
	charmlog.Default()
}

var (
	DebugColor = lipgloss.AdaptiveColor{Light: "63", Dark: "63"}
	InfoColor  = lipgloss.AdaptiveColor{Light: "74", Dark: "86"}
	WarnColor  = lipgloss.AdaptiveColor{Light: "214", Dark: "192"}
	ErrorColor = lipgloss.AdaptiveColor{Light: "203", Dark: "203"}
	FatalColor = lipgloss.AdaptiveColor{Light: "134", Dark: "134"}
)

func defaultOptions() *log.Options {
	return &log.Options{
		ReportCaller:    false,
		ReportTimestamp: false,
	}
}

// setup styles to use for logger
func defaultStyles() *log.Styles {
	styles := log.DefaultStyles()

	styles.Levels = map[log.Level]lipgloss.Style{
		log.DebugLevel: lipgloss.NewStyle().
			SetString(strings.ToUpper(log.DebugLevel.String())).
			Foreground(DebugColor).
			Faint(true),
		log.InfoLevel: lipgloss.NewStyle().
			SetString(strings.ToUpper(log.InfoLevel.String())).
			Foreground(InfoColor),
		log.WarnLevel: lipgloss.NewStyle().
			SetString(strings.ToUpper(log.WarnLevel.String())).
			Foreground(WarnColor),
		log.ErrorLevel: lipgloss.NewStyle().
			SetString(strings.ToUpper(log.ErrorLevel.String())).
			Foreground(ErrorColor),
		log.FatalLevel: lipgloss.NewStyle().
			SetString(strings.ToUpper(log.FatalLevel.String())).
			Foreground(FatalColor).
			Bold(true),
	}

	// add custom format for keys
	styles.Keys["err"] = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	styles.Values["err"] = lipgloss.NewStyle().Bold(true)

	styles.Keys["plugin"] = lipgloss.NewStyle().Foreground(DebugColor)
	styles.Values["plugin"] = lipgloss.NewStyle()

	return styles
}

// Wrap the functionality of charmlogger in hclogger
func WrapLog(charmlog *charmlog.Logger) hclog.Logger { return &CharmHclog{charmlog} }

// CharmHclog will be a structure that accesses the attributes of charm logger
type CharmHclog struct {
	logger *charmlog.Logger
}

// CharmHclog will implement the hclog.Logger

var _ hclog.Logger = &CharmHclog{}

// Declaring hclogCharmLevels as a map with key values for adapting hclog to charmlog
// key: value -> hclog.LEVEL: charmlog.LEVEL
var hclogCharmLevels = map[hclog.Level]charmlog.Level{
	hclog.NoLevel: charmlog.InfoLevel,  // There is no "NoLevel" equivalent in charm, use info
	hclog.Trace:   charmlog.DebugLevel, // There is no "Trace" equivalent in charm, use debug
	hclog.Debug:   charmlog.DebugLevel,
	hclog.Info:    charmlog.InfoLevel,
	hclog.Warn:    charmlog.WarnLevel,
	hclog.Error:   charmlog.ErrorLevel,
	hclog.Off:     charmlog.FatalLevel, // There is no "Off" level equivalent in charm
}

// Declaring charmHclogLevels to map the key: value pairs as charmlogger and hclog
var charmHclogLevels = map[charmlog.Level]hclog.Level{
	charmlog.DebugLevel: hclog.Debug,
	charmlog.InfoLevel:  hclog.Info,
	charmlog.WarnLevel:  hclog.Warn,
	charmlog.ErrorLevel: hclog.Error,
	charmlog.FatalLevel: hclog.Error, // There is no "fatal" equivalent in hclog
}

// c will have information from the CharmHclog structure and will access the charm logger
// Log will use the level of hclog and do the identical logger operation using the charmlogger
// The map defines the level matches

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

// Functions from go-hc-log
func (c *CharmHclog) IsTrace() bool              { return false }
func (c *CharmHclog) IsDebug() bool              { return false }
func (c *CharmHclog) IsInfo() bool               { return false }
func (c *CharmHclog) IsWarn() bool               { return false }
func (c *CharmHclog) IsError() bool              { return false }
func (c *CharmHclog) ImpliedArgs() []interface{} { return nil }
func (c *CharmHclog) With(args ...interface{}) hclog.Logger {
	return &CharmHclog{c.logger.With(args...)}
}

// Need to configure a Name function
func (c *CharmHclog) Name() string { return c.Name() }

//func (c *CharmHclog) Name() string { return hclog.Logger(c.logger).Name() }

// Take input and then prepend name string
func (c *CharmHclog) Named(name string) hclog.Logger {
	return &CharmHclog{c.logger.With()}
}

// go-hclog logger resetnamed function to implement
func (c *CharmHclog) ResetNamed(name string) hclog.Logger {
	c.Named(name)
	logger, err := charmlog.SetOutput()
	if err != nil {
		panic(err)
	}
	return &CharmHclog{logger: charmlog.New()}
}

// Enables setting log level
func (c *CharmHclog) SetLevel(level hclog.Level) {
	charmlog.SetLevel(hclogCharmLevels[level])
}

// GetLevel using charm logger GetLevel
func (c *CharmHclog) GetLevel() hclog.Level {
	return charmHclogLevels[hclog.Level(charmlog.GetLevel())]
}

// GetLevel using charm logger Level method to extract
//func (c *CharmHclog) GetLevel() hclog.Level { return charmHclogLevels[hclog.Level(charmlog.Level(level))]}

// Look at stdlog.go for forcing level
// The standard logger needs to be implemented to return a standard logger of the charm type
// TODO: Need to point the opts to the hclog LoggerOptions
// The standard logger options configure the new logger
func (c *CharmHclog) StandardLogger(opts *hclog.StandardLoggerOptions) *charmlog.Logger {
	return charmlog.NewWithOptions(c.logger.StandardLog())
}

func (c *CharmHclog) StandardWriter(opts *hclog.StandardLoggerOptions) io.Writer { return os.Stdout }

func Logger() *charmlog.Logger {
	logger := log.NewWithOptions(os.Stderr, *defaultOptions())

	styles := defaultStyles()
	logger.SetStyles(styles)
	return logger

}
