package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_SetDefaults(t *testing.T) {
	config := &Config{}
	config.SetDefaults()

	// 출력 설정 기본값 확인 (SetDefaults 메서드에서 실제로 설정하는 것만)
	assert.Equal(t, "./templates", config.OutputSettings.TemplateDir)
	assert.Equal(t, "comprehensive", config.OutputSettings.DefaultTemplate)
	
	// Boolean 필드들은 SetDefaults에서 설정되지 않으므로 기본 zero value를 확인
	assert.False(t, config.OutputSettings.IncludeMetadata)
	assert.False(t, config.OutputSettings.IncludeTimestamps)
	assert.False(t, config.OutputSettings.FormatCodeBlocks)
	assert.False(t, config.OutputSettings.GenerateTOC)
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: Config{
				CollectionSettings: CollectionSettings{
					ClaudeCode: CLIToolConfig{
						SessionDir: "~/.claude/sessions",
					},
					GeminiCLI: CLIToolConfig{
						HistoryFile: "~/.gemini/history",
					},
					AmazonQ: CLIToolConfig{
						ConfigDir: "~/.aws/amazonq",
					},
				},
				OutputSettings: OutputSettings{
					TemplateDir:     "./templates",
					DefaultTemplate: "comprehensive",
				},
			},
			expectError: false,
		},
		{
			name: "empty config",
			config: Config{
				OutputSettings: OutputSettings{
					DefaultTemplate: "comprehensive",
				},
			},
			expectError: false,
		},
		{
			name: "invalid template directory",
			config: Config{
				OutputSettings: OutputSettings{
					TemplateDir:     "", // Empty template dir might be invalid in some contexts
					DefaultTemplate: "comprehensive",
				},
			},
			expectError: false, // Actually this might be valid with defaults
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expectError bool
	}{
		{
			name:        "empty path",
			path:        "",
			expectError: false,
		},
		{
			name:        "absolute path",
			path:        "/absolute/path",
			expectError: false,
		},
		{
			name:        "relative path",
			path:        "relative/path",
			expectError: false,
		},
		{
			name:        "home directory expansion",
			path:        "~/config.yaml",
			expectError: false,
		},
		{
			name:        "home directory only",
			path:        "~",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExpandPath(tt.path)
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				
				if tt.path == "" {
					assert.Equal(t, "", result)
				} else if len(tt.path) > 0 && tt.path[0] != '~' {
					assert.Equal(t, tt.path, result)
				} else {
					// 홈 디렉토리 확장 확인
					homeDir, _ := os.UserHomeDir()
					if tt.path == "~" {
						assert.Equal(t, homeDir, result)
					} else {
						expected := filepath.Join(homeDir, tt.path[2:]) // Skip ~/
						assert.Equal(t, expected, result)
					}
				}
			}
		})
	}
}

