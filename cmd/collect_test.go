package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ssamai/internal/config"
	"ssamai/pkg/models"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCollectionConfig(t *testing.T) {
	tests := []struct {
		name           string
		setupFlags     func()
		config         *config.Config
		expectedError  string
		expectedConfig *models.CollectionConfig
	}{
		{
			name: "collect all sources",
			setupFlags: func() {
				collectAll = true
				collectSources = nil
				collectIncludeFiles = true
				collectIncludeCmds = true
			},
			config: &config.Config{
				OutputSettings: config.OutputSettings{
					DefaultTemplate: "comprehensive",
				},
			},
			expectedConfig: &models.CollectionConfig{
				Sources: []models.CollectionSource{
					models.SourceClaudeCode,
					models.SourceGeminiCLI,
					models.SourceAmazonQ,
				},
				IncludeFiles:    true,
				IncludeCommands: true,
				Template:        "comprehensive",
			},
		},
		{
			name: "collect specific sources",
			setupFlags: func() {
				collectAll = false
				collectSources = []string{"claude_code", "gemini_cli"}
				collectIncludeFiles = false
				collectIncludeCmds = false
			},
			config: &config.Config{
				OutputSettings: config.OutputSettings{
					DefaultTemplate: "minimal",
				},
			},
			expectedConfig: &models.CollectionConfig{
				Sources: []models.CollectionSource{
					models.SourceClaudeCode,
					models.SourceGeminiCLI,
				},
				IncludeFiles:    false,
				IncludeCommands: false,
				Template:        "minimal",
			},
		},
		{
			name: "with date range",
			setupFlags: func() {
				collectAll = true
				collectDateFrom = "2024-01-01"
				collectDateTo = "2024-01-31"
				collectIncludeFiles = false
				collectIncludeCmds = false
			},
			config: &config.Config{
				OutputSettings: config.OutputSettings{
					DefaultTemplate: "comprehensive",
				},
			},
			expectedConfig: &models.CollectionConfig{
				Sources: []models.CollectionSource{
					models.SourceClaudeCode,
					models.SourceGeminiCLI,
					models.SourceAmazonQ,
				},
				IncludeFiles:    false,
				IncludeCommands: false,
				Template:        "comprehensive",
				DateRange: &models.DateRange{
					Start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					End:   time.Date(2024, 1, 31, 23, 59, 59, 999999999, time.UTC),
				},
			},
		},
		{
			name: "invalid source name",
			setupFlags: func() {
				collectAll = false
				collectSources = []string{"invalid_source"}
			},
			config:        &config.Config{},
			expectedError: "알 수 없는 데이터 소스: invalid_source",
		},
		{
			name: "no sources specified",
			setupFlags: func() {
				collectAll = false
				collectSources = nil
			},
			config:        &config.Config{},
			expectedError: "--all 또는 --sources 플래그를 지정해야 합니다",
		},
		{
			name: "invalid date format",
			setupFlags: func() {
				collectAll = true
				collectDateFrom = "invalid-date"
			},
			config:        &config.Config{},
			expectedError: "시작 날짜 형식 오류",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global flags
			collectAll = false
			collectSources = nil
			collectDateFrom = ""
			collectDateTo = ""
			collectIncludeFiles = false
			collectIncludeCmds = false

			// Setup test flags
			tt.setupFlags()

			// Execute
			result, err := buildCollectionConfig(tt.config)

			// Verify
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				
				assert.Equal(t, tt.expectedConfig.Sources, result.Sources)
				assert.Equal(t, tt.expectedConfig.IncludeFiles, result.IncludeFiles)
				assert.Equal(t, tt.expectedConfig.IncludeCommands, result.IncludeCommands)
				assert.Equal(t, tt.expectedConfig.Template, result.Template)
				
				if tt.expectedConfig.DateRange != nil {
					require.NotNil(t, result.DateRange)
					// Allow for small time differences due to processing
					assert.WithinDuration(t, tt.expectedConfig.DateRange.Start, result.DateRange.Start, time.Second)
					assert.WithinDuration(t, tt.expectedConfig.DateRange.End, result.DateRange.End, time.Second)
				} else {
					assert.Nil(t, result.DateRange)
				}
			}
		})
	}
}

