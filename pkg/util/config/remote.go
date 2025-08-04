package config

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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
	if !strings.HasPrefix(remote, "http") {
		remote = "http://" + remote
	}
	remote = fmt.Sprintf("%s/api/dcd/%s", remote, token)
	fmt.Printf("Fetching remote config from %s\n", remote)
	u, err := url.Parse(remote)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("GET", u.String(), nil)
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
