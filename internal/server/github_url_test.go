package server

import (
	"testing"
)

// TestParseGitHubIssueURL tests parsing valid GitHub issue URLs
func TestParseGitHubIssueURL(t *testing.T) {
	tests := []struct {
		name      string
		issueURL  string
		wantOwner string
		wantRepo  string
		wantIssue int
		wantErr   bool
	}{
		{
			name:      "valid GitHub issue URL",
			issueURL:  "https://github.com/owner/repo/issues/123",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantIssue: 123,
			wantErr:   false,
		},
		{
			name:      "valid GitHub issue URL with hyphenated owner",
			issueURL:  "https://github.com/my-org/my-repo/issues/456",
			wantOwner: "my-org",
			wantRepo:  "my-repo",
			wantIssue: 456,
			wantErr:   false,
		},
		{
			name:      "valid GitHub issue URL with underscored names",
			issueURL:  "https://github.com/user_name/repo_name/issues/789",
			wantOwner: "user_name",
			wantRepo:  "repo_name",
			wantIssue: 789,
			wantErr:   false,
		},
		{
			name:      "invalid URL - not GitHub",
			issueURL:  "https://gitlab.com/owner/repo/issues/123",
			wantOwner: "",
			wantRepo:  "",
			wantIssue: 0,
			wantErr:   true,
		},
		{
			name:      "invalid URL - missing issue number",
			issueURL:  "https://github.com/owner/repo/issues/",
			wantOwner: "",
			wantRepo:  "",
			wantIssue: 0,
			wantErr:   true,
		},
		{
			name:      "invalid URL - not issues path",
			issueURL:  "https://github.com/owner/repo/pull/123",
			wantOwner: "",
			wantRepo:  "",
			wantIssue: 0,
			wantErr:   true,
		},
		{
			name:      "invalid URL - malformed",
			issueURL:  "not-a-url",
			wantOwner: "",
			wantRepo:  "",
			wantIssue: 0,
			wantErr:   true,
		},
		{
			name:      "empty URL",
			issueURL:  "",
			wantOwner: "",
			wantRepo:  "",
			wantIssue: 0,
			wantErr:   true,
		},
		{
			name:      "invalid issue number - not numeric",
			issueURL:  "https://github.com/owner/repo/issues/abc",
			wantOwner: "",
			wantRepo:  "",
			wantIssue: 0,
			wantErr:   true,
		},
		{
			name:      "invalid URL - missing owner",
			issueURL:  "https://github.com//repo/issues/123",
			wantOwner: "",
			wantRepo:  "",
			wantIssue: 0,
			wantErr:   true,
		},
		{
			name:      "invalid URL - missing repo",
			issueURL:  "https://github.com/owner//issues/123",
			wantOwner: "",
			wantRepo:  "",
			wantIssue: 0,
			wantErr:   true,
		},
		{
			name:      "invalid issue number - zero",
			issueURL:  "https://github.com/owner/repo/issues/0",
			wantOwner: "",
			wantRepo:  "",
			wantIssue: 0,
			wantErr:   true,
		},
		{
			name:      "invalid issue number - negative",
			issueURL:  "https://github.com/owner/repo/issues/-1",
			wantOwner: "",
			wantRepo:  "",
			wantIssue: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, issueNum, err := parseGitHubIssueURL(tt.issueURL)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseGitHubIssueURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if owner != tt.wantOwner {
				t.Errorf("parseGitHubIssueURL() owner = %v, want %v", owner, tt.wantOwner)
			}

			if repo != tt.wantRepo {
				t.Errorf("parseGitHubIssueURL() repo = %v, want %v", repo, tt.wantRepo)
			}

			if issueNum != tt.wantIssue {
				t.Errorf("parseGitHubIssueURL() issueNum = %v, want %v", issueNum, tt.wantIssue)
			}
		})
	}
}

// TestBuildGitCloneURL tests building git clone URLs from owner/repo
func TestBuildGitCloneURL(t *testing.T) {
	tests := []struct {
		name     string
		owner    string
		repo     string
		expected string
	}{
		{
			name:     "basic owner/repo",
			owner:    "owner",
			repo:     "repo",
			expected: "https://github.com/owner/repo.git",
		},
		{
			name:     "hyphenated names",
			owner:    "my-org",
			repo:     "my-repo",
			expected: "https://github.com/my-org/my-repo.git",
		},
		{
			name:     "underscored names",
			owner:    "user_name",
			repo:     "repo_name",
			expected: "https://github.com/user_name/repo_name.git",
		},
		{
			name:     "numeric names",
			owner:    "org123",
			repo:     "repo456",
			expected: "https://github.com/org123/repo456.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildGitCloneURL(tt.owner, tt.repo)
			if result != tt.expected {
				t.Errorf("buildGitCloneURL() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestIsGitHubIssueURL tests detecting GitHub issue URLs
func TestIsGitHubIssueURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{
			name: "valid GitHub issue URL",
			url:  "https://github.com/owner/repo/issues/123",
			want: true,
		},
		{
			name: "GitHub pull request URL",
			url:  "https://github.com/owner/repo/pull/123",
			want: false,
		},
		{
			name: "GitHub repository URL",
			url:  "https://github.com/owner/repo",
			want: false,
		},
		{
			name: "GitLab issue URL",
			url:  "https://gitlab.com/owner/repo/issues/123",
			want: false,
		},
		{
			name: "empty string",
			url:  "",
			want: false,
		},
		{
			name: "malformed URL",
			url:  "not-a-url",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isGitHubIssueURL(tt.url); got != tt.want {
				t.Errorf("isGitHubIssueURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