func TestExecuteCollection(t *testing.T) {
	// Create temporary config file for testing
	tempDir, err := os.MkdirTemp("", "execute_collection_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configContent := `
mcp_settings:
  timeout: 30000
  max_retries: 3
  log_level: "info"
collection_settings:
  claude_code:
    session_dir: "~/.claude/sessions"
  gemini_cli:
    history_file: "~/.gemini/history"
  amazon_q:
    logs_dir: "~/.aws/amazonq/logs"
`
	configPath := filepath.Join(tempDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Set global config file
	oldCfgFile := cfgFile
	defer func() { cfgFile = oldCfgFile }()
	cfgFile = configPath

	tests := []struct {
		name                string
		config              *models.CollectionConfig
		expectedSessionsMin int
		expectedSources     []models.CollectionSource
	}{
		{
			name: "collect from claude code only",
			config: &models.CollectionConfig{
				Sources: []models.CollectionSource{models.SourceClaudeCode},
			},
			expectedSessionsMin: 1,
			expectedSources:     []models.CollectionSource{models.SourceClaudeCode},
		},
		{
			name: "collect from all sources",
			config: &models.CollectionConfig{
				Sources: []models.CollectionSource{
					models.SourceClaudeCode,
					models.SourceGeminiCLI,
					models.SourceAmazonQ,
				},
			},
			expectedSessionsMin: 3, // At least one session per source (fallback data)
			expectedSources: []models.CollectionSource{
				models.SourceClaudeCode,
				models.SourceGeminiCLI,
				models.SourceAmazonQ,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute
			result, err := executeCollection(tt.config)

			// Verify
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.expectedSources, result.Sources)
			assert.GreaterOrEqual(t, len(result.Sessions), tt.expectedSessionsMin)
			assert.Equal(t, len(result.Sessions), result.TotalCount)
			assert.Positive(t, result.Duration)
			assert.False(t, result.CollectedAt.IsZero())
		})
	}
}

func TestCollectFromSource(t *testing.T) {
	// Create temporary config file for testing
	tempDir, err := os.MkdirTemp("", "collect_from_source_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configContent := `
mcp_settings:
  timeout: 30000
  max_retries: 3
  log_level: "info"
collection_settings:
  claude_code:
    session_dir: "~/.claude/sessions"
  gemini_cli:
    history_file: "~/.gemini/history"
  amazon_q:
    logs_dir: "~/.aws/amazonq/logs"
`
	configPath := filepath.Join(tempDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Set global config file
	oldCfgFile := cfgFile
	defer func() { cfgFile = oldCfgFile }()
	cfgFile = configPath

	config := &models.CollectionConfig{
		IncludeFiles:    true,
		IncludeCommands: true,
	}

	tests := []struct {
		name           string
		source         models.CollectionSource
		expectedMinLen int
	}{
		{
			name:           "claude code source",
			source:         models.SourceClaudeCode,
			expectedMinLen: 1,
		},
		{
			name:           "gemini cli source",
			source:         models.SourceGeminiCLI,
			expectedMinLen: 1,
		},
		{
			name:           "amazon q source",
			source:         models.SourceAmazonQ,
			expectedMinLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessions, err := collectFromSource(tt.source, config)

			assert.NoError(t, err)
			assert.GreaterOrEqual(t, len(sessions), tt.expectedMinLen)
			
			// Verify all sessions have correct source
			for _, session := range sessions {
				assert.Equal(t, tt.source, session.Source)
				assert.NotEmpty(t, session.ID)
				assert.False(t, session.Timestamp.IsZero())
			}
		})
	}

	t.Run("invalid source", func(t *testing.T) {
		invalidSource := models.CollectionSource("invalid")
		sessions, err := collectFromSource(invalidSource, config)

		assert.Error(t, err)
		assert.Nil(t, sessions)
		assert.Contains(t, err.Error(), "지원하지 않는 소스")
	})
}

func TestSaveCollectedData(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "collect_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Change to temp directory for testing
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test data
	now := time.Now()
	result := &models.CollectionResult{
		Sessions: []models.SessionData{
			{
				ID:        "test-session",
				Source:    models.SourceClaudeCode,
				Timestamp: now,
				Title:     "Test Session",
				Messages: []models.Message{
					{
						ID:        "msg-1",
						Role:      "user",
						Content:   "Test message",
						Timestamp: now,
					},
				},
			},
		},
		TotalCount:  1,
		Sources:     []models.CollectionSource{models.SourceClaudeCode},
		CollectedAt: now,
		Duration:    time.Second,
	}

	// Execute
	err = saveCollectedData(result)
	assert.NoError(t, err)

	// Verify data directory was created
	dataDir := getDataDirectory()
	_, err = os.Stat(dataDir)
	assert.NoError(t, err)

	// Verify timestamped file was created
	entries, err := os.ReadDir(dataDir)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(entries), 1)

	// Find the collection file
	var collectionFile string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "collection-") && strings.HasSuffix(entry.Name(), ".json") {
			collectionFile = entry.Name()
			break
		}
	}
	assert.NotEmpty(t, collectionFile)

	// Verify file content
	filePath := filepath.Join(dataDir, collectionFile)
	data, err := os.ReadFile(filePath)
	assert.NoError(t, err)

	var savedResult models.CollectionResult
	err = json.Unmarshal(data, &savedResult)
	assert.NoError(t, err)
	assert.Equal(t, result.TotalCount, savedResult.TotalCount)
	assert.Equal(t, result.Sessions[0].ID, savedResult.Sessions[0].ID)

	// Verify latest.json was created
	latestPath := filepath.Join(dataDir, "latest.json")
	_, err = os.Stat(latestPath)
	assert.NoError(t, err)

	// Verify latest.json content
	latestData, err := os.ReadFile(latestPath)
	assert.NoError(t, err)

	var latestResult models.CollectionResult
	err = json.Unmarshal(latestData, &latestResult)
	assert.NoError(t, err)
	assert.Equal(t, result.TotalCount, latestResult.TotalCount)
}

func TestSaveCollectedData_DirectoryCreationFailure(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "collect_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create a file where we expect a directory, to cause mkdir failure
	ssaDirPath := ".ssamai"
	err = os.WriteFile(ssaDirPath, []byte("this is a file, not a directory"), 0644)
	require.NoError(t, err)

	result := &models.CollectionResult{
		Sessions:    []models.SessionData{},
		TotalCount:  0,
		CollectedAt: time.Now(),
	}

	// Execute - should fail due to mkdir error
	err = saveCollectedData(result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "데이터 디렉토리 생성 실패")
}

func TestGetDataDirectory(t *testing.T) {
	dataDir := getDataDirectory()
	expected := filepath.Join(".", ".ssamai", "data")
	assert.Equal(t, expected, dataDir)
}

func TestRunCollect_Integration(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "collect_integration_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test config
	configContent := `
mcp_settings:
  timeout: 30000
  max_retries: 3
  log_level: "info"

output_settings:
  default_template: "comprehensive"
  format_code_blocks: true
  generate_toc: true

collection_settings:
  claude_code:
    session_dir: "~/.claude/sessions"
  gemini_cli:
    history_file: "~/.gemini/history"
  amazon_q:
    logs_dir: "~/.aws/amazonq/logs"
`

	configPath := filepath.Join(tempDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Setup global variables
	cfgFile = configPath
	verbose = true

	// Test successful collection
	t.Run("successful collection all sources", func(t *testing.T) {
		// Reset ALL flags to clean state
		collectAll = false
		collectSources = nil
		collectDateFrom = ""
		collectDateTo = ""
		collectIncludeFiles = false
		collectIncludeCmds = false
		
		// Set flags for this test
		collectAll = true
		collectIncludeFiles = true
		collectIncludeCmds = true
		
		// Create mock command
		cmd := &cobra.Command{}
		
		err := runCollect(cmd, []string{})
		assert.NoError(t, err)

		// Verify data was saved
		dataDir := getDataDirectory()
		entries, err := os.ReadDir(dataDir)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(entries), 1)
	})

	t.Run("collection specific sources", func(t *testing.T) {
		// Reset ALL flags to clean state
		collectAll = false
		collectSources = nil
		collectDateFrom = ""
		collectDateTo = ""
		collectIncludeFiles = false
		collectIncludeCmds = false
		
		// Set flags for this test
		collectAll = false
		collectSources = []string{"claude_code", "gemini_cli"}
		collectIncludeFiles = false
		collectIncludeCmds = false
		
		cmd := &cobra.Command{}
		
		err := runCollect(cmd, []string{})
		assert.NoError(t, err)
	})
}

func TestRunCollect_ConfigLoadFailure(t *testing.T) {
	cfgFile = "/nonexistent/config.yaml"
	
	cmd := &cobra.Command{}
	err := runCollect(cmd, []string{})
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "설정 로드 실패")
}

func TestRunCollect_InvalidFlags(t *testing.T) {
	// Create temporary config
	tempDir, err := os.MkdirTemp("", "collect_flag_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	configContent := `
mcp_settings:
  timeout: 30000
  max_retries: 3
  log_level: "info"
output_settings:
  default_template: "comprehensive"
`

	configPath := filepath.Join(tempDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfgFile = configPath

	t.Run("no sources specified", func(t *testing.T) {
		collectAll = false
		collectSources = nil
		
		cmd := &cobra.Command{}
		err := runCollect(cmd, []string{})
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "수집 설정 구성 실패")
	})

	t.Run("invalid source name", func(t *testing.T) {
		collectAll = false
		collectSources = []string{"invalid_source"}
		
		cmd := &cobra.Command{}
		err := runCollect(cmd, []string{})
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "수집 설정 구성 실패")
	})
}

func TestPrintCollectionResult(t *testing.T) {
	now := time.Now()
	result := &models.CollectionResult{
		Sessions: []models.SessionData{
			{
				ID:        "session-1",
				Source:    models.SourceClaudeCode,
				Title:     "Test Session 1",
				Timestamp: now,
			},
			{
				ID:        "session-2", 
				Source:    models.SourceGeminiCLI,
				Title:     "Test Session 2",
				Timestamp: now.Add(-1 * time.Hour),
			},
		},
		TotalCount: 2,
		Sources:    []models.CollectionSource{models.SourceClaudeCode, models.SourceGeminiCLI},
		CollectedAt: now,
		Duration:   5 * time.Second,
		Errors:     []string{"경고: 일부 데이터 누락", "경고: 권한 부족"},
	}

	// This test mainly verifies that the function doesn't panic
	// In a real scenario, you might want to capture stdout to verify output
	verbose = true
	assert.NotPanics(t, func() {
		printCollectionResult(result)
	})

	verbose = false
	assert.NotPanics(t, func() {
		printCollectionResult(result)
	})
}

// Test helpers
func setupTestEnvironment(t *testing.T) (string, func()) {
	tempDir, err := os.MkdirTemp("", "collect_test")
	require.NoError(t, err)

	oldWd, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	cleanup := func() {
		os.Chdir(oldWd)
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

// Benchmark tests
func BenchmarkBuildCollectionConfig(b *testing.B) {
	config := &config.Config{
		OutputSettings: config.OutputSettings{
			DefaultTemplate: "comprehensive",
		},
	}
	
	// Setup flags
	collectAll = true
	collectIncludeFiles = true
	collectIncludeCmds = true

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := buildCollectionConfig(config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkExecuteCollection(b *testing.B) {
	cfg := &models.CollectionConfig{
		Sources: []models.CollectionSource{models.SourceClaudeCode},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := executeCollection(cfg)
		if err != nil {
			b.Fatal(err)
		}
		if len(result.Sessions) == 0 {
			b.Fatal("No sessions collected")
		}
	}
}