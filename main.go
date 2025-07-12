package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/debuggerboy/mcp-git-connector/handlers"
	"github.com/debuggerboy/mcp-git-connector/repository"
)

func main() {
	// 1. Load configuration
	if err := godotenv.Load(); err != nil {
		log.Printf("Note: No .env file found - using default settings")
	}

	// 2. Initialize components
	baseDir := getEnv("REPO_BASE_DIR", "/tmp/mcp-repos")
	port := getEnv("SERVER_PORT", "8080")
	ollamaURL := getEnv("OLLAMA_URL", "http://localhost:11434")

	log.Printf("Initializing MCP server (repo dir: %s, port: %s)", baseDir, port)
	repoManager := repository.NewGitManager(baseDir)
	mcpHandler := handlers.NewMCPHandler(repoManager, ollamaURL)

	// 3. Create HTTP server with timeouts
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      setupRoutes(mcpHandler),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 4. Start server in background
	serverReady := make(chan bool)
	go func() {
		log.Printf("Starting MCP server on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// 5. Quick health check to ensure server is up
	go func() {
		time.Sleep(500 * time.Millisecond)
		if _, err := http.Get("http://localhost:" + port + "/health"); err == nil {
			serverReady <- true
		} else {
			log.Printf("Server health check failed: %v", err)
			serverReady <- false
		}
	}()

	// 6. Signal readiness to Ollama
	if <-serverReady {
		signalReadyToOllama()
		log.Printf("MCP server ready and running in background")
	} else {
		log.Fatal("Failed to start MCP server")
	}

	// 7. Wait for shutdown signal
	waitForShutdown(server)
}

func setupRoutes(h *handlers.MCPHandler) *http.ServeMux {
	mux := http.NewServeMux()
	
	// Repository endpoints
	mux.Handle("/api/repo/clone", h.AuthMiddleware(h.CloneRepositoryHandler))
	mux.Handle("/api/repo/branch", h.AuthMiddleware(h.SwitchBranchHandler))
	mux.Handle("/api/repo/files", h.AuthMiddleware(h.ListFilesHandler))
	mux.Handle("/api/repo/file", h.AuthMiddleware(h.GetFileHandler))
	mux.Handle("/api/repo/update", h.AuthMiddleware(h.UpdateFileHandler))
	mux.Handle("/api/repo/commit", h.AuthMiddleware(h.CommitChangesHandler))
	mux.Handle("/api/repo/push", h.AuthMiddleware(h.PushChangesHandler))
	
	// LLM integration endpoints
	mux.Handle("/api/llm/review", h.AuthMiddleware(h.RequestCodeReviewHandler))
	
	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	return mux
}

func signalReadyToOllama() {
	// Implementation options:
	
	// Option 1: Write to stdout (Ollama can parse this)
	os.Stdout.Write([]byte("MCP_SERVER_READY\n"))
	
	// Option 2: Call Ollama's API
	// resp, err := http.Post("http://localhost:11434/api/ready", "text/plain", nil)
	// if err != nil {
	//     log.Printf("Failed to notify Ollama: %v", err)
	// }
}

func waitForShutdown(server *http.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	log.Println("Shutting down server...")
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	} else {
		log.Println("Server stopped gracefully")
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
