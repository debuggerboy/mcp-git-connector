package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/debuggerboy/mcp-git-connector/handlers"
	"github.com/debuggerboy/mcp-git-connector/repository"
)

func main() {
	// Initialize the repository manager
	repoManager := repository.NewGitManager("/tmp/mcp-repos")

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
	port := "8080"
	log.Printf("Starting MCP server on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
