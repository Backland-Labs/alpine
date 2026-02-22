package remote

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	sprites "github.com/superfly/sprites-go"
)

type TransferConfig struct {
	LocalAuth      string
	LocalEnv       string
	LocalConfigDir string
	LocalClaudeDir string
}

type backupEntry struct {
	existed bool
	data    []byte
	mode    fs.FileMode
}

func Transfer(ctx context.Context, sp *sprites.Sprite, cfg TransferConfig) (string, error) {
	home, err := remoteHome(ctx, sp)
	if err != nil {
		return "", err
	}
	remoteFS := sp.FilesystemAt("/")

	restore := map[string]backupEntry{}
	created := map[string]struct{}{}
	rollback := func() {
		for p := range created {
			_ = remoteFS.RemoveAll(p)
		}
		for p, b := range restore {
			if !b.existed {
				_ = remoteFS.RemoveAll(p)
				continue
			}
			_ = remoteFS.WriteFile(p, b.data, b.mode)
			_ = remoteFS.Chmod(p, b.mode)
		}
	}

	writeFileSafe := func(dst string, srcBytes []byte, mode fs.FileMode) error {
		if err := backup(remoteFS, dst, restore); err != nil {
			return err
		}
		if err := removeIfExists(remoteFS, dst); err != nil {
			return err
		}
		if err := remoteFS.WriteFile(dst, srcBytes, mode); err != nil {
			return err
		}
		if err := remoteFS.Chmod(dst, mode); err != nil {
			return err
		}
		created[dst] = struct{}{}
		return nil
	}

	authDst, err := safeAllowedPath(home, ".local/share/opencode/auth.json")
	if err != nil {
		return "", err
	}
	envDst, err := safeAllowedPath(home, ".env")
	if err != nil {
		return "", err
	}
	configRoot, err := safeAllowedPath(home, ".config/opencode")
	if err != nil {
		return "", err
	}
	claudeRoot, err := safeAllowedPath(home, ".claude")
	if err != nil {
		return "", err
	}

	authBytes, err := os.ReadFile(cfg.LocalAuth)
	if err != nil {
		return "", fmt.Errorf("read local auth.json: %w", err)
	}
	authParent := path.Dir(authDst)
	if err := remoteFS.MkdirAll(authParent, 0o700); err != nil {
		return "", fmt.Errorf("create remote auth.json parent dir: %w", err)
	}
	if err := writeFileSafe(authDst, authBytes, 0o600); err != nil {
		rollback()
		return "", fmt.Errorf("write remote auth.json: %w", err)
	}


	envBytes, err := os.ReadFile(cfg.LocalEnv)
	if err != nil {
		rollback()
		return "", fmt.Errorf("read local .env: %w", err)
	}
	if err := writeFileSafe(envDst, envBytes, 0o600); err != nil {
		rollback()
		return "", fmt.Errorf("write remote .env: %w", err)
	}

	if err := copyTree(remoteFS, cfg.LocalConfigDir, configRoot, restore, created, true); err != nil {
		rollback()
		return "", fmt.Errorf("copy ~/.config/opencode: %w", err)
	}
	if err := copyTree(remoteFS, cfg.LocalClaudeDir, claudeRoot, restore, created, true); err != nil {
		rollback()
		return "", fmt.Errorf("copy ~/.claude: %w", err)
	}

	_ = remoteFS.Chmod(configRoot, 0o700)
	_ = remoteFS.Chmod(claudeRoot, 0o700)
	credPath := path.Join(claudeRoot, ".credentials.json")
	if _, err := remoteFS.Stat(credPath); err == nil {
		_ = remoteFS.Chmod(credPath, 0o600)
	}

	repoParent, err := safeAllowedPath(home, "code")
	if err != nil {
		return "", err
	}
	repoDir, err := safeAllowedPath(home, path.Join("code", filepath.Base(filepath.Clean(filepath.Dir(cfg.LocalEnv)))))
	if err != nil {
		return "", err
	}
	if err := remoteFS.MkdirAll(repoParent, 0o755); err != nil {
		return "", fmt.Errorf("create remote repo parent: %w", err)
	}
	return repoDir, nil
}

