package config

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func getConfigPath() string {
	userHome, err := os.UserHomeDir()
	if err != nil {
		userHome = "."
	}
	return filepath.Join(userHome, ".frp.client.json")
}

func FetchRemoteConfig(remote, token string) (string, error) {
	res := getConfigPath()
	req, err := http.NewRequest("GET", remote, nil)
	if err != nil {
		return "", fmt.Errorf("failed to fetch remote config: %w", err)
	}

	req.Header.Add("Authorization", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch remote config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch remote config: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to fetch remote config: %w", err)
	}
	if err = os.WriteFile(res, body, 0600); err != nil {
		return "", fmt.Errorf("failed to fetch remote config: %w", err)
	}
	return res, nil
}
