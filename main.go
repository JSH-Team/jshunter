package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/JSH-Team/JSHunter/cmd"

	_ "github.com/JSH-Team/JSHunter/internal/db"

	m "github.com/pocketbase/pocketbase/migrations"

	"github.com/pocketbase/pocketbase/tools/security"

	"github.com/pocketbase/pocketbase/core"
)

// Version information set during build
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	// Set version information in cmd package
	cmd.SetVersion(Version, BuildTime, GitCommit)

	// Set up signal handling for immediate shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signalChan
		os.Exit(0)
	}()

	m.Register(func(app core.App) error {
		superusers, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
		if err != nil {
			return err
		}

		record := core.NewRecord(superusers)

		password := security.RandomString(64)
		record.Set("email", "sa@jshunter.local")
		record.Set("password", password)
		//logger.Info("Superuser created with password: %s", password)
		if err := app.Save(record); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error { // optional revert operation
		record, _ := app.FindAuthRecordByEmail(core.CollectionNameSuperusers, "sa@jshunter.local")
		if record == nil {
			return nil
		}

		return app.Delete(record)
	})

	// Execute command
	cmd.Execute()
}
