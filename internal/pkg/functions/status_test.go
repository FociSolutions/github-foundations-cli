package functions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFindConfigFiles_SinglePattern tests finding files with a single pattern
func TestFindConfigFiles_SinglePattern(t *testing.T) {
	// Create temporary directory structure
	tmpDir, err := os.MkdirTemp("", "test-findconfig-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test files
	projectDir := filepath.Join(tmpDir, "project1", "org1", "repositories")
	err = os.MkdirAll(projectDir, 0755)
	require.NoError(t, err)

	testFile := filepath.Join(projectDir, "terragrunt.hcl")
	err = os.WriteFile(testFile, []byte("# test file"), 0644)
	require.NoError(t, err)

	// Test finding files with single pattern
	files, err := findConfigFiles(tmpDir, "repositories/terragrunt.hcl")
	require.NoError(t, err)
	assert.Equal(t, 1, len(files))
	assert.Contains(t, files[0], "repositories/terragrunt.hcl")
}

// TestFindConfigFiles_MultiplePatterns tests finding files with multiple patterns
func TestFindConfigFiles_MultiplePatterns(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-findconfig-multi-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test files with different patterns
	org1Dir := filepath.Join(tmpDir, "org1")
	err = os.MkdirAll(org1Dir, 0755)
	require.NoError(t, err)

	org2Dir := filepath.Join(tmpDir, "org2")
	err = os.MkdirAll(org2Dir, 0755)
	require.NoError(t, err)

	// Create providers.hcl
	providersFile := filepath.Join(org1Dir, "providers.hcl")
	err = os.WriteFile(providersFile, []byte("# providers file"), 0644)
	require.NoError(t, err)

	// Create root.hcl
	rootFile := filepath.Join(org2Dir, "root.hcl")
	err = os.WriteFile(rootFile, []byte("# root file"), 0644)
	require.NoError(t, err)

	// Test finding files with multiple patterns
	files, err := findConfigFiles(tmpDir, "providers.hcl", "root.hcl")
	require.NoError(t, err)
	assert.Equal(t, 2, len(files))

	// Check both files were found
	foundProviders := false
	foundRoot := false
	for _, f := range files {
		if filepath.Base(f) == "providers.hcl" {
			foundProviders = true
		}
		if filepath.Base(f) == "root.hcl" {
			foundRoot = true
		}
	}
	assert.True(t, foundProviders, "providers.hcl should be found")
	assert.True(t, foundRoot, "root.hcl should be found")
}

// TestFindConfigFiles_DefaultPattern tests the default pattern behavior
func TestFindConfigFiles_DefaultPattern(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-findconfig-default-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create repositories/terragrunt.hcl (default pattern)
	repoDir := filepath.Join(tmpDir, "project", "org", "repositories")
	err = os.MkdirAll(repoDir, 0755)
	require.NoError(t, err)

	testFile := filepath.Join(repoDir, "terragrunt.hcl")
	err = os.WriteFile(testFile, []byte("# test file"), 0644)
	require.NoError(t, err)

	// Test with no pattern (should use default)
	files, err := findConfigFiles(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, 1, len(files))
	assert.Contains(t, files[0], "repositories/terragrunt.hcl")
}

// TestFindConfigFiles_NoMatches tests when no files match the pattern
func TestFindConfigFiles_NoMatches(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-findconfig-nomatch-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a file that doesn't match
	err = os.WriteFile(filepath.Join(tmpDir, "other.txt"), []byte("# other file"), 0644)
	require.NoError(t, err)

	// Test finding files that don't exist
	files, err := findConfigFiles(tmpDir, "nonexistent.hcl")
	require.NoError(t, err)
	assert.Equal(t, 0, len(files))
}

// TestFindConfigFiles_EmptyDirectory tests with an empty directory
func TestFindConfigFiles_EmptyDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-findconfig-empty-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	files, err := findConfigFiles(tmpDir, "test.hcl")
	require.NoError(t, err)
	assert.Equal(t, 0, len(files))
}

// TestFindConfigFiles_InvalidDirectory tests with an invalid directory
func TestFindConfigFiles_InvalidDirectory(t *testing.T) {
	files, err := findConfigFiles("/nonexistent/directory/path", "test.hcl")
	assert.Error(t, err)
	assert.Nil(t, files)
}

