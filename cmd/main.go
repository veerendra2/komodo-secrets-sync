package main

import (
	"fmt"
	"log/slog"

	"github.com/alecthomas/kong"
	"github.com/veerendra2/gopackages/slogger"
	"github.com/veerendra2/gopackages/version"
	"github.com/veerendra2/komodo-secrets-injector/pkg/komodo"
	"github.com/veerendra2/komodo-secrets-injector/pkg/secretsmanager"
)

const appName = "komodo-secrets-injector"

var cli struct {
	Komodo komodo.Config `embed:"" prefix:"komodo-" envprefix:"KOMODO_"`

	Bitwarden secretsmanager.BitwardenConfig `cmd:"" help:"Bitwarden Secrets Manager." group:"Secret Managers" default:""`

	Log     slogger.Config   `embed:"" prefix:"log-" envprefix:"LOG_"`
	Version kong.VersionFlag `name:"version" help:"Print version information and exit"`
}

func main() {
	kongCtx := kong.Parse(&cli,
		kong.Name(appName),
		kong.Description("A tool to inject secrets into Komodo from a secrets manager."),
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

	// rootCtx := context.Background()

	var secretMgrClient secretsmanager.Client
	var err error

	switch cmd := kongCtx.Command(); cmd {
	case "bitwarden":
		secretMgrClient, err = secretsmanager.NewBwClient(cli.Bitwarden)
		if err != nil {
			slog.Error("Failed to create Bitwarden client", "error", err)
			kongCtx.Exit(1)
		}
	}

	value, _ := secretMgrClient.Get("")
	fmt.Println(value)
	// ctx, cancel := context.WithTimeout(rootCtx, 5*time.Second)
	// defer cancel()

	// komodoClient, err := komodo.NewClient(cli.Komodo)
	// if err != nil {
	// 	slog.Error("Failed to create Komodo client", "error", err)
	// 	kongCtx.Exit(1)
	// }

	// err = komodoClient.UpsertVariable(ctx, "TEST", "test", "this is a test", true)
	// if err != nil {
	// 	slog.Error("Failed to create variable", "error", err)
	// 	kongCtx.Exit(1)
	// }

}
