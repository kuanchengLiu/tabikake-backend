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
	appdb "github.com/yourname/tabikake/internal/db"
	"github.com/yourname/tabikake/internal/handler"
	appmiddleware "github.com/yourname/tabikake/internal/middleware"
	"github.com/yourname/tabikake/internal/notion"
	"github.com/yourname/tabikake/internal/service"
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
	database, err := appdb.New(cfg.SQLitePath)
	if err != nil {
		log.Fatalf("sqlite error: %v", err)
	}
	defer database.Close()

	// Clients
	notionClient := notion.New(cfg.NotionIntegrationToken, cfg.NotionRootPageID)
	claudeClient := claude.New(cfg.AnthropicAPIKey)

	// Services
	authSvc := service.NewAuthService(cfg.NotionOAuthClientID, cfg.NotionOAuthClientSecret, cfg.NotionOAuthRedirectURI, cfg.JWTSecret)
	parseSvc := service.NewParseService(claudeClient)
	tripSvc := service.NewTripService(database, notionClient)
	recordSvc := service.NewRecordService(database, notionClient)
	dashboardSvc := service.NewDashboardService(database, notionClient)
	splitSvc := service.NewSplitService(database, notionClient, dashboardSvc)
	memberSvc := service.NewMemberService(database)
	settlementSvc := service.NewSettlementService(database, notionClient)

	// Handlers
	authHandler := handler.NewAuthHandler(authSvc)
	parseHandler := handler.NewParseHandler(parseSvc)
	tripHandler := handler.NewTripHandler(tripSvc)
	recordHandler := handler.NewRecordHandler(recordSvc)
	dashboardHandler := handler.NewDashboardHandler(dashboardSvc)
	splitHandler := handler.NewSplitHandler(splitSvc)
	memberHandler := handler.NewMemberHandler(memberSvc, settlementSvc)

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

	// Middleware factories
	auth := appmiddleware.JWTAuth(cfg.JWTSecret)
	memberOf := func(paramName string) echo.MiddlewareFunc {
		return appmiddleware.ValidateTripMember(database, paramName)
	}

	// Public
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	e.POST("/auth/notion/callback", authHandler.NotionCallback)
	e.GET("/trips/join-info", tripHandler.GetJoinInfo) // public: show trip info before joining

	// Protected — JWT only
	e.GET("/auth/me", authHandler.Me, auth)

	e.GET("/trips", tripHandler.ListTrips, auth)
	e.POST("/trips", tripHandler.CreateTrip, auth)
	e.GET("/trips/:id", tripHandler.GetTrip, auth)         // is_member populated via X-Member-ID header
	e.POST("/trips/join", memberHandler.JoinTrip, auth)    // join via invite code → creates member

	e.POST("/parse", parseHandler.ParseReceipt, auth)

	e.PATCH("/records/:id", recordHandler.UpdateRecord, auth)
	e.DELETE("/records/:id", recordHandler.DeleteRecord, auth)

	// Protected — JWT + must be a trip member (X-Member-ID)
	e.GET("/trips/:id/members", memberHandler.ListMembers, auth, memberOf("id"))
	e.POST("/trips/:id/members", memberHandler.AddMember, auth, memberOf("id"))
	e.DELETE("/trips/:id/members/:member_id", memberHandler.DeleteMember, auth) // owner check inside handler
	e.GET("/trips/:id/settlement", memberHandler.GetSettlement, auth, memberOf("id"))

	e.GET("/records", recordHandler.ListRecords, auth, memberOf(""))        // trip_id from query param
	e.POST("/records", recordHandler.CreateRecord, auth)                    // member validated in service

	e.GET("/dashboard/:trip_id", dashboardHandler.GetDashboard, auth, memberOf("trip_id"))

	e.POST("/split/export/:trip_id", splitHandler.ExportSettlement, auth, memberOf("trip_id"))

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
