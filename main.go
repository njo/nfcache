package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/njo/nfcache/pkg/apiclient"
	"github.com/njo/nfcache/pkg/apiserver"
	"github.com/njo/nfcache/pkg/datasource"
	"go.uber.org/zap"
)

func main() {
	// Load CLI Options
	var port int
	flag.IntVar(&port, "p", 8080, "Set the port number to listen on (Default 8080)")
	flag.Parse()

	// Set up logger
	zLogger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v\n", err)
	}
	defer zLogger.Sync()      // ensure logger buffer flushed on shutdown
	logger := zLogger.Sugar() // Sugar logger allows for printf style formatting

	// Load env vars
	err = godotenv.Load() // Adds .env file into regular os.env vars
	if err == nil {
		logger.Info("Loaded .env file")
	}
	githubToken := os.Getenv("GITHUB_API_TOKEN")
	if githubToken == "" {
		logger.Warn("Unable to load GITHUB_API_TOKEN")
	}

	// Init servers
	githubClient := apiclient.NewGithub(githubToken)
	apiCache := datasource.NewCachedAPI(githubClient, logger)
	server := apiserver.New(apiCache, logger)
	initalEndpoints := apiserver.CachedEndpoints()
	logger.Info("Pre-fetching initial endpoint data")
	for _, path := range initalEndpoints {
		logger.Infof("Fetching %s", path)
		err = apiCache.WatchEndpoint(path)
		if err != nil {
			// Choosing to early exit here as there's probably an external api issue
			logger.Fatalf("unable to fetch %s on startup: %v\n", path, err)
		}
	}
	apiCache.Run(time.Second * 60)            // Keeps the cache updated in the background
	listenAddress := ":" + strconv.Itoa(port) // Listen on all interfaces with specified port
	go server.Run(listenAddress)              // Run the service in a separate thread to not block signal handler

	// Wait for shutdown signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	server.Shutdown(5 * time.Second)
	apiCache.Shutdown() // Can take a bit if we're in the middle of a cache update
	logger.Info("Service gracefully exited")
}
