package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"

	"github.com/yourname/tabikake/config"
	"github.com/yourname/tabikake/internal/claude"
	"github.com/yourname/tabikake/internal/handler"
	appmiddleware "github.com/yourname/tabikake/internal/middleware"
	"github.com/yourname/tabikake/internal/notion"
	"github.com/yourname/tabikake/internal/service"
	"github.com/yourname/tabikake/internal/store"
)

func main() {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("warning: could not load .env file: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	// SQLite
	db, err := store.New(cfg.SQLitePath)
	if err != nil {
		log.Fatalf("sqlite error: %v", err)
	}
	defer db.Close()

	// Clients
	notionClient := notion.New(cfg.NotionIntegrationToken, cfg.NotionRootPageID)
	claudeClient := claude.New(cfg.AnthropicAPIKey)

	// Services
	authSvc := service.NewAuthService(db, cfg.NotionOAuthClientID, cfg.NotionOAuthClientSecret, cfg.NotionOAuthRedirectURI, cfg.JWTSecret, cfg.TokenEncryptKey)
	tripSvc := service.NewTripService(db, notionClient)
	memberSvc := service.NewMemberService(db)
	recordSvc := service.NewRecordService(db, notionClient, claudeClient)
	dashboardSvc := service.NewDashboardService(db, notionClient)
	settlementSvc := service.NewSettlementService(db, notionClient)

	// Handlers
	authHandler := handler.NewAuthHandler(authSvc, cfg.FrontendURL)
	tripHandler := handler.NewTripHandler(tripSvc, memberSvc)
	memberHandler := handler.NewMemberHandler(memberSvc, tripSvc)
	recordHandler := handler.NewRecordHandler(recordSvc)
	dashboardHandler := handler.NewDashboardHandler(dashboardSvc)
	settlementHandler := handler.NewSettlementHandler(settlementSvc)

	// Echo setup
	e := echo.New()
	e.HideBanner = true

	e.Use(appmiddleware.Logger())
	e.Use(appmiddleware.Recover())
	e.Use(appmiddleware.CORS(cfg.FrontendURL))

	e.HTTPErrorHandler = func(err error, c echo.Context) {
		var he *echo.HTTPError
		if errors.As(err, &he) {
			_ = c.JSON(he.Code, map[string]interface{}{"error": he.Message})
			return
		}
		_ = c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "internal server error"})
	}

	auth := appmiddleware.JWTAuth(authSvc)

	// Public routes
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	e.GET("/auth/notion/url", authHandler.OAuthURL)
	e.GET("/auth/notion/callback", authHandler.NotionCallback)
	e.POST("/auth/notion/callback", authHandler.NotionCallbackPost)
	e.GET("/trips/join-info", tripHandler.GetJoinInfo)

	// Protected routes
	e.GET("/auth/me", authHandler.Me, auth)
	e.POST("/auth/logout", authHandler.Logout, auth)

	e.GET("/trips", tripHandler.ListTrips, auth)
	e.POST("/trips", tripHandler.CreateTrip, auth)
	e.GET("/trips/:id", tripHandler.GetTrip, auth)
	e.PATCH("/trips/:id", tripHandler.UpdateTrip, auth)
	e.DELETE("/trips/:id", tripHandler.DeleteTrip, auth)

	e.POST("/trips/join", memberHandler.JoinTrip, auth)
	e.GET("/trips/:id/members", memberHandler.ListMembers, auth)
	e.DELETE("/trips/:id/members/:user_id", memberHandler.DeleteMember, auth)

	e.GET("/trips/:id/settlement", settlementHandler.Calculate, auth)
	e.POST("/trips/:id/settlement/export", settlementHandler.Export, auth)

	e.GET("/records", recordHandler.ListRecords, auth)
	e.POST("/records", recordHandler.CreateRecord, auth)
	e.PATCH("/records/:id", recordHandler.UpdateRecord, auth)
	e.DELETE("/records/:id", recordHandler.DeleteRecord, auth)

	e.POST("/parse", recordHandler.ParseReceipt, auth)

	e.GET("/dashboard/:trip_id", dashboardHandler.GetDashboard, auth)

	// Graceful shutdown
	addr := fmt.Sprintf(":%s", cfg.Port)
	go func() {
		log.Printf("starting server on %s", addr)
		if err := e.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}
}
