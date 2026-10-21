package main

import (
	"context"
	"os"
	"os/signal"

	domaincheck "github.com/sd800/domain-check"
	"github.com/sd800/domain-check/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	os.Exit(cli.Run(ctx, os.Args[1:], os.Stdout, os.Stderr, domaincheck.Version))
}
