package remote

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	sprites "github.com/superfly/sprites-go"
)

type TransferConfig struct {
	LocalAuth      string
	LocalEnv       string
	LocalConfigDir string
}

func Transfer(ctx context.Context, sp *sprites.Sprite, cfg TransferConfig) (string, error) {
	home, err := remoteHome(ctx, sp)
	if err != nil {
		return "", err
	}

	repoName, err := repoNameFromLocalEnv(cfg.LocalEnv)
	if err != nil {
		return "", err
	}

	remoteFS := sp.FilesystemAt("/")
	remoteAuth := path.Join(home, ".local", "share", "opencode", "auth.json")
	remoteEnv := path.Join(home, ".env")
	remoteConfigParent := path.Join(home, ".config")
	remoteConfigRoot := path.Join(remoteConfigParent, filepath.Base(filepath.Clean(cfg.LocalConfigDir)))
	remoteRepoParent := path.Join(home, "code")
	remoteRepoDir := path.Join(remoteRepoParent, repoName)
	remoteTmpTar := fmt.Sprintf("/tmp/opencode-config-%d-%d.tar.gz", time.Now().UnixNano(), os.Getpid())

	prepareCmd := sp.CommandContext(ctx, "sh", "-lc", `mkdir -p "$HOME/.local/share/opencode" "$HOME/.config"`)
	if out, err := prepareCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("prepare remote directories: %w: %s", err, strings.TrimSpace(string(out)))
	}

	authBytes, err := os.ReadFile(cfg.LocalAuth)
	if err != nil {
		return "", fmt.Errorf("read local auth.json: %w", err)
	}
	if err := remoteFS.WriteFile(remoteAuth, authBytes, 0o600); err != nil {
		return "", fmt.Errorf("write remote auth.json: %w", err)
	}
	if err := remoteFS.Chmod(remoteAuth, 0o600); err != nil {
		return "", fmt.Errorf("chmod remote auth.json: %w", err)
	}

	envBytes, err := os.ReadFile(cfg.LocalEnv)
	if err != nil {
		return "", fmt.Errorf("read local .env: %w", err)
	}
	if err := remoteFS.WriteFile(remoteEnv, envBytes, 0o600); err != nil {
		return "", fmt.Errorf("write remote .env: %w", err)
	}
	if err := remoteFS.Chmod(remoteEnv, 0o600); err != nil {
		return "", fmt.Errorf("chmod remote .env: %w", err)
	}

	configTar, err := packLocalConfigTar(cfg.LocalConfigDir)
	if err != nil {
		return "", err
	}
	if err := remoteFS.WriteFile(remoteTmpTar, configTar, 0o600); err != nil {
		return "", fmt.Errorf("write remote config tar: %w", err)
	}
	_ = remoteFS.Chmod(remoteTmpTar, 0o600)

	extractCmd := sp.CommandContext(
		ctx,
		"sh",
		"-lc",
		"tar -xzf "+shellQuote(remoteTmpTar)+" -C "+shellQuote(remoteConfigParent)+" && rm -f "+shellQuote(remoteTmpTar),
	)
	if out, err := extractCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("extract remote config tar: %w: %s", err, strings.TrimSpace(string(out)))
	}

	if _, err := remoteFS.Stat(remoteConfigRoot); err != nil {
		return "", fmt.Errorf("verify remote config directory: %w", err)
	}

	if err := remoteFS.MkdirAll(remoteRepoParent, 0o755); err != nil {
		return "", fmt.Errorf("create remote repo parent: %w", err)
	}

	return remoteRepoDir, nil
}

func packLocalConfigTar(localConfigDir string) ([]byte, error) {
	localConfigDir = filepath.Clean(localConfigDir)
	configParent := filepath.Dir(localConfigDir)
	configName := filepath.Base(localConfigDir)

	tmpFile, err := os.CreateTemp("", "opencode-config-*.tar.gz")
	if err != nil {
		return nil, fmt.Errorf("create temp tar file: %w", err)
	}
	tmpTarPath := tmpFile.Name()
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpTarPath)
		return nil, fmt.Errorf("close temp tar file: %w", err)
	}
	defer os.Remove(tmpTarPath)

	if err := runTarWithFallback(configParent, configName, tmpTarPath); err != nil {
		return nil, err
	}

	b, err := os.ReadFile(tmpTarPath)
	if err != nil {
		return nil, fmt.Errorf("read packed config tar: %w", err)
	}
	return b, nil
}

func runTarWithFallback(configParent, configName, tarPath string) error {
	withMacOptions := []string{"--no-xattrs", "--no-mac-metadata", "-C", configParent, "-czf", tarPath, configName}
	if out, err := exec.Command("tar", withMacOptions...).CombinedOutput(); err == nil {
		return nil
	} else {
		basic := []string{"-C", configParent, "-czf", tarPath, configName}
		out2, err2 := exec.Command("tar", basic...).CombinedOutput()
		if err2 == nil {
			return nil
		}
		if msg := strings.TrimSpace(string(out2)); msg != "" {
			return fmt.Errorf("pack local config directory: %w: %s", err2, msg)
		}

		if msg := strings.TrimSpace(string(out)); msg != "" {
			return fmt.Errorf("pack local config directory: %w: %s", err, msg)
		}
		return fmt.Errorf("pack local config directory: %w", err2)
	}
}

func repoNameFromLocalEnv(localEnvPath string) (string, error) {
	name := filepath.Base(filepath.Clean(filepath.Dir(localEnvPath)))
	if name == "" || name == "." || name == string(filepath.Separator) {
		return "", fmt.Errorf("unable to derive repo name from local env path: %s", localEnvPath)
	}
	return name, nil
}

func remoteHome(ctx context.Context, sp *sprites.Sprite) (string, error) {
	cmd := sp.CommandContext(ctx, "sh", "-lc", "printf %s \"$HOME\"")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("resolve remote $HOME: %w", err)
	}
	h := strings.TrimSpace(string(out))
	if h == "" {
		return "", errors.New("remote HOME is empty")
	}
	return h, nil
}
