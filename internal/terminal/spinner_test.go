// SPDX-License-Identifier: Apache-2.0

package terminal

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestSpin(t *testing.T) {
	testData := []struct {
		Sleep  time.Duration
		Output string
	}{
		{
			Sleep:  1050 * time.Millisecond,
			Output: "\r\x1b[K|\r\x1b[K/\r\x1b[K-\r\x1b[K",
		},
		{
			Sleep:  1550 * time.Millisecond,
			Output: "\r\x1b[K|\r\x1b[K/\r\x1b[K-\r\x1b[K\\\r\x1b[K",
		},
	}

	ansiEscape := func(str string) string {
		escaped := str
		escaped = strings.ReplaceAll(escaped, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "\x1b", "\\x1b")
		escaped = strings.ReplaceAll(escaped, "\r", "\\r")
		return escaped
	}

	for _, test := range testData {
		out := bytes.NewBuffer(nil)
		stop := make(chan int)
		// Kick off fake long-running task
		go time.AfterFunc(test.Sleep, func() { stop <- 1 })
		// Show the spinner while we wait
		ShowSpinnerOut(out, stop)
		if actual := out.String(); actual != test.Output {
			escapedActual := ansiEscape(actual)
			escapedOutput := ansiEscape(test.Output)
			t.Errorf("\nExpected: %s\nActual: %s", escapedOutput, escapedActual)
		}
	}
}
