package main

import (
	"log"
	"net/http"
	"os"

	"github.com/debuggerboy/mcp-git-connector/handlers"
	"github.com/debuggerboy/mcp-git-connector/repository"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Couldn't load .env file: %v", err)
	}

	// Get configuration from environment with defaults
	baseDir := getEnv("REPO_BASE_DIR", "/tmp/mcp-repos")
	port := getEnv("SERVER_PORT", "8080")

	// Initialize the repository manager
	repoManager := repository.NewGitManager(baseDir)
	//repoManager := repository.NewGitManager("/tmp/mcp-repos")

	// Initialize HTTP handlers
	mcpHandler := handlers.NewMCPHandler(repoManager)

	// Set up routes
	http.Handle("/api/repo/clone", mcpHandler.AuthMiddleware(mcpHandler.CloneRepositoryHandler))
	http.Handle("/api/repo/branch", mcpHandler.AuthMiddleware(mcpHandler.SwitchBranchHandler))
	http.Handle("/api/repo/files", mcpHandler.AuthMiddleware(mcpHandler.ListFilesHandler))
	http.Handle("/api/repo/file", mcpHandler.AuthMiddleware(mcpHandler.GetFileHandler))
	http.Handle("/api/repo/update", mcpHandler.AuthMiddleware(mcpHandler.UpdateFileHandler))
	http.Handle("/api/repo/commit", mcpHandler.AuthMiddleware(mcpHandler.CommitChangesHandler))
	http.Handle("/api/repo/push", mcpHandler.AuthMiddleware(mcpHandler.PushChangesHandler))
	http.Handle("/api/llm/review", mcpHandler.AuthMiddleware(mcpHandler.RequestCodeReviewHandler))

	// Start server
	log.Printf("Starting MCP server on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
