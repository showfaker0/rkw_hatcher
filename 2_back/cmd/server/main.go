package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"rkw_hatcher/internal/config"
	"rkw_hatcher/internal/db"
	"rkw_hatcher/internal/handlers"
)

func main() {
	cfg := config.Load()
	conn, err := db.Open(cfg.DSN())
	if err != nil {
		log.Fatalf("mysql connect failed: %v", err)
	}
	defer conn.Close()

	r := gin.Default()
	api := &handlers.API{DB: conn}
	api.Register(r)

	log.Printf("listening on %s", cfg.HTTPAddr)
	if err := r.Run(cfg.HTTPAddr); err != nil {
		log.Fatal(err)
	}
}
