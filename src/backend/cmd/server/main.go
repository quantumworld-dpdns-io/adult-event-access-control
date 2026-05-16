package main

import (
	"log"
	"net/http"
	"os"

	"github.com/quantumworld-dpdns-io/adult-event-access-control/backend/internal/auth"
	"github.com/quantumworld-dpdns-io/adult-event-access-control/backend/internal/db"
	"github.com/quantumworld-dpdns-io/adult-event-access-control/backend/internal/handlers"
	"github.com/quantumworld-dpdns-io/adult-event-access-control/backend/internal/middleware"
)

func main() {
	dsn := os.Getenv("DB_HOST")
	if dsn == "" {
		dsn = "localhost"
	}

	database, err := db.Connect(db.Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "aev"),
		Password: getEnv("DB_PASSWORD", "aev_secret"),
		DBName:   getEnv("DB_NAME", "aev_events"),
	})
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer database.Close()

	mux := http.NewServeMux()

	authHandler := auth.NewHandler(database)
	authHandler.RegisterRoutes(mux)

	eventHandler := handlers.NewEventHandler(database)
	eventHandler.RegisterRoutes(mux)

	ticketHandler := handlers.NewTicketHandler(database)
	ticketHandler.RegisterRoutes(mux)

	checkinHandler := handlers.NewCheckInHandler(database)
	checkinHandler.RegisterRoutes(mux)

	zkHandler := handlers.NewZKHandler(database)
	zkHandler.RegisterRoutes(mux)

	adminHandler := handlers.NewAdminHandler(database)
	adminHandler.RegisterRoutes(mux)

	wrapped := middleware.CORS(middleware.Logger(middleware.Auth(mux)))

	port := getEnv("PORT", "8080")
	log.Printf("aeV backend listening on :%s", port)
	if err := http.ListenAndServe(":"+port, wrapped); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
