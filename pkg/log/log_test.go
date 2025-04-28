// SPDX-License-Identifier: Apache-2.0

package log

import (
	"bytes"
	"fmt"
	"testing"

	charmlogger "github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"
)

// Declaring the test logger to be a new charm logger.
func TestLogger(t *testing.T) {
	var buf bytes.Buffer
	lc := charmlogger.New(&buf) // declaring buffer as the charm logger stored in lc
	cases := []struct {
		prefix   string
		expected string
		msg      string
	}{
		{
			prefix:   "",
			expected: "INFO info\n",
			msg:      "info",
		},
	}
	for _, tc := range cases {
		t.Run(tc.prefix, func(t *testing.T) {
			lc.With(tc.prefix).Info(tc.msg)
			charmlogger.Print("")
			assert.Equal(t, tc.expected, buf.String()) // Check if the expected string level == test val
		})
	}
}

// Setting level using the buffer, initializing level as Info.
func TestNonExistentLevel(t *testing.T) {
	var buf bytes.Buffer
	l := charmlogger.New(&buf)
	l.SetLevel(charmlogger.InfoLevel)

	cases := []struct {
		prefix   string
		expected string
		level    charmlogger.Level
	}{
		{
			prefix:   " ",
			expected: "INFO: info\n",
			level:    charmlogger.InfoLevel,
		},
		{
			prefix:   "incorrect",
			expected: "INFO info\n",
			level:    charmlogger.InfoLevel,
		},
		{
			prefix:   "fake level",
			expected: "INFO info\n",
			level:    charmlogger.InfoLevel,
		},
		{
			prefix:   " ",
			expected: "INFO: info\n",
			level:    charmlogger.InfoLevel,
		},
	}
	for _, c := range cases {
		buf.Reset()
		charmlogger.Printf("The named prefix: %s is of level: %s.", c.prefix, c.level)
	}
}

// Testing different input types and prefixes.
func TestTypes(t *testing.T) {
	tests := []struct {
		prefix   string
		expected string
		level    string
		msg      string
	}{
		{
			// Testing Debug level prefix
			prefix:   "Debug",
			expected: "Debug",
			level:    "Debug",
			msg:      "The ComplyTime command has been executed.",
		},
		{
			// Testing Info level prefix
			prefix:   "Info",
			expected: "Info",
			level:    "Info",
			msg:      "The ComplyTime command has been executed.",
		},
		{
			// Testing Warn level prefix
			prefix:   "Warn",
			expected: "Warn",
			level:    "Warn",
			msg:      "The ComplyTime command has been executed.",
		},
		{
			// Testing Error level prefix
			prefix:   "Error",
			expected: "Error",
			level:    "Error",
			msg:      "The ComplyTime command has been executed.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.prefix, func(t *testing.T) {
			var buf bytes.Buffer
			charmlogger.New(&buf)
			charmlogger.Info(fmt.Sprintf("The ComplyTime command at level %s was executed successfully.", tt.level))
		})
	}
}

// Testing various prefixes and levels entered.
func TestPrefixMatch(t *testing.T) {
	tests := []struct {
		prefix   string
		expected string
		level    string
		msg      string
	}{
		{
			// Testing Debug level prefix
			prefix:   "Debug",
			expected: "Debug",
			level:    "Debug",
			msg:      "The ComplyTime command has been executed.",
		},
		{
			// Testing Info level prefix
			prefix:   "Info",
			expected: "Info",
			level:    "Info",
			msg:      "The ComplyTime command has been executed.",
		},
		{
			// Testing Warn level prefix
			prefix:   "Warn",
			expected: "Warn",
			level:    "Warn",
			msg:      "The ComplyTime command has been executed.",
		},
		{
			// Testing Error level prefix
			prefix:   "Error",
			expected: "Error",
			level:    "Error",
			msg:      "The ComplyTime command has been executed.",
		},
	}
	for _, tdata := range tests {
		t.Run(tdata.prefix, func(t *testing.T) {
			var buf bytes.Buffer
			charmlogger.New(&buf)
			charmlogger.Info(fmt.Sprintf("The ComplyTime command at level %s was executed successfully. %s", tdata.level, tdata.msg))
		})
	}
}
