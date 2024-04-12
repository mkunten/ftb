package main

import (
	"log"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var cfg *Config

func main() {
	// config
	c, err := NewConfig()
	if err != nil {
		log.Fatal("config: ", err)
	}
	cfg = c

	// elasticsearch
	var es = &ES{}
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
	api.GET("/ocrraw/:id", GetOCRRaw(es))
	api.GET("/countRecord", GetCount(es))
	api.GET("/search", GetNgramSearch(es))
	api.POST("/register", PostRegister(es))
	api.POST("/bulkRegister", PostBulkRegister(es))

	e.Logger.Fatal(e.Start(":1323"))
}
