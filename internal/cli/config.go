package cli

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"alpine/internal/apperr"
)

const usageText = `Usage: setup-sprite-opencode --branch <branch-name> [--org <org-name>]

Creates a Sprite environment with a repo/branch-tagged random name, installs OpenCode and ast-grep,
copies ~/.local/share/opencode/auth.json, ~/.config/opencode, ~/.claude, and .env,
then clones the current repository in the Sprite and checks out the target branch.

Options:
  -b, --branch <branch-name> Branch to check out/create inside sprite (required)
  -o, --org <org-name>       Sprite organization name (currently fail-closed)
  -h, --help                 Show this help`

type Config struct {
	Branch         string
	Org            string
	ShowHelp       bool
	RepoRoot       string
	RepoURL        string
	RepoName       string
	RepoSlug       string
	BranchSlug     string
	NamePrefix     string
	LocalAuth      string
	LocalConfigDir string
	LocalClaudeDir string
	LocalEnvFile   string
	SpritesToken   string
}

func Parse(args []string, env []string, stderr io.Writer) (Config, error) {
	cfg := Config{}

	fs := flag.NewFlagSet("setup-sprite-opencode", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&cfg.Branch, "branch", "", "")
	fs.StringVar(&cfg.Branch, "b", "", "")
	fs.StringVar(&cfg.Org, "org", os.Getenv("SPRITE_ORG"), "")
	fs.StringVar(&cfg.Org, "o", os.Getenv("SPRITE_ORG"), "")
	fs.BoolVar(&cfg.ShowHelp, "help", false, "")
	fs.BoolVar(&cfg.ShowHelp, "h", false, "")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(stderr, usageText)
		return cfg, fmt.Errorf("%w: %v", apperr.ErrUsage, err)
	}
	if fs.NArg() > 0 {
		fmt.Fprintln(stderr, usageText)
		return cfg, fmt.Errorf("%w: unexpected argument: %s", apperr.ErrUsage, fs.Arg(0))
	}
	if cfg.ShowHelp {
		fmt.Fprintln(stderr, usageText)
		return cfg, nil
	}
	if strings.TrimSpace(cfg.Branch) == "" {
		fmt.Fprintln(stderr, usageText)
		return cfg, fmt.Errorf("%w: missing required argument --branch", apperr.ErrUsage)
	}
	if err := checkBranchName(cfg.Branch); err != nil {
		return cfg, fmt.Errorf("%w: invalid branch name %q: %v", apperr.ErrUsage, cfg.Branch, err)
	}

	if strings.TrimSpace(cfg.Org) != "" {
		return cfg, fmt.Errorf("%w: --org is currently fail-closed; sprites-go does not guarantee org scoping for create/list/exec operations", apperr.ErrAuth)
	}

	repoRoot, err := gitOut("rev-parse", "--show-toplevel")
	if err != nil {
		return cfg, fmt.Errorf("%w: must run inside a git repository", apperr.ErrPreflight)
	}
	repoURL, err := gitOut("-C", repoRoot, "config", "--get", "remote.origin.url")
	if err != nil || strings.TrimSpace(repoURL) == "" {
		return cfg, fmt.Errorf("%w: unable to determine remote.origin.url for %s", apperr.ErrPreflight, repoRoot)
	}
	repoName := strings.TrimSuffix(filepath.Base(repoURL), ".git")
	if repoName == "" || repoName == "." {
		return cfg, fmt.Errorf("%w: unable to derive repository name from %s", apperr.ErrPreflight, repoURL)
	}

	cfg.RepoRoot = repoRoot
	cfg.RepoURL = repoURL
	cfg.RepoName = repoName
	cfg.RepoSlug = truncate(slugify(repoName), 16)
	cfg.BranchSlug = truncate(slugify(cfg.Branch), 16)
	cfg.NamePrefix = nonEmpty(cfg.RepoSlug, "repo") + "-" + nonEmpty(cfg.BranchSlug, "branch")
	cfg.LocalAuth = filepath.Join(os.Getenv("HOME"), ".local", "share", "opencode", "auth.json")
	cfg.LocalConfigDir = filepath.Join(os.Getenv("HOME"), ".config", "opencode")
	cfg.LocalClaudeDir = filepath.Join(os.Getenv("HOME"), ".claude")
	cfg.LocalEnvFile = filepath.Join(repoRoot, ".env")

	if err := requireFile(cfg.LocalAuth); err != nil {
		return cfg, fmt.Errorf("%w: %v", apperr.ErrPreflight, err)
	}
	if err := requireDir(cfg.LocalConfigDir); err != nil {
		return cfg, fmt.Errorf("%w: %v", apperr.ErrPreflight, err)
	}
	if err := requireDir(cfg.LocalClaudeDir); err != nil {
		return cfg, fmt.Errorf("%w: %v", apperr.ErrPreflight, err)
	}
	if err := requireFile(cfg.LocalEnvFile); err != nil {
		return cfg, fmt.Errorf("%w: %v", apperr.ErrPreflight, err)
	}

	token, warnDifferent, err := resolveToken(env, cfg.LocalAuth)
	if err != nil {
		return cfg, fmt.Errorf("%w: %v", apperr.ErrAuth, err)
	}
	if warnDifferent {
		fmt.Fprintln(stderr, "warning: SPRITES_TOKEN differs from auth.json token; preferring SPRITES_TOKEN")
	}
	cfg.SpritesToken = token

	return cfg, nil
}

func checkBranchName(branch string) error {
	cmd := exec.Command("git", "check-ref-format", "--branch", branch)
	if err := cmd.Run(); err != nil {
		return errors.New("git check-ref-format failed")
	}
	return nil
}

