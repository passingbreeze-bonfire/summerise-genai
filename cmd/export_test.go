package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ssamai/internal/config"
	"ssamai/internal/processor"
	"ssamai/pkg/models"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildExportConfig(t *testing.T) {
	tests := []struct {
		name           string
		setupFlags     func()
		config         *config.Config
		expectedError  string
		expectedConfig *models.ExportConfig
	}{
		{
			name: "basic export config",
			setupFlags: func() {
				exportOutputFile = "output.md"
				exportTemplate = "comprehensive"
				exportNoTOC = false
				exportNoMeta = false
				exportNoTimestamp = false
				exportCustomFields = map[string]string{}
			},
			config: &config.Config{
				OutputSettings: config.OutputSettings{
					DefaultTemplate:   "default",
					FormatCodeBlocks:  true,
					GenerateTOC:       true,
				},
			},
			expectedConfig: &models.ExportConfig{
				Template:          "comprehensive",
				OutputPath:        "output.md",
				IncludeMetadata:   true,
				IncludeTimestamps: true,
				FormatCodeBlocks:  true,
				GenerateTOC:       true,
				CustomFields:      map[string]string{},
			},
		},
		{
			name: "with custom fields and exclusions",
			setupFlags: func() {
				exportOutputFile = "custom-output"
				exportTemplate = ""
				exportNoTOC = true
				exportNoMeta = true
				exportNoTimestamp = true
				exportCustomFields = map[string]string{
					"author":  "Test Author",
					"version": "1.0.0",
				}
			},
			config: &config.Config{
				OutputSettings: config.OutputSettings{
					DefaultTemplate:   "minimal",
					FormatCodeBlocks:  false,
					GenerateTOC:       false,
				},
			},
			expectedConfig: &models.ExportConfig{
				Template:          "minimal",
				OutputPath:        "custom-output.md", // .md extension added
				IncludeMetadata:   false,
				IncludeTimestamps: false,
				FormatCodeBlocks:  false,
				GenerateTOC:       false,
				CustomFields: map[string]string{
					"author":  "Test Author",
					"version": "1.0.0",
				},
			},
		},
		{
			name: "file extension already present",
			setupFlags: func() {
				exportOutputFile = "report.markdown"
				exportTemplate = "technical"
				exportNoTOC = false
				exportNoMeta = false
				exportNoTimestamp = false
				exportCustomFields = map[string]string{}
			},
			config: &config.Config{
				OutputSettings: config.OutputSettings{
					DefaultTemplate:   "default",
					FormatCodeBlocks:  true,
					GenerateTOC:       true,
				},
			},
			expectedConfig: &models.ExportConfig{
				Template:          "technical",
				OutputPath:        "report.markdown",
				IncludeMetadata:   true,
				IncludeTimestamps: true,
				FormatCodeBlocks:  true,
				GenerateTOC:       true,
				CustomFields:      map[string]string{},
			},
		},
		{
			name: "missing output file",
			setupFlags: func() {
				exportOutputFile = ""
			},
			config:        &config.Config{},
			expectedError: "출력 파일 경로가 지정되지 않았습니다",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			exportOutputFile = ""
			exportTemplate = ""
			exportNoTOC = false
			exportNoMeta = false
			exportNoTimestamp = false
			exportCustomFields = map[string]string{}

			// Setup test flags
			tt.setupFlags()

			// Execute
			result, err := buildExportConfig(tt.config)

			// Verify
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedConfig.Template, result.Template)
				assert.Equal(t, tt.expectedConfig.OutputPath, result.OutputPath)
				assert.Equal(t, tt.expectedConfig.IncludeMetadata, result.IncludeMetadata)
				assert.Equal(t, tt.expectedConfig.IncludeTimestamps, result.IncludeTimestamps)
				assert.Equal(t, tt.expectedConfig.FormatCodeBlocks, result.FormatCodeBlocks)
				assert.Equal(t, tt.expectedConfig.GenerateTOC, result.GenerateTOC)
				assert.Equal(t, tt.expectedConfig.CustomFields, result.CustomFields)
			}
		})
	}
}

func TestLoadDataFromFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "export_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("valid data file", func(t *testing.T) {
		// Create test data
		now := time.Now()
		originalResult := &models.CollectionResult{
			Sessions: []models.SessionData{
				{
					ID:        "test-session-1",
					Source:    models.SourceClaudeCode,
					Timestamp: now,
					Title:     "Test Session 1",
					Messages: []models.Message{
						{
							ID:        "msg-1",
							Role:      "user",
							Content:   "Hello",
							Timestamp: now,
						},
					},
				},
			},
			TotalCount:  1,
			Sources:     []models.CollectionSource{models.SourceClaudeCode},
			CollectedAt: now,
			Duration:    time.Second * 5,
		}

		// Save test data to file
		dataPath := filepath.Join(tempDir, "test-data.json")
		data, err := json.MarshalIndent(originalResult, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(dataPath, data, 0644)
		require.NoError(t, err)

		// Load and verify
		result, err := loadDataFromFile(dataPath)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, originalResult.TotalCount, result.TotalCount)
		assert.Equal(t, len(originalResult.Sessions), len(result.Sessions))
		assert.Equal(t, originalResult.Sessions[0].ID, result.Sessions[0].ID)
		assert.Equal(t, originalResult.Sessions[0].Source, result.Sessions[0].Source)
	})

	t.Run("file not found", func(t *testing.T) {
		result, err := loadDataFromFile("/nonexistent/file.json")
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "데이터 파일을 읽을 수 없습니다")
	})

	t.Run("invalid json format", func(t *testing.T) {
		invalidPath := filepath.Join(tempDir, "invalid.json")
		err := os.WriteFile(invalidPath, []byte("not valid json"), 0644)
		require.NoError(t, err)

		result, err := loadDataFromFile(invalidPath)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "데이터 파일 형식이 올바르지 않습니다")
	})
}

func TestLoadLatestCollectedData(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "export_latest_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	t.Run("with latest.json file", func(t *testing.T) {
		// Create data directory and latest.json
		dataDir := filepath.Join(".", ".ssamai", "data")
		err := os.MkdirAll(dataDir, 0755)
		require.NoError(t, err)

		now := time.Now()
		testResult := &models.CollectionResult{
			Sessions: []models.SessionData{
				{
					ID:        "latest-session",
					Source:    models.SourceClaudeCode,
					Title:     "Latest Session",
					Timestamp: now,
				},
			},
			TotalCount:  1,
			CollectedAt: now,
		}

		latestPath := filepath.Join(dataDir, "latest.json")
		data, err := json.MarshalIndent(testResult, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(latestPath, data, 0644)
		require.NoError(t, err)

		result, err := loadLatestCollectedData()
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, testResult.Sessions[0].ID, result.Sessions[0].ID)
	})

	t.Run("without latest.json but with collection files", func(t *testing.T) {
		// Clean up any existing latest.json
		dataDir := filepath.Join(".", ".ssamai", "data")
		os.RemoveAll(dataDir)
		err := os.MkdirAll(dataDir, 0755)
		require.NoError(t, err)

		// Create some collection files with different timestamps
		now := time.Now()
		
		// Older file
		olderResult := &models.CollectionResult{
			Sessions: []models.SessionData{
				{ID: "older-session", Source: models.SourceClaudeCode, Title: "Older Session", Timestamp: now.Add(-2 * time.Hour)},
			},
			TotalCount:  1,
			CollectedAt: now.Add(-2 * time.Hour),
		}
		olderData, _ := json.Marshal(olderResult)
		olderPath := filepath.Join(dataDir, "collection-20240101-100000.json")
		os.WriteFile(olderPath, olderData, 0644)
		
		// Set older modification time
		olderTime := now.Add(-2 * time.Hour)
		os.Chtimes(olderPath, olderTime, olderTime)

		// Newer file  
		newerResult := &models.CollectionResult{
			Sessions: []models.SessionData{
				{ID: "newer-session", Source: models.SourceGeminiCLI, Title: "Newer Session", Timestamp: now},
			},
			TotalCount:  1,
			CollectedAt: now,
		}
		newerData, _ := json.Marshal(newerResult)
		newerPath := filepath.Join(dataDir, "collection-20240101-120000.json")
		os.WriteFile(newerPath, newerData, 0644)
		
		// Set newer modification time
		newerTime := now
		os.Chtimes(newerPath, newerTime, newerTime)

		result, err := loadLatestCollectedData()
		assert.NoError(t, err)
		assert.NotNil(t, result)
		
		// Should load the newer file
		assert.Equal(t, "newer-session", result.Sessions[0].ID)
	})

	t.Run("no data files - fallback to dummy data", func(t *testing.T) {
		// Clean up data directory completely
		os.RemoveAll(filepath.Join(".", ".ssamai"))

		result, err := loadLatestCollectedData()
		assert.NoError(t, err)
		assert.NotNil(t, result)
		
		// Should return dummy data
		assert.GreaterOrEqual(t, len(result.Sessions), 3)
		assert.Contains(t, result.Errors, "실제 수집 데이터가 없어 더미 데이터를 사용합니다.")
		
		// Check dummy data has expected fallback flag
		for _, session := range result.Sessions {
			assert.Equal(t, "true", session.Metadata["fallback"])
		}
	})
}

func TestFindLatestDataFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "find_latest_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("directory not exist", func(t *testing.T) {
		nonexistentDir := filepath.Join(tempDir, "nonexistent")
		file, err := findLatestDataFile(nonexistentDir)
		assert.Error(t, err)
		assert.Empty(t, file)
		assert.Contains(t, err.Error(), "데이터 디렉토리가 존재하지 않습니다")
	})

	t.Run("no collection files", func(t *testing.T) {
		emptyDir := filepath.Join(tempDir, "empty")
		err := os.MkdirAll(emptyDir, 0755)
		require.NoError(t, err)

		// Create some non-collection files
		os.WriteFile(filepath.Join(emptyDir, "other.json"), []byte("{}"), 0644)
		os.WriteFile(filepath.Join(emptyDir, "latest.json"), []byte("{}"), 0644)
		os.WriteFile(filepath.Join(emptyDir, "not-json.txt"), []byte("text"), 0644)

		file, err := findLatestDataFile(emptyDir)
		assert.Error(t, err)
		assert.Empty(t, file)
		assert.Contains(t, err.Error(), "수집 데이터 파일을 찾을 수 없습니다")
	})

	t.Run("find latest collection file", func(t *testing.T) {
		dataDir := filepath.Join(tempDir, "data")
		err := os.MkdirAll(dataDir, 0755)
		require.NoError(t, err)

		now := time.Now()

		// Create multiple collection files
		files := []struct {
			name    string
			modTime time.Time
		}{
			{"collection-20240101-100000.json", now.Add(-3 * time.Hour)},
			{"collection-20240101-110000.json", now.Add(-2 * time.Hour)},
			{"collection-20240101-120000.json", now.Add(-1 * time.Hour)}, // This should be latest
			{"latest.json", now},                                           // Should be ignored
			{"other.json", now},                                            // Should be ignored
		}

		for _, f := range files {
			path := filepath.Join(dataDir, f.name)
			os.WriteFile(path, []byte("{}"), 0644)
			os.Chtimes(path, f.modTime, f.modTime)
		}

		latest, err := findLatestDataFile(dataDir)
		assert.NoError(t, err)
		assert.Equal(t, filepath.Join(dataDir, "collection-20240101-120000.json"), latest)
	})

	t.Run("directory read error", func(t *testing.T) {
		// Create directory with restricted permissions
		restrictedDir := filepath.Join(tempDir, "restricted")
		err := os.MkdirAll(restrictedDir, 0000) // No permissions
		require.NoError(t, err)

		file, err := findLatestDataFile(restrictedDir)
		assert.Error(t, err)
		assert.Empty(t, file)
		assert.Contains(t, err.Error(), "데이터 디렉토리 읽기 실패")

		// Restore permissions for cleanup
		os.Chmod(restrictedDir, 0755)
	})
}

func TestSaveDataToFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "save_data_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("successful save", func(t *testing.T) {
		now := time.Now()
		result := &models.CollectionResult{
			Sessions: []models.SessionData{
				{
					ID:        "save-test-session",
					Source:    models.SourceClaudeCode,
					Title:     "Save Test",
					Timestamp: now,
				},
			},
			TotalCount:  1,
			CollectedAt: now,
		}

		filePath := filepath.Join(tempDir, "test-save.json")
		err := saveDataToFile(result, filePath)
		assert.NoError(t, err)

		// Verify file was created and has correct content
		data, err := os.ReadFile(filePath)
		assert.NoError(t, err)

		var loaded models.CollectionResult
		err = json.Unmarshal(data, &loaded)
		assert.NoError(t, err)
		assert.Equal(t, result.Sessions[0].ID, loaded.Sessions[0].ID)
	})

	t.Run("invalid directory path", func(t *testing.T) {
		result := &models.CollectionResult{
			Sessions:    []models.SessionData{},
			TotalCount:  0,
			CollectedAt: time.Now(),
		}

		invalidPath := filepath.Join("/nonexistent/directory", "file.json")
		err := saveDataToFile(result, invalidPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "파일 저장 실패")
	})
}

func TestRunExport_Integration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "export_integration_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create test config
	configContent := `
mcp_settings:
  timeout: 30000
  max_retries: 3
  log_level: "info"

output_settings:
  template_dir: "./templates"
  default_template: "comprehensive"
  format_code_blocks: true
  generate_toc: true
  include_metadata: true
  include_timestamps: true
`

	configPath := filepath.Join(tempDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Set global config
	cfgFile = configPath
	verbose = true

	t.Run("export without collected data - uses fallback", func(t *testing.T) {
		exportOutputFile = "test-output.md"
		exportTemplate = "comprehensive"
		exportDataFile = ""

		cmd := &cobra.Command{}
		err := runExport(cmd, []string{})
		
		// Should succeed with dummy data
		assert.NoError(t, err)
		
		// Verify output file was created
		_, err = os.Stat(exportOutputFile)
		assert.NoError(t, err)
	})

	t.Run("export from specific data file", func(t *testing.T) {
		// Create test data file
		now := time.Now()
		testData := &models.CollectionResult{
			Sessions: []models.SessionData{
				{
					ID:        "export-test-session",
					Source:    models.SourceClaudeCode,
					Title:     "Export Test Session",
					Timestamp: now,
					Messages: []models.Message{
						{ID: "msg-1", Role: "user", Content: "Test message", Timestamp: now},
					},
				},
			},
			TotalCount:  1,
			Sources:     []models.CollectionSource{models.SourceClaudeCode},
			CollectedAt: now,
			Duration:    time.Second,
		}

		dataFilePath := "test-data.json"
		data, err := json.MarshalIndent(testData, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(dataFilePath, data, 0644)
		require.NoError(t, err)

		exportOutputFile = "from-file-output.md"
		exportTemplate = ""
		exportDataFile = dataFilePath

		cmd := &cobra.Command{}
		err = runExport(cmd, []string{})
		assert.NoError(t, err)

		// Verify output file
		_, err = os.Stat(exportOutputFile)
		assert.NoError(t, err)
	})

	t.Run("export with custom fields", func(t *testing.T) {
		exportOutputFile = "custom-output.md"
		exportTemplate = "minimal"
		exportDataFile = ""
		exportCustomFields = map[string]string{
			"project": "Test Project",
			"author":  "Test Author",
		}
		exportNoTOC = true
		exportNoMeta = false

		cmd := &cobra.Command{}
		err := runExport(cmd, []string{})
		assert.NoError(t, err)

		// Verify output file
		_, err = os.Stat(exportOutputFile)
		assert.NoError(t, err)
	})
}

func TestRunExport_ErrorCases(t *testing.T) {
	t.Run("config load failure", func(t *testing.T) {
		cfgFile = "/nonexistent/config.yaml"
		exportOutputFile = "output.md"
		
		cmd := &cobra.Command{}
		err := runExport(cmd, []string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "설정 로드 실패")
	})

	t.Run("missing output file", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "export_error_test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		configContent := `
mcp_settings:
  timeout: 30000
output_settings:
  default_template: "comprehensive"
`
		configPath := filepath.Join(tempDir, "config.yaml")
		os.WriteFile(configPath, []byte(configContent), 0644)

		cfgFile = configPath
		exportOutputFile = ""
		
		cmd := &cobra.Command{}
		err = runExport(cmd, []string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "내보내기 설정 구성 실패")
	})

	t.Run("invalid data file", func(t *testing.T) {
		tempDir2, err := os.MkdirTemp("", "export_error_test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir2)

		configContent := `
output_settings:
  default_template: "comprehensive"
`
		configPath := filepath.Join(tempDir2, "config.yaml")
		os.WriteFile(configPath, []byte(configContent), 0644)

		cfgFile = configPath
		exportOutputFile = "output.md"
		exportDataFile = "/nonexistent/data.json"
		
		cmd := &cobra.Command{}
		err = runExport(cmd, []string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "데이터 파일 로드 실패")
	})
}

func TestPrintExportResult(t *testing.T) {
	// Create temporary output file for file size check
	tempDir, err := os.MkdirTemp("", "print_export_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	outputPath := filepath.Join(tempDir, "test-output.md")
	testContent := "# Test Output\nThis is test markdown content."
	err = os.WriteFile(outputPath, []byte(testContent), 0644)
	require.NoError(t, err)

	cfg := &models.ExportConfig{
		Template:          "comprehensive",
		OutputPath:        outputPath,
		GenerateTOC:       true,
		IncludeMetadata:   true,
		IncludeTimestamps: true,
		CustomFields: map[string]string{
			"author": "Test Author",
		},
	}

	collectionResult := &models.CollectionResult{
		TotalCount: 3,
	}

	// Create mock processed data
	processedData := processor.ProcessedData{
		Sessions: []models.SessionData{
			{Source: models.SourceClaudeCode},
			{Source: models.SourceGeminiCLI},
			{Source: models.SourceAmazonQ},
		},
		SourceGroups: map[models.CollectionSource][]models.SessionData{
			models.SourceClaudeCode: {{Source: models.SourceClaudeCode}},
			models.SourceGeminiCLI:  {{Source: models.SourceGeminiCLI}},
			models.SourceAmazonQ:    {{Source: models.SourceAmazonQ}},
		},
	}

	// This test mainly verifies the function doesn't panic
	// In real scenarios, you might want to capture stdout
	assert.NotPanics(t, func() {
		printExportResult(cfg, collectionResult, &processedData)
	})
}

// Benchmark tests
func BenchmarkBuildExportConfig(b *testing.B) {
	config := &config.Config{
		OutputSettings: config.OutputSettings{
			DefaultTemplate:  "comprehensive",
			FormatCodeBlocks: true,
			GenerateTOC:      true,
		},
	}

	exportOutputFile = "benchmark-output.md"
	exportTemplate = "comprehensive"
	exportCustomFields = map[string]string{
		"test": "value",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := buildExportConfig(config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLoadLatestCollectedData(b *testing.B) {
	// Setup temp directory with test data
	tempDir, err := os.MkdirTemp("", "benchmark_test")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tempDir)

	// Create test data structure
	dataDir := filepath.Join(".", ".ssamai", "data")
	os.MkdirAll(dataDir, 0755)

	testResult := &models.CollectionResult{
		Sessions: []models.SessionData{
			{ID: "benchmark-session", Source: models.SourceClaudeCode, Title: "Benchmark", Timestamp: time.Now()},
		},
		TotalCount:  1,
		CollectedAt: time.Now(),
	}

	data, _ := json.Marshal(testResult)
	os.WriteFile(filepath.Join(dataDir, "latest.json"), data, 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := loadLatestCollectedData()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Test helper functions
func createTestCollectionResult() *models.CollectionResult {
	now := time.Now()
	return &models.CollectionResult{
		Sessions: []models.SessionData{
			{
				ID:        "helper-session-1",
				Source:    models.SourceClaudeCode,
				Title:     "Helper Test Session 1",
				Timestamp: now,
				Messages: []models.Message{
					{
						ID:        "msg-1",
						Role:      "user",
						Content:   "Test message 1",
						Timestamp: now,
					},
				},
			},
			{
				ID:        "helper-session-2",
				Source:    models.SourceGeminiCLI,
				Title:     "Helper Test Session 2", 
				Timestamp: now.Add(-1 * time.Hour),
				Messages: []models.Message{
					{
						ID:        "msg-2",
						Role:      "assistant",
						Content:   "Test response 2",
						Timestamp: now.Add(-1 * time.Hour),
					},
				},
			},
		},
		TotalCount:  2,
		Sources:     []models.CollectionSource{models.SourceClaudeCode, models.SourceGeminiCLI},
		CollectedAt: now,
		Duration:    time.Second * 10,
	}
}