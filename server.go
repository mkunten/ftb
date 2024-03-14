package main

import (
	"log"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var cfg *Config
var es = &ES{}

func main() {
	// config
	c, err := NewConfig()
	if err != nil {
		log.Fatal("config: ", err)
	}
	cfg = c

	// elasticsearch
	if err := es.Init(); err != nil {
		log.Fatal("es.Init: ", err)
	}
	if err := es.InitIndex(cfg.ResetES); err != nil {
		log.Fatal("es.InitIndex: ", err)
	}

	// echo
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	api := e.Group("/api")
	api.GET("/countRecord", es.GetCount)
	api.GET("/search", es.GetNgramSearch)
	api.POST("/register", es.PostRegister)
	api.POST("/bulkRegister", es.PostBulkRegister)

	e.Logger.Fatal(e.Start(":1323"))
}