// TestFindOrgsFromFilenames tests extracting org names from file paths
func TestFindOrgsFromFilenames(t *testing.T) {
	testCases := []struct {
		name     string
		files    []string
		expected map[string][]string
	}{
		{
			name: "single org single file",
			files: []string{
				"/path/to/org1/repos/terragrunt.hcl",
			},
			expected: map[string][]string{
				"org1": {"/path/to/org1/repos/terragrunt.hcl"},
			},
		},
		{
			name: "multiple orgs",
			files: []string{
				"/path/to/org1/repos/terragrunt.hcl",
				"/path/to/org2/repos/terragrunt.hcl",
			},
			expected: map[string][]string{
				"org1": {"/path/to/org1/repos/terragrunt.hcl"},
				"org2": {"/path/to/org2/repos/terragrunt.hcl"},
			},
		},
		{
			name: "single org multiple files",
			files: []string{
				"/path/to/org1/repos/terragrunt.hcl",
				"/another/path/org1/repos/other.hcl",
			},
			expected: map[string][]string{
				"org1": {
					"/path/to/org1/repos/terragrunt.hcl",
					"/another/path/org1/repos/other.hcl",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := findOrgsFromFilenames(tc.files)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestFindManagedOrgSlugs_WithProvidersHcl tests org discovery with providers.hcl
func TestFindManagedOrgSlugs_WithProvidersHcl(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-orgs-providers-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create org directory
	orgDir := filepath.Join(tmpDir, "org1")
	err = os.MkdirAll(orgDir, 0755)
	require.NoError(t, err)

	// Create providers.hcl with organization_name
	providersContent := `
locals {
	organization_name = "test-org-1"
}
`
	err = os.WriteFile(filepath.Join(orgDir, "providers.hcl"), []byte(providersContent), 0644)
	require.NoError(t, err)

	orgs, err := FindManagedOrgSlugs(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, 1, len(orgs))
	assert.Contains(t, orgs, "test-org-1")
}

// TestFindManagedOrgSlugs_WithRootHcl tests org discovery with root.hcl
func TestFindManagedOrgSlugs_WithRootHcl(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-orgs-root-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create org directory
	orgDir := filepath.Join(tmpDir, "org2")
	err = os.MkdirAll(orgDir, 0755)
	require.NoError(t, err)

	// Create root.hcl with organization_name
	rootContent := `
locals {
	organization_name = "test-org-2"
}
`
	err = os.WriteFile(filepath.Join(orgDir, "root.hcl"), []byte(rootContent), 0644)
	require.NoError(t, err)

	orgs, err := FindManagedOrgSlugs(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, 1, len(orgs))
	assert.Contains(t, orgs, "test-org-2")
}

// TestFindManagedOrgSlugs_Mixed tests org discovery with both file types
func TestFindManagedOrgSlugs_Mixed(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-orgs-mixed-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create org1 with providers.hcl
	org1Dir := filepath.Join(tmpDir, "org1")
	err = os.MkdirAll(org1Dir, 0755)
	require.NoError(t, err)
	providersContent := `
locals {
	organization_name = "org-from-providers"
}
`
	err = os.WriteFile(filepath.Join(org1Dir, "providers.hcl"), []byte(providersContent), 0644)
	require.NoError(t, err)

	// Create org2 with root.hcl
	org2Dir := filepath.Join(tmpDir, "org2")
	err = os.MkdirAll(org2Dir, 0755)
	require.NoError(t, err)
	rootContent := `
locals {
	organization_name = "org-from-root"
}
`
	err = os.WriteFile(filepath.Join(org2Dir, "root.hcl"), []byte(rootContent), 0644)
	require.NoError(t, err)

	orgs, err := FindManagedOrgSlugs(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, 2, len(orgs))
	assert.Contains(t, orgs, "org-from-providers")
	assert.Contains(t, orgs, "org-from-root")
}

// TestFindManagedOrgSlugs_NoOrgName tests when file has no organization_name
func TestFindManagedOrgSlugs_NoOrgName(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-orgs-noname-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create org directory with file that has no organization_name
	orgDir := filepath.Join(tmpDir, "org1")
	err = os.MkdirAll(orgDir, 0755)
	require.NoError(t, err)

	content := `
locals {
	other_value = "something"
}
`
	err = os.WriteFile(filepath.Join(orgDir, "providers.hcl"), []byte(content), 0644)
	require.NoError(t, err)

	orgs, err := FindManagedOrgSlugs(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, 0, len(orgs))
}

// TestFindManagedOrgSlugs_EmptyDirectory tests with empty directory
func TestFindManagedOrgSlugs_EmptyDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-orgs-empty-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	orgs, err := FindManagedOrgSlugs(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, 0, len(orgs))
}
