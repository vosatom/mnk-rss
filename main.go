package main

import (
	"errors"
	"fmt"
	"mnk-rss/lib"
	"net/http"

	"github.com/ardanlabs/conf/v3"
	"github.com/coocood/freecache"
	cache "github.com/gitsight/go-echo-cache"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

func main() {
	e := echo.New()
	const prefix = "APP"

	cfg := struct {
		Address    string  `conf:"default:0.0.0.0:8010"`
		ConfigPath string  `conf:"default:/mnk-rss/config.yaml"`
		LogLevel   log.Lvl `conf:"default:0"`
	}{}

	help, err := conf.Parse(prefix, &cfg)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			fmt.Println(help)
		}
		e.Logger.Error(nil)
		return
	}

	c := freecache.NewCache(7 * 1024 * 1024)
	e.Use(cache.New(&cache.Config{}, c))
	e.Logger.SetLevel(cfg.LogLevel)

	e.GET("/*", func(c echo.Context) error {
		config, err := lib.ReadConfig(cfg.ConfigPath)
		if err != nil {
			return err
		}

		feedConfig, ok := config.Paths[c.Request().URL.Path]
		if !ok {
			return c.NoContent(http.StatusNotFound)
		}

		var result string
		switch feedConfig.Type {
		case "comments":
			result, err = lib.HandleComments(c.Request(), config, feedConfig)
		default:
			result, err = lib.HandleWFS(c.Request(), config, feedConfig)
		}

		if err != nil {
			return err
		}

		return c.String(http.StatusOK, result)
	})

	e.Logger.Fatal(e.Start(cfg.Address))
}
