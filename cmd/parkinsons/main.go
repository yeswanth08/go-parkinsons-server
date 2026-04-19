package main

import (
	"log"

	api         "go-parkinsons-server/internal/api/gen"
	internalapi "go-parkinsons-server/internal/server"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"os"
)

func main() {
    grpcAddr := os.Getenv("GRPC_ADDR")
    if grpcAddr == "" {
        grpcAddr = "localhost:50051"
    }

    handler, err := internalapi.NewRPCHandler(grpcAddr)
    if err != nil {
        log.Fatalf("failed to connect to grpc: %v", err)
    }

    allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
    if allowedOrigin == "" {
        allowedOrigin = "http://localhost:5731"
    }

    e := echo.New()
    e.HideBanner = true
    e.Use(middleware.Logger())
    e.Use(middleware.Recover())
    e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
        AllowOrigins: []string{allowedOrigin, "https://v0-parkinsons-prod.vercel.app"},
        AllowMethods: []string{"GET", "POST", "OPTIONS"},
        AllowHeaders: []string{
            "Content-Type",
            "Upgrade",
            "Connection",
            "Sec-WebSocket-Key",
            "Sec-WebSocket-Version",
			"Sec-WebSocket-Extensions",
			"Sec-WebSocket-Protocol",
        },
    }))

    api.RegisterHandlers(e, handler)

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    log.Printf("server listening on :%s", port)
    log.Fatal(e.Start(":" + port))
}