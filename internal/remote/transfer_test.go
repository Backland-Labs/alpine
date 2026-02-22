package remote

import "testing"

func TestRepoNameFromLocalEnv(t *testing.T) {
	name, err := repoNameFromLocalEnv("/Users/max/code/alpine/.env")
	if err != nil {
		t.Fatalf("expected repo name, got error: %v", err)
	}
	if name != "alpine" {
		t.Fatalf("unexpected repo name: %s", name)
	}
}

func TestRepoNameFromLocalEnvRejectsEmpty(t *testing.T) {
	if _, err := repoNameFromLocalEnv(""); err == nil {
		t.Fatal("expected error for empty env path")
	}
}
