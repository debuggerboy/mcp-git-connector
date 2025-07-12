ipackage repository

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type GitManager struct {
	baseDir string
}

func NewGitManager(baseDir string) *GitManager {
	// Ensure base directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		panic(fmt.Sprintf("failed to create base directory: %v", err))
	}
	return &GitManager{baseDir: baseDir}
}

func (gm *GitManager) CloneRepository(repoURL, appPassword, repoName string) (string, error) {
	// Format URL with authentication
	authURL := strings.Replace(repoURL, "https://", fmt.Sprintf("https://x-token-auth:%s@", appPassword), 1)
	
	repoPath := filepath.Join(gm.baseDir, repoName)
	
	cmd := exec.Command("git", "clone", authURL, repoPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git clone failed: %v, output: %s", err, string(output))
	}
	
	return repoPath, nil
}

func (gm *GitManager) SwitchBranch(repoPath, branchName string) error {
	cmd := exec.Command("git", "checkout", branchName)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout failed: %v, output: %s", err, string(output))
	}
	return nil
}

func (gm *GitManager) ListFiles(repoPath string) ([]string, error) {
	var files []string
	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && !strings.Contains(path, "/.git/") {
			relPath, _ := filepath.Rel(repoPath, path)
			files = append(files, relPath)
		}
		return nil
	})
	return files, err
}

func (gm *GitManager) GetFileContent(repoPath, filePath string) (string, error) {
	content, err := os.ReadFile(filepath.Join(repoPath, filePath))
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (gm *GitManager) UpdateFile(repoPath, filePath, content string) error {
	fullPath := filepath.Join(repoPath, filePath)
	return os.WriteFile(fullPath, []byte(content), 0644)
}

func (gm *GitManager) CommitChanges(repoPath, message string) error {
	// Add all changes
	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = repoPath
	if output, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %v, output: %s", err, string(output))
	}

	// Commit
	commitCmd := exec.Command("git", "commit", "-m", message)
	commitCmd.Dir = repoPath
	if output, err := commitCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit failed: %v, output: %s", err, string(output))
	}

	return nil
}

func (gm *GitManager) PushChanges(repoPath, appPassword string) error {
	// Get remote URL to inject auth
	remoteCmd := exec.Command("git", "remote", "get-url", "origin")
	remoteCmd.Dir = repoPath
	remoteURL, err := remoteCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get remote URL: %v", err)
	}

	// Inject auth into URL
	authURL := strings.TrimSpace(string(remoteURL))
	authURL = strings.Replace(authURL, "https://", fmt.Sprintf("https://x-token-auth:%s@", appPassword), 1)

	// Push changes
	pushCmd := exec.Command("git", "push", authURL)
	pushCmd.Dir = repoPath
	if output, err := pushCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push failed: %v, output: %s", err, string(output))
	}

	return nil
}
