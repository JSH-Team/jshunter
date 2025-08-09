package db

import (
	"github.com/jsh-team/jshunter/internal/config"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterRoutes(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {

		se.Router.BindFunc(func(e *core.RequestEvent) error {
			if e.RealIP() != "127.0.0.1" {
				return e.UnauthorizedError("Unauthorized", nil)
			}
			return e.Next()
		})
		se.Router.GET("/api/config", func(c *core.RequestEvent) error {
			data := map[string]interface{}{
				"target":      config.Target,
				"storage_dir": config.StorageDir,
			}
			return c.JSON(200, data)
		})

		return se.Next()
	})
}
