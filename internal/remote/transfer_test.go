package remote

import "testing"

func TestSafeAllowedPathRejectsTraversal(t *testing.T) {
	if _, err := safeAllowedPath("/home/sprite", "../etc/passwd"); err == nil {
		t.Fatal("expected traversal rejection")
	}
}

func TestSafeChildRejectsEscape(t *testing.T) {
	if _, err := safeChild("/home/sprite/.claude", "../../tmp/x"); err == nil {
		t.Fatal("expected escape rejection")
	}
}

func TestSafeAllowedPathAllowsConfiguredTargets(t *testing.T) {
	p, err := safeAllowedPath("/home/sprite", ".config/opencode/settings.json")
	if err != nil {
		t.Fatalf("expected allowlisted path, got error: %v", err)
	}
	if p != "/home/sprite/.config/opencode/settings.json" {
		t.Fatalf("unexpected path: %s", p)
	}
}

func TestSafeChildAllowsNormalRelativePath(t *testing.T) {
	p, err := safeChild("/home/sprite/.claude", "subdir/file.txt")
	if err != nil {
		t.Fatalf("expected safe child path, got error: %v", err)
	}
	if p != "/home/sprite/.claude/subdir/file.txt" {
		t.Fatalf("unexpected path: %s", p)
	}
}
