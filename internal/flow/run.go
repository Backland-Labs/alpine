package flow

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/url"
	"strings"
	"time"

	"alpine/internal/apperr"
	"alpine/internal/cli"
	"alpine/internal/remote"
	spritesclient "alpine/internal/sprites"
	sprites "github.com/superfly/sprites-go"
)

var adjectives = []string{"amber", "brisk", "cedar", "daring", "ember", "frosty", "golden", "hazel", "ivory", "jade", "lunar", "misty", "nimble", "ochre", "quiet", "rapid", "silver", "sunny", "vivid", "wild"}
var nouns = []string{"badger", "canyon", "comet", "delta", "dune", "falcon", "forest", "harbor", "meadow", "mesa", "orbit", "otter", "pine", "quill", "ridge", "river", "summit", "thicket", "valley", "willow"}

func Run(ctx context.Context, cfg cli.Config, stdout, stderr io.Writer) error {
	logf(stderr, "preflight", "start", "repository=%s branch=%s", cfg.RepoName, cfg.Branch)

	client := spritesclient.New(cfg.SpritesToken, cfg.Org)
	defer client.Close()

	name, err := chooseName(ctx, client, cfg.NamePrefix)
	if err != nil {
		return fmt.Errorf("generate sprite name: %w", err)
	}
	logf(stderr, "sprite", "create", "name=%s", name)

	sp, err := createWithRetry(ctx, client, name, cfg.NamePrefix)
	if err != nil {
		return err
	}
	created := true
	cleanupOnErr := func(cause error) error {
		if !created {
			return cause
		}
		logf(stderr, "cleanup", "delete", "sprite=%s", sp.Name())
		if delErr := client.DeleteSprite(ctx, sp.Name()); delErr != nil {
			return fmt.Errorf("%w: %v (cleanup failed: %v)", apperr.ErrCleanup, cause, delErr)
		}
		return cause
	}

	logf(stderr, "bootstrap", "install", "sprite=%s", sp.Name())
	if err := remote.Bootstrap(ctx, sp, stdout, stderr); err != nil {
		return cleanupOnErr(err)
	}

	logf(stderr, "transfer", "start", "sprite=%s", sp.Name())
	repoDir, err := remote.Transfer(ctx, sp, remote.TransferConfig{
		LocalAuth:      cfg.LocalAuth,
		LocalEnv:       cfg.LocalEnvFile,
		LocalConfigDir: cfg.LocalConfigDir,
	})
	if err != nil {
		return cleanupOnErr(err)
	}

	logf(stderr, "git", "setup", "repo=%s", sanitizeRepoURL(cfg.RepoURL))
	if err := remote.GitSetup(ctx, sp, cfg.RepoURL, repoDir, cfg.Branch, stdout, stderr); err != nil {
		return cleanupOnErr(err)
	}

	logf(stderr, "launch", "start", "sprite_id=%s", sp.Name())
	if err := remote.Launch(ctx, sp, repoDir, stdout, stderr); err != nil {
		return cleanupOnErr(err)
	}

	return nil
}

func createWithRetry(ctx context.Context, client *spritesclient.Client, firstName, prefix string) (*sprites.Sprite, error) {
	name := firstName
	for attempt := 1; attempt <= 50; attempt++ {
		sp, createErr := client.CreateSprite(ctx, name)
		if createErr == nil {
			return sp, nil
		}
		if !spritesclient.IsNameCollision(createErr) {
			return nil, fmt.Errorf("create sprite: %w", createErr)
		}
		r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(attempt)))
		name = randomSpriteName(prefix, r)
	}
	return nil, fmt.Errorf("unable to create unique sprite after 50 attempts")
}

func chooseName(ctx context.Context, client *spritesclient.Client, prefix string) (string, error) {
	existing, err := client.ListSpriteNames(ctx)
	if err != nil {
		return "", fmt.Errorf("list sprites: %w", err)
	}
	seen := map[string]struct{}{}
	for _, n := range existing {
		seen[n] = struct{}{}
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 50; i++ {
		cand := randomSpriteName(prefix, r)
		if _, ok := seen[cand]; !ok {
			return cand, nil
		}
	}
	return "", fmt.Errorf("unable to generate a unique sprite name after 50 attempts")
}

func randomSpriteName(prefix string, r *rand.Rand) string {
	a := adjectives[r.Intn(len(adjectives))]
	n := nouns[r.Intn(len(nouns))]
	if prefix == "" {
		return a + "-" + n
	}
	return prefix + "-" + a + "-" + n
}

func logf(w io.Writer, phase, step, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "phase=%s step=%s %s\n", phase, step, msg)
}

func sanitizeRepoURL(raw string) string {
	u, err := url.Parse(raw)
	if err == nil && u.Scheme != "" {
		u.User = nil
		return strings.TrimSpace(u.String())
	}
	trimmed := strings.TrimSpace(raw)
	at := strings.Index(trimmed, "@")
	colon := strings.Index(trimmed, ":")
	if at > 0 && colon > at {
		return trimmed[at+1:]
	}
	return trimmed
}