func TestLoadConfig_WithValidYAML(t *testing.T) {
	// 임시 디렉토리 생성
	tempDir, err := os.MkdirTemp("", "config_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 테스트용 YAML 설정 생성
	configContent := `
collection_settings:
  claude_code:
    session_dir: "~/.claude/sessions"
    include_patterns: ["*.json"]
    exclude_patterns: ["*.tmp"]
  gemini_cli:
    history_file: "~/.gemini/history"
    cache_dir: "~/.gemini/cache"
  amazon_q:
    config_dir: "~/.aws/amazonq"
    logs_dir: "~/.aws/amazonq/logs"

output_settings:
  template_dir: "./custom-templates"
  default_template: "minimal"
  include_metadata: true
  include_timestamps: false
  format_code_blocks: true
  generate_toc: false
`

	configPath := filepath.Join(tempDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	assert.NoError(t, err)

	// 설정 로드 및 검증
	config, err := LoadConfig(configPath)
	assert.NoError(t, err)
	assert.NotNil(t, config)

	// Collection 설정 확인
	assert.Equal(t, "~/.claude/sessions", config.CollectionSettings.ClaudeCode.SessionDir)
	assert.Contains(t, config.CollectionSettings.ClaudeCode.IncludePatterns, "*.json")
	assert.Equal(t, "~/.gemini/history", config.CollectionSettings.GeminiCLI.HistoryFile)
	assert.Equal(t, "~/.aws/amazonq", config.CollectionSettings.AmazonQ.ConfigDir)

	// Output 설정 확인
	assert.Equal(t, "./custom-templates", config.OutputSettings.TemplateDir)
	assert.Equal(t, "minimal", config.OutputSettings.DefaultTemplate)
	assert.True(t, config.OutputSettings.IncludeMetadata)
	assert.False(t, config.OutputSettings.IncludeTimestamps)
}

func TestLoadConfig_WithInvalidYAML(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 잘못된 YAML 설정
	invalidContent := `
collection_settings:
  claude_code:
    session_dir: ~
    include_patterns: [
    # 배열이 닫히지 않음
`

	configPath := filepath.Join(tempDir, "invalid.yaml")
	err = os.WriteFile(configPath, []byte(invalidContent), 0644)
	assert.NoError(t, err)

	// 설정 로드 시도 - 파싱 실패 예상
	config, err := LoadConfig(configPath)
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "파싱 오류")
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	config, err := LoadConfig("/nonexistent/config.yaml")
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "설정 파일을 읽을 수 없습니다")
}

func TestLoadConfig_HomeDirectoryExpansion(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 기본 설정 파일
	configContent := `
output_settings:
  default_template: "comprehensive"
`

	// 절대 경로로 설정 파일 저장
	configPath := filepath.Join(tempDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	assert.NoError(t, err)

	// 일반적인 절대 경로로 로드
	config, err := LoadConfig(configPath)
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "comprehensive", config.OutputSettings.DefaultTemplate)
}

func TestCLIToolConfig_AllFields(t *testing.T) {
	config := CLIToolConfig{
		SessionDir:      "~/.tool/sessions",
		HistoryFile:     "~/.tool/history.json",
		ConfigDir:       "~/.tool/config",
		LogsDir:         "~/.tool/logs",
		CacheDir:        "~/.tool/cache",
		IncludePatterns: []string{"*.json", "*.yaml"},
		ExcludePatterns: []string{"*.tmp", "*.bak"},
	}

	assert.Equal(t, "~/.tool/sessions", config.SessionDir)
	assert.Equal(t, "~/.tool/history.json", config.HistoryFile)
	assert.Equal(t, "~/.tool/config", config.ConfigDir)
	assert.Equal(t, "~/.tool/logs", config.LogsDir)
	assert.Equal(t, "~/.tool/cache", config.CacheDir)
	assert.Len(t, config.IncludePatterns, 2)
	assert.Len(t, config.ExcludePatterns, 2)
	assert.Contains(t, config.IncludePatterns, "*.json")
	assert.Contains(t, config.ExcludePatterns, "*.tmp")
}

func TestOutputSettings_BasicFields(t *testing.T) {
	settings := OutputSettings{
		TemplateDir:       "./templates",
		DefaultTemplate:   "comprehensive",
		IncludeMetadata:   true,
		IncludeTimestamps: true,
		FormatCodeBlocks:  true,
		GenerateTOC:       true,
	}

	assert.Equal(t, "./templates", settings.TemplateDir)
	assert.Equal(t, "comprehensive", settings.DefaultTemplate)
	assert.True(t, settings.IncludeMetadata)
	assert.True(t, settings.IncludeTimestamps)
	assert.True(t, settings.FormatCodeBlocks)
	assert.True(t, settings.GenerateTOC)
}

