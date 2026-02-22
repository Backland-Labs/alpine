package spritesclient

import (
	"context"
	"fmt"
	"strings"

	sprites "github.com/superfly/sprites-go"
)

type Client struct {
	sdk *sprites.Client
	org *sprites.OrganizationInfo
}

func New(token, org string) *Client {
	var orgInfo *sprites.OrganizationInfo
	if trimmed := strings.TrimSpace(org); trimmed != "" {
		orgInfo = &sprites.OrganizationInfo{Name: trimmed}
	}
	return &Client{sdk: sprites.New(token), org: orgInfo}
}

func (c *Client) Close() error {
	if c == nil || c.sdk == nil {
		return nil
	}
	return c.sdk.Close()
}

func (c *Client) CreateSprite(ctx context.Context, name string) (*sprites.Sprite, error) {
	var (
		sp  *sprites.Sprite
		err error
	)
	if c.org != nil {
		sp, err = c.sdk.CreateSpriteWithOrg(ctx, name, nil, c.org)
	} else {
		sp, err = c.sdk.CreateSprite(ctx, name, nil)
	}
	if err != nil {
		return nil, err
	}
	if c.org != nil {
		return c.sdk.GetSpriteWithOrg(ctx, sp.Name(), c.org)
	}
	return c.sdk.GetSprite(ctx, sp.Name())
}

func (c *Client) DeleteSprite(ctx context.Context, name string) error {
	return c.sdk.DeleteSprite(ctx, name)
}

func (c *Client) ListSpriteNames(ctx context.Context) ([]string, error) {
	var (
		all []*sprites.Sprite
		err error
	)
	if c.org != nil {
		all, err = c.sdk.ListAllSpritesWithOrg(ctx, "", c.org)
	} else {
		all, err = c.sdk.ListAllSprites(ctx, "")
	}
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(all))
	for _, sp := range all {
		out = append(out, sp.Name())
	}
	return out, nil
}

func (c *Client) Sprite(name string) *sprites.Sprite {
	if c.org != nil {
		return c.sdk.SpriteWithOrg(name, c.org)
	}
	return c.sdk.Sprite(name)
}

func IsNameCollision(err error) bool {
	if err == nil {
		return false
	}
	s := fmt.Sprintf("%v", err)
	return containsAny(s, []string{"already exists", "name is taken", "409"})
}

func containsAny(s string, parts []string) bool {
	for _, p := range parts {
		if p != "" && containsFold(s, p) {
			return true
		}
	}
	return false
}

func containsFold(s, sub string) bool {
	if len(sub) > len(s) {
		return false
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if equalFoldASCII(s[i:i+len(sub)], sub) {
			return true
		}
	}
	return false
}

func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca := a[i]
		cb := b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