func gitOut(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func requireFile(path string) error {
	st, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("missing file: %s", path)
	}
	if st.IsDir() {
		return fmt.Errorf("expected file but found directory: %s", path)
	}
	return nil
}

func requireDir(path string) error {
	st, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("missing directory: %s", path)
	}
	if !st.IsDir() {
		return fmt.Errorf("expected directory but found file: %s", path)
	}
	return nil
}

func resolveToken(env []string, authPath string) (token string, warnDifferent bool, err error) {
	return resolveTokenWithSpriteLookup(env, authPath, tokenFromSpritesLogin)
}

func resolveTokenWithSpriteLookup(env []string, authPath string, spriteLookup func() (string, error)) (token string, warnDifferent bool, err error) {
	envToken := ""
	for _, kv := range env {
		if strings.HasPrefix(kv, "SPRITES_TOKEN=") {
			envToken = strings.TrimPrefix(kv, "SPRITES_TOKEN=")
			break
		}
	}
	authToken, _ := tokenFromAuthJSON(authPath)

	if envToken != "" {
		if authToken != "" && authToken != envToken {
			return envToken, true, nil
		}
		return envToken, false, nil
	}
	if authToken != "" {
		return authToken, false, nil
	}
	if spriteLookup != nil {
		spriteToken, spriteErr := spriteLookup()
		if spriteErr == nil && strings.TrimSpace(spriteToken) != "" {
			return strings.TrimSpace(spriteToken), false, nil
		}
	}
	return "", false, errors.New("missing SPRITES_TOKEN and no token found in auth.json or sprites login")
}

type spritesConfig struct {
	CurrentSelection struct {
		URL string `json:"url"`
		Org string `json:"org"`
	} `json:"current_selection"`
	URLs map[string]struct {
		Orgs map[string]spritesOrg `json:"orgs"`
	} `json:"urls"`
	Users []struct {
		ID string `json:"id"`
	} `json:"users"`
	CurrentUser string `json:"current_user"`
}

type spritesOrg struct {
	KeyringKey string `json:"keyring_key"`
	UseKeyring *bool  `json:"use_keyring"`
	APIToken   string `json:"api_token"`
	Token      string `json:"token"`
}

func tokenFromSpritesLogin() (string, error) {
	home := strings.TrimSpace(os.Getenv("HOME"))
	if home == "" {
		return "", errors.New("HOME is not set")
	}
	configPath := filepath.Join(home, ".sprites", "sprites.json")
	b, err := os.ReadFile(configPath)
	if err != nil {
		return "", err
	}

	var cfg spritesConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return "", err
	}

	url := strings.TrimSpace(cfg.CurrentSelection.URL)
	if url == "" {
		url = "https://api.sprites.dev"
	}
	org := strings.TrimSpace(cfg.CurrentSelection.Org)
	if org == "" {
		return "", errors.New("sprites current_selection.org is empty")
	}

	orgCfg, ok := spritesOrgConfig(cfg, url, org)
	if !ok {
		return "", fmt.Errorf("sprites org config not found for %s at %s", org, url)
	}

	if tok := strings.TrimSpace(firstNonEmpty(orgCfg.APIToken, orgCfg.Token)); tok != "" {
		return tok, nil
	}

	userID := strings.TrimSpace(cfg.CurrentUser)
	if userID == "" && len(cfg.Users) > 0 {
		userID = strings.TrimSpace(cfg.Users[0].ID)
	}
	if userID == "" {
		return "", errors.New("sprites current_user is empty")
	}

	service := "sprites-cli:" + userID
	account := strings.TrimSpace(orgCfg.KeyringKey)
	if account == "" {
		account = fmt.Sprintf("sprites:org:%s:%s", url, org)
	}

	if _, err := exec.LookPath("security"); err != nil {
		return "", err
	}

	cmd := exec.Command("security", "find-generic-password", "-s", service, "-a", account, "-w")
	out, err := cmd.Output()
	if err != nil {
		cmd = exec.Command("security", "find-generic-password", "-a", account, "-w")
		out, err = cmd.Output()
		if err != nil {
			return "", err
		}
	}

	raw := strings.TrimSpace(string(out))
	if strings.HasPrefix(raw, "go-keyring-base64:") {
		enc := strings.TrimPrefix(raw, "go-keyring-base64:")
		decoded, err := base64.StdEncoding.DecodeString(enc)
		if err != nil {
			return "", err
		}
		raw = string(decoded)
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("sprites token was empty")
	}
	return raw, nil
}

func spritesOrgConfig(cfg spritesConfig, url, org string) (spritesOrg, bool) {
	if byURL, ok := cfg.URLs[url]; ok {
		if orgCfg, ok := byURL.Orgs[org]; ok {
			return orgCfg, true
		}
	}
	for _, byURL := range cfg.URLs {
		if orgCfg, ok := byURL.Orgs[org]; ok {
			return orgCfg, true
		}
	}
	return spritesOrg{}, false
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func tokenFromAuthJSON(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var generic map[string]any
	if err := json.Unmarshal(b, &generic); err != nil {
		return "", err
	}
	keys := []string{"sprites_token", "token", "access_token", "api_key"}
	for _, k := range keys {
		if v, ok := generic[k]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s, nil
			}
		}
	}
	if v, ok := generic["auth"]; ok {
		if nested, ok := v.(map[string]any); ok {
			for _, k := range keys {
				if raw, ok := nested[k]; ok {
					if s, ok := raw.(string); ok && s != "" {
						return s, nil
					}
				}
			}
		}
	}
	return "", nil
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
		case r == '-', r == '/', r == '.', r == '_', r == ' ':
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func nonEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
