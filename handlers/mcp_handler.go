package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/debuggerboy/mcp-git-connector/repository"
)

type MCPHandler struct {
	repoManager *repository.GitManager
	ollamaURL   string
}

func NewMCPHandler(rm *repository.GitManager, ollamaURL string) *MCPHandler {
	return &MCPHandler{
		repoManager: rm,
		ollamaURL:   ollamaURL,
	}
}

func (h *MCPHandler) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appPassword := r.Header.Get("X-Bitbucket-Token")
		if appPassword == "" {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), "appPassword", appPassword)
		next(w, r.WithContext(ctx))
	}
}

type CloneRequest struct {
	RepoURL  string `json:"repo_url"`
	RepoName string `json:"repo_name"`
}

func (h *MCPHandler) CloneRepositoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CloneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	appPassword := r.Context().Value("appPassword").(string)
	repoPath, err := h.repoManager.CloneRepository(req.RepoURL, appPassword, req.RepoName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to clone repository: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status":    "success",
		"repo_path": repoPath,
	})
}

type BranchRequest struct {
	RepoPath   string `json:"repo_path"`
	BranchName string `json:"branch_name"`
}

func (h *MCPHandler) SwitchBranchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req BranchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.repoManager.SwitchBranch(req.RepoPath, req.BranchName); err != nil {
		http.Error(w, fmt.Sprintf("Failed to switch branch: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"branch": req.BranchName,
	})
}

func (h *MCPHandler) ListFilesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	repoPath := r.URL.Query().Get("repo_path")
	if repoPath == "" {
		http.Error(w, "repo_path parameter is required", http.StatusBadRequest)
		return
	}

	files, err := h.repoManager.ListFiles(repoPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list files: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(files)
}

func (h *MCPHandler) GetFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	repoPath := r.URL.Query().Get("repo_path")
	filePath := r.URL.Query().Get("file_path")
	if repoPath == "" || filePath == "" {
		http.Error(w, "repo_path and file_path parameters are required", http.StatusBadRequest)
		return
	}

	content, err := h.repoManager.GetFileContent(repoPath, filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get file content: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(content))
}

type UpdateFileRequest struct {
	RepoPath string `json:"repo_path"`
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

func (h *MCPHandler) UpdateFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req UpdateFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.repoManager.UpdateFile(req.RepoPath, req.FilePath, req.Content); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update file: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

type CommitRequest struct {
	RepoPath string `json:"repo_path"`
	Message  string `json:"message"`
}

func (h *MCPHandler) CommitChangesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CommitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.repoManager.CommitChanges(req.RepoPath, req.Message); err != nil {
		http.Error(w, fmt.Sprintf("Failed to commit changes: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (h *MCPHandler) PushChangesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		RepoPath string `json:"repo_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	appPassword := r.Context().Value("appPassword").(string)
	if err := h.repoManager.PushChanges(req.RepoPath, appPassword); err != nil {
		http.Error(w, fmt.Sprintf("Failed to push changes: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

type CodeReviewRequest struct {
	RepoPath     string   `json:"repo_path"`
	FilePaths    []string `json:"file_paths"`
	Instructions string   `json:"instructions"`
}

func (h *MCPHandler) RequestCodeReviewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CodeReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	reviewResults, err := h.callOllamaForReview(req.RepoPath, req.FilePaths, req.Instructions)
	if err != nil {
		http.Error(w, fmt.Sprintf("LLM review failed: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(reviewResults)
}

func (h *MCPHandler) callOllamaForReview(repoPath string, filePaths []string, instructions string) ([]map[string]interface{}, error) {
	var filesForReview []map[string]string

	for _, filePath := range filePaths {
		content, err := h.repoManager.GetFileContent(repoPath, filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %v", filePath, err)
		}

		filesForReview = append(filesForReview, map[string]string{
			"file_path": filePath,
			"content":   content,
		})
	}

	ollamaURL := h.ollamaURL + "/api/generate"
	prompt := h.buildPrompt(filesForReview, instructions)

	requestBody := map[string]interface{}{
		"model":  "llama3",
		"prompt": prompt,
		"stream": false,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", ollamaURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Ollama API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama API returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var ollamaResponse struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(body, &ollamaResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return []map[string]interface{}{
		{
			"review": ollamaResponse.Response,
		},
	}, nil
}

func (h *MCPHandler) buildPrompt(files []map[string]string, instructions string) string {
	var prompt strings.Builder
	prompt.WriteString("Please review the following code files and provide feedback based on these instructions:\n")
	prompt.WriteString(instructions + "\n\n")

	for _, file := range files {
		prompt.WriteString(fmt.Sprintf("File: %s\n", file["file_path"]))
		prompt.WriteString("Content:\n```\n")
		prompt.WriteString(file["content"])
		prompt.WriteString("\n```\n\n")
	}

	prompt.WriteString("Please provide your review with:\n")
	prompt.WriteString("- Code quality assessment\n")
	prompt.WriteString("- Suggested improvements\n")
	prompt.WriteString("- Any security concerns\n")
	prompt.WriteString("- Best practices recommendations\n")
	prompt.WriteString("- Specific code changes if applicable\n")

	return prompt.String()
}