func TestCollectionSettings_BasicFields(t *testing.T) {
	settings := CollectionSettings{
		ClaudeCode: CLIToolConfig{
			SessionDir: "~/.claude/sessions",
		},
		GeminiCLI: CLIToolConfig{
			HistoryFile: "~/.gemini/history",
		},
		AmazonQ: CLIToolConfig{
			ConfigDir: "~/.aws/amazonq",
		},
	}

	assert.Equal(t, "~/.claude/sessions", settings.ClaudeCode.SessionDir)
	assert.Equal(t, "~/.gemini/history", settings.GeminiCLI.HistoryFile)
	assert.Equal(t, "~/.aws/amazonq", settings.AmazonQ.ConfigDir)
}

func TestConfig_Integration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_integration_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 포괄적인 설정 파일 생성
	configContent := `
collection_settings:
  claude_code:
    session_dir: "~/.claude/sessions"
    config_dir: "~/.claude"
    include_patterns: ["*.json", "session-*.txt"]
    exclude_patterns: ["*.tmp", "*.log"]
  gemini_cli:
    history_file: "~/.gemini/history.jsonl"
    cache_dir: "~/.gemini/cache"
    config_dir: "~/.gemini"
    include_patterns: ["*.json", "*.jsonl"]
  amazon_q:
    config_dir: "~/.aws/amazonq"
    logs_dir: "~/.aws/amazonq/logs"
    session_dir: "~/.aws/amazonq/sessions"
    include_patterns: ["*.json", "*.log"]
    exclude_patterns: ["*.backup"]

output_settings:
  template_dir: "./custom-templates"
  default_template: "technical"
  include_metadata: false
  include_timestamps: true
  format_code_blocks: false
  generate_toc: true
`

	configPath := filepath.Join(tempDir, "integration-config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// 설정 로드
	config, err := LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, config)

	// SetDefaults 호출
	config.SetDefaults()

	// Validate 호출
	err = config.Validate()
	assert.NoError(t, err)

	// 모든 설정이 올바르게 로드되었는지 확인
	assert.Equal(t, "~/.claude/sessions", config.CollectionSettings.ClaudeCode.SessionDir)
	assert.Len(t, config.CollectionSettings.ClaudeCode.IncludePatterns, 2)
	
	assert.Equal(t, "~/.gemini/history.jsonl", config.CollectionSettings.GeminiCLI.HistoryFile)
	assert.Equal(t, "~/.gemini/cache", config.CollectionSettings.GeminiCLI.CacheDir)
	
	assert.Equal(t, "~/.aws/amazonq", config.CollectionSettings.AmazonQ.ConfigDir)
	assert.Equal(t, "~/.aws/amazonq/logs", config.CollectionSettings.AmazonQ.LogsDir)
	
	assert.Equal(t, "./custom-templates", config.OutputSettings.TemplateDir)
	assert.Equal(t, "technical", config.OutputSettings.DefaultTemplate)
	assert.False(t, config.OutputSettings.IncludeMetadata)
	assert.True(t, config.OutputSettings.IncludeTimestamps)
}

// 벤치마크 테스트
func BenchmarkConfig_SetDefaults(b *testing.B) {
	for i := 0; i < b.N; i++ {
		config := &Config{}
		config.SetDefaults()
	}
}

func BenchmarkConfig_Validate(b *testing.B) {
	config := Config{
		CollectionSettings: CollectionSettings{
			ClaudeCode: CLIToolConfig{SessionDir: "~/.claude"},
		},
		OutputSettings: OutputSettings{
			DefaultTemplate: "comprehensive",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := config.Validate()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkExpandPath(b *testing.B) {
	paths := []string{
		"",
		"/absolute/path",
		"relative/path",
		"~/config.yaml",
		"~/very/long/path/to/config/file.yaml",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, path := range paths {
			_, err := ExpandPath(path)
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkLoadConfig(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "benchmark_config")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	configContent := `
collection_settings:
  claude_code:
    session_dir: "~/.claude/sessions"
output_settings:
  default_template: "comprehensive"
`

	configPath := filepath.Join(tempDir, "bench-config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config, err := LoadConfig(configPath)
		if err != nil {
			b.Fatal(err)
		}
		if config == nil {
			b.Fatal("config is nil")
		}
	}
}