// SPDX-License-Identifier: Apache-2.0

package terminal

import (
	"fmt"
	"io"
	"os"
	"time"
)

// ShowSpinner synchronously shows a spinner in the terminal until a stop signal is received.
// The spinner is cleared before returning.
func ShowSpinner(stop chan int) {
	ShowSpinnerOut(os.Stdout, stop)
}

// ShowSpinnerOut synchronously shows a spinner in the terminal until a stop signal is received.
// The spinner is cleared before returning.
func ShowSpinnerOut(out io.Writer, stop chan int) {
	spinStates := []string{"|", "/", "-", "\\", "|", "/", "-", "\\"}
	i := 0
	for {
		select {
		case <-stop:
			_, _ = fmt.Fprintf(out, "\r\x1b[K")
			return
		default:
			_, _ = fmt.Fprintf(out, "\r\x1b[K%s", spinStates[i])
			i = (i + 1) % len(spinStates)
			time.Sleep(500 * time.Millisecond)
		}
	}
}
