package main

import (
	"context"
	"time"
)

var openBrowser = defaultOpenBrowser

func defaultOpenBrowser(ctx context.Context, targetURL, platform string) bool {
	browserCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if platform == "darwin" {
		_, _, err := run(browserCtx, "open", targetURL)
		return err == nil
	}

	_, _, err := run(browserCtx, "xdg-open", targetURL)
	return err == nil
}
