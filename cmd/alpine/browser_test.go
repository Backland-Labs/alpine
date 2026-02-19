package main

import (
	"context"
	"testing"
)

func TestDefaultOpenBrowser(t *testing.T) {
	t.Run("darwin success", func(t *testing.T) {
		mockRun(t, []cmdResult{{stdout: ""}})
		if !defaultOpenBrowser(context.Background(), "https://example.com", "darwin") {
			t.Fatal("expected true")
		}
	})

	t.Run("darwin failure", func(t *testing.T) {
		mockRun(t, []cmdResult{errResult("open failed")})
		if defaultOpenBrowser(context.Background(), "https://example.com", "darwin") {
			t.Fatal("expected false")
		}
	})

	t.Run("linux success", func(t *testing.T) {
		mockRun(t, []cmdResult{{stdout: ""}})
		if !defaultOpenBrowser(context.Background(), "https://example.com", "linux") {
			t.Fatal("expected true")
		}
	})

	t.Run("linux failure", func(t *testing.T) {
		mockRun(t, []cmdResult{errResult("xdg-open failed")})
		if defaultOpenBrowser(context.Background(), "https://example.com", "linux") {
			t.Fatal("expected false")
		}
	})
}
