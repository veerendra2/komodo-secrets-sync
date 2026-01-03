package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/veerendra2/gopackages/slogger"
	"github.com/veerendra2/gopackages/version"
	"github.com/veerendra2/komodo-secrets-sync/internal/reconciler"
	"github.com/veerendra2/komodo-secrets-sync/pkg/komodo"
	"github.com/veerendra2/komodo-secrets-sync/pkg/secrets"
)

const appName = "komodo-secrets-sync"

var cli struct {
	Komodo     komodo.Config     `embed:"" prefix:"komodo-" envprefix:"KOMODO_"`
	Reconciler reconciler.Config `embed:"" prefix:"reconciler-" envprefix:"RECONCILER_"`

	Bitwarden secrets.BitwardenConfig `cmd:"" help:"Bitwarden Secrets Manager." group:"Secret Managers" default:""`

	Log     slogger.Config   `embed:"" prefix:"log-" envprefix:"LOG_"`
	Version kong.VersionFlag `name:"version" help:"Print version information and exit"`
}

func main() {
	kongCtx := kong.Parse(&cli,
		kong.Name(appName),
		kong.Description("Sync secrets from a secrets manager into Komodo."),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
		kong.Vars{
			"version": version.Version,
		},
	)

	kongCtx.FatalIfErrorf(kongCtx.Error)

	slog.SetDefault(slogger.New(cli.Log))

	slog.Info("Version information", version.Info()...)
	slog.Info("Build context", version.BuildContext()...)

	// Create context that listens for shutdown signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var smClient secrets.Client
	var err error

	switch cmd := kongCtx.Command(); cmd {
	case "bitwarden":
		smClient, err = secrets.NewBitwarden(cli.Bitwarden)
		if err != nil {
			slog.Error("Failed to create Bitwarden client", "error", err)
			kongCtx.Exit(1)
		}
	default:
		slog.Error("No secrets manager specified")
		kongCtx.Exit(1)
	}

	kClient, err := komodo.NewClient(cli.Komodo)
	if err != nil {
		slog.Error("Failed to create Komodo client", "error", err)
		kongCtx.Exit(1)
	}

	r := reconciler.New(cli.Reconciler, smClient, kClient)
	if err := r.Run(ctx); err != nil {
		slog.Error("Reconciliation failed", "error", err)
		kongCtx.Exit(1)
	}
}