func copyTree(remoteFS sprites.FS, srcRoot, dstRoot string, restore map[string]backupEntry, created map[string]struct{}, strictSecretMode bool) error {
	return filepath.WalkDir(srcRoot, func(srcPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlinks are not allowed: %s", srcPath)
		}
		rel, err := filepath.Rel(srcRoot, srcPath)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			if err := remoteFS.MkdirAll(dstRoot, 0o700); err != nil {
				return err
			}
			created[dstRoot] = struct{}{}
			return nil
		}
		dst, err := safeChild(dstRoot, rel)
		if err != nil {
			return err
		}
		if d.IsDir() {
			if err := remoteFS.MkdirAll(dst, 0o700); err != nil {
				return err
			}
			created[dst] = struct{}{}
			return nil
		}

		data, err := os.ReadFile(srcPath)
		if err != nil {
			return err
		}
		if err := backup(remoteFS, dst, restore); err != nil {
			return err
		}
		if err := removeIfExists(remoteFS, dst); err != nil {
			return err
		}
		mode := fs.FileMode(0o644)
		if strictSecretMode {
			mode = 0o600
		}
		if strings.HasSuffix(srcPath, ".credentials.json") {
			mode = 0o600
		}
		if err := remoteFS.WriteFile(dst, data, mode); err != nil {
			return err
		}
		if err := remoteFS.Chmod(dst, mode); err != nil {
			return err
		}
		created[dst] = struct{}{}
		return nil
	})
}

func backup(remoteFS sprites.FS, dst string, store map[string]backupEntry) error {
	if _, ok := store[dst]; ok {
		return nil
	}
	st, err := remoteFS.Stat(dst)
	if err != nil {
		store[dst] = backupEntry{existed: false}
		return nil
	}
	if st.IsDir() {
		return errors.New("refusing to overwrite directory with file: " + dst)
	}
	b, err := remoteFS.ReadFile(dst)
	if err != nil {
		return err
	}
	store[dst] = backupEntry{existed: true, data: b, mode: st.Mode()}
	return nil
}

func removeIfExists(remoteFS sprites.FS, dst string) error {
	if _, err := remoteFS.Stat(dst); err != nil {
		return nil
	}
	if err := remoteFS.Remove(dst); err != nil {
		return err
	}
	return nil
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

func safeAllowedPath(home, rel string) (string, error) {
	if strings.HasPrefix(rel, "/") {
		return "", fmt.Errorf("absolute path not allowed: %s", rel)
	}
	if strings.Contains(rel, "..") {
		return "", fmt.Errorf("path traversal rejected: %s", rel)
	}
	clean := path.Clean(path.Join(home, rel))
	allowed := []string{
		path.Join(home, ".local", "share", "opencode"),
		path.Join(home, ".config", "opencode"),
		path.Join(home, ".claude"),
		path.Join(home, ".env"),
		path.Join(home, "code"),
	}
	for _, base := range allowed {
		if clean == base || strings.HasPrefix(clean, base+"/") {
			return clean, nil
		}
	}
	return "", fmt.Errorf("destination not in allowlist: %s", clean)
}

func safeChild(root, rel string) (string, error) {
	rel = strings.TrimPrefix(rel, "./")
	if strings.HasPrefix(rel, "/") || strings.Contains(rel, "..") {
		return "", fmt.Errorf("invalid relative path: %s", rel)
	}
	out := path.Clean(path.Join(root, rel))
	if out != root && !strings.HasPrefix(out, root+"/") {
		return "", fmt.Errorf("symlink/path escape rejected: %s", rel)
	}
	return out, nil
}
