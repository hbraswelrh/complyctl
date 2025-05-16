// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/complytime/complytime/cmd/complytime/cli"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	complytime := cli.New()
	if err := complytime.ExecuteContext(ctx); err != nil {
		cli.Error(fmt.Sprintf("error running complytime: %v", err))
		os.Exit(1)
	}
}
