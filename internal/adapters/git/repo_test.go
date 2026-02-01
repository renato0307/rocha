package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsGitURL_HTTPUrls(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://github.com/owner/repo", true},
		{"https://github.com/owner/repo.git", true},
		{"http://github.com/owner/repo", true},
		{"https://gitlab.com/owner/repo.git", true},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := isGitURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsGitURL_SSHUrls(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"git@github.com:owner/repo", true},
		{"git@github.com:owner/repo.git", true},
		{"git@gitlab.com:owner/repo.git", true},
		{"ssh://git@github.com/owner/repo", true},
		{"user@host.com:path/repo", true},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := isGitURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsGitURL_GitProtocol(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"git://github.com/owner/repo", true},
		{"git://github.com/owner/repo.git", true},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := isGitURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsGitURL_FTPUrls(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"ftp://example.com/repo.git", true},
		{"ftps://example.com/repo.git", true},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := isGitURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsGitURL_LocalPaths(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/home/user/repo", false},
		{"./relative/path", false},
		{"~/projects/repo", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isGitURL(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsGitURL_DotGitSuffix(t *testing.T) {
	// Paths ending with .git are considered URLs
	tests := []struct {
		path     string
		expected bool
	}{
		{"/path/to/repo.git", true},
		{"repo.git", true},
		{"/path/to/repo.git/", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isGitURL(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseRepoSource_EmptySource(t *testing.T) {
	_, err := parseRepoSource("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestParseRepoSource_HTTPSUrl(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		expectedOwner string
		expectedRepo  string
	}{
		{"github", "https://github.com/owner/repo", "owner", "repo"},
		{"github with .git", "https://github.com/owner/repo.git", "owner", "repo"},
		{"gitlab", "https://gitlab.com/owner/repo.git", "owner", "repo"},
		{"nested path", "https://github.com/org/subgroup/repo", "subgroup", "repo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseRepoSource(tt.url)
			require.NoError(t, err)
			assert.True(t, result.isRemote)
			assert.Equal(t, tt.expectedOwner, result.owner)
			assert.Equal(t, tt.expectedRepo, result.repo)
		})
	}
}

func TestParseRepoSource_SSHUrl(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		expectedOwner string
		expectedRepo  string
	}{
		{"github", "git@github.com:owner/repo", "owner", "repo"},
		{"github with .git", "git@github.com:owner/repo.git", "owner", "repo"},
		{"gitlab", "git@gitlab.com:owner/repo.git", "owner", "repo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseRepoSource(tt.url)
			require.NoError(t, err)
			assert.True(t, result.isRemote)
			assert.Equal(t, tt.expectedOwner, result.owner)
			assert.Equal(t, tt.expectedRepo, result.repo)
		})
	}
}

func TestParseRepoSource_SSHProtocolUrl(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		expectedOwner string
		expectedRepo  string
	}{
		{"with user", "ssh://git@github.com/owner/repo", "owner", "repo"},
		{"with .git", "ssh://git@github.com/owner/repo.git", "owner", "repo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseRepoSource(tt.url)
			require.NoError(t, err)
			assert.True(t, result.isRemote)
			assert.Equal(t, tt.expectedOwner, result.owner)
			assert.Equal(t, tt.expectedRepo, result.repo)
		})
	}
}

func TestParseRepoSource_BranchFragment(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		expectedBranch string
		expectedPath   string
	}{
		{"with branch", "https://github.com/owner/repo#feature-branch", "feature-branch", "https://github.com/owner/repo"},
		{"with main", "https://github.com/owner/repo#main", "main", "https://github.com/owner/repo"},
		{"no branch", "https://github.com/owner/repo", "", "https://github.com/owner/repo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseRepoSource(tt.url)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedBranch, result.branch)
			assert.Equal(t, tt.expectedPath, result.path)
		})
	}
}

func TestParseRepoSource_LocalPath(t *testing.T) {
	result, err := parseRepoSource("/home/user/repo")
	require.NoError(t, err)
	assert.False(t, result.isRemote)
	assert.Equal(t, "/home/user/repo", result.path)
}

func TestIsSameRepo_NormalizeGitSuffix(t *testing.T) {
	tests := []struct {
		name     string
		url1     string
		url2     string
		expected bool
	}{
		{"same url", "https://github.com/owner/repo", "https://github.com/owner/repo", true},
		{"one with .git", "https://github.com/owner/repo", "https://github.com/owner/repo.git", true},
		{"both with .git", "https://github.com/owner/repo.git", "https://github.com/owner/repo.git", true},
		{"trailing slash", "https://github.com/owner/repo/", "https://github.com/owner/repo", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSameRepo(tt.url1, tt.url2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSameRepo_CaseInsensitivity(t *testing.T) {
	tests := []struct {
		name     string
		url1     string
		url2     string
		expected bool
	}{
		{"different case", "https://github.com/Owner/Repo", "https://github.com/owner/repo", true},
		{"mixed case", "https://GitHub.COM/owner/repo", "https://github.com/owner/repo", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSameRepo(tt.url1, tt.url2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSameRepo_DifferentProtocols(t *testing.T) {
	tests := []struct {
		name     string
		url1     string
		url2     string
		expected bool
	}{
		{"https vs ssh", "https://github.com/owner/repo", "git@github.com:owner/repo", true},
		{"https vs ssh .git", "https://github.com/owner/repo.git", "git@github.com:owner/repo.git", true},
		{"http vs https", "http://github.com/owner/repo", "https://github.com/owner/repo", true},
		{"ssh:// vs git@", "ssh://git@github.com/owner/repo", "git@github.com:owner/repo", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSameRepo(tt.url1, tt.url2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsSameRepo_DifferentRepos(t *testing.T) {
	tests := []struct {
		name     string
		url1     string
		url2     string
		expected bool
	}{
		{"different owner", "https://github.com/owner1/repo", "https://github.com/owner2/repo", false},
		{"different repo", "https://github.com/owner/repo1", "https://github.com/owner/repo2", false},
		{"different host", "https://github.com/owner/repo", "https://gitlab.com/owner/repo", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSameRepo(tt.url1, tt.url2)
			assert.Equal(t, tt.expected, result)
		})
	}
}
