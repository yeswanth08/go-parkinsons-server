package main

import (
	"log"

	api "go-parkinsons-server/internal/api/gen"
	internalapi "go-parkinsons-server/internal/server"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	handler, err := internalapi.NewRPCHandler("localhost:50051")
	if err != nil {
		log.Fatalf("failed to connect to grpc: %v", err)
	}

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
        AllowOrigins: []string{"http://localhost:3000"},
        AllowMethods: []string{"POST"},
        AllowHeaders: []string{"Content-Type"},
    }))

	api.RegisterHandlers(e, handler)

	log.Fatal(e.Start(":8080"))
}