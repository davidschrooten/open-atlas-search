package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/davidschrooten/open-atlas-search/config"
	"github.com/davidschrooten/open-atlas-search/internal/api"
	"github.com/davidschrooten/open-atlas-search/internal/cluster"
	"github.com/davidschrooten/open-atlas-search/internal/indexer"
	"github.com/davidschrooten/open-atlas-search/internal/mongodb"
	"github.com/davidschrooten/open-atlas-search/internal/search"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the Open Atlas Search server",
	Long: `Start the HTTP server that provides MongoDB Atlas Search compatible API endpoints.
The server will automatically create and maintain search indexes based on the configuration.`,
	RunE: runServer,
}

func init() {
	rootCmd.AddCommand(serverCmd)

	// Server-specific flags
	serverCmd.Flags().String("host", "0.0.0.0", "Host to bind the server to")
	serverCmd.Flags().Int("port", 8080, "Port to bind the server to")

	// Bind flags to viper
	viper.BindPFlag("server.host", serverCmd.Flags().Lookup("host"))
	viper.BindPFlag("server.port", serverCmd.Flags().Lookup("port"))
}

func runServer(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize MongoDB client
	mongoClient, err := mongodb.NewClient(cfg.MongoDB)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer mongoClient.Disconnect()

	// Initialize search engine
	searchEngine, err := search.NewEngine(cfg.Search)
	if err != nil {
		return fmt.Errorf("failed to initialize search engine: %w", err)
	}
	defer searchEngine.Close()

	// Initialize indexer
	indexerService, err := indexer.NewService(mongoClient, searchEngine, cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize indexer: %w", err)
	}

	// Initialize cluster manager if cluster mode is enabled
	var clusterManager *cluster.Manager
	if cfg.Cluster.Enabled {
		clusterManager, err = cluster.NewManager(cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize cluster manager: %w", err)
		}

		if err := clusterManager.Start(); err != nil {
			return fmt.Errorf("failed to start cluster manager: %w", err)
		}
		defer clusterManager.Stop()
	}

	// Start indexing process
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := indexerService.Start(ctx); err != nil {
		return fmt.Errorf("failed to start indexer: %w", err)
	}

	// Initialize API server
	apiServer := api.NewServer(searchEngine, indexerService, cfg, clusterManager)

	// Setup HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      apiServer.Router(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on %s:%d", cfg.Server.Host, cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Cancel context to stop indexer
	cancel()

	// Shutdown server with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
		return err
	}

	log.Println("Server exited")
	return nil
}
