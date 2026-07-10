// Command uptime-server is the single DeltaUptime Control Plane binary.
// It dispatches to subcommands: api, scheduler, worker, migrate, version.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"deltauptime/apps/control-plane/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "uptime-server: "+err.Error())
		os.Exit(1)
	}
}
