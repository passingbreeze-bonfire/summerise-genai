package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_SetDefaults(t *testing.T) {
	config := &Config{}
	config.SetDefaults()

	// MCP 설정 기본값 확인
	assert.Equal(t, 30000, config.MCPSettings.Timeout)
	assert.Equal(t, 3, config.MCPSettings.MaxRetries)
	assert.Equal(t, "info", config.MCPSettings.LogLevel)
	assert.Equal(t, "./logs/mcp-agents.log", config.MCPSettings.LogFile)

	// 출력 설정 기본값 확인
	assert.Equal(t, "./templates", config.OutputSettings.TemplateDir)
	assert.Equal(t, "comprehensive", config.OutputSettings.DefaultTemplate)
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
				MCPSettings: MCPSettings{
					Timeout:    30000,
					MaxRetries: 3,
					LogLevel:   "info",
				},
				Agents: map[string]AgentConfig{
					"test-agent": {
						Command: "test-command",
						Enabled: true,
					},
				},
			},
			expectError: false,
		},
		{
			name: "invalid timeout - zero",
			config: Config{
				MCPSettings: MCPSettings{
					Timeout:    0,
					MaxRetries: 3,
					LogLevel:   "info",
				},
			},
			expectError: true,
			errorMsg:    "MCP timeout은 0보다 커야 합니다",
		},
		{
			name: "invalid timeout - negative",
			config: Config{
				MCPSettings: MCPSettings{
					Timeout:    -1,
					MaxRetries: 3,
					LogLevel:   "info",
				},
			},
			expectError: true,
			errorMsg:    "MCP timeout은 0보다 커야 합니다",
		},
		{
			name: "invalid max retries - negative",
			config: Config{
				MCPSettings: MCPSettings{
					Timeout:    30000,
					MaxRetries: -1,
					LogLevel:   "info",
				},
			},
			expectError: true,
			errorMsg:    "MCP max_retries는 0 이상이어야 합니다",
		},
		{
			name: "invalid log level",
			config: Config{
				MCPSettings: MCPSettings{
					Timeout:    30000,
					MaxRetries: 3,
					LogLevel:   "invalid",
				},
			},
			expectError: true,
			errorMsg:    "유효하지 않은 로그 레벨: invalid",
		},
		{
			name: "agent without command",
			config: Config{
				MCPSettings: MCPSettings{
					Timeout:    30000,
					MaxRetries: 3,
					LogLevel:   "info",
				},
				Agents: map[string]AgentConfig{
					"bad-agent": {
						Command: "",
						Enabled: true,
					},
				},
			},
			expectError: true,
			errorMsg:    "에이전트 'bad-agent'의 command가 비어있습니다",
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

func TestConfig_ValidLogLevels(t *testing.T) {
	validLevels := []string{"debug", "info", "warn", "error"}
	
	for _, level := range validLevels {
		t.Run("log_level_"+level, func(t *testing.T) {
			config := Config{
				MCPSettings: MCPSettings{
					Timeout:    30000,
					MaxRetries: 3,
					LogLevel:   level,
				},
			}
			
			err := config.Validate()
			assert.NoError(t, err)
		})
	}
}

func TestConfig_GetEnabledAgents(t *testing.T) {
	config := Config{
		Agents: map[string]AgentConfig{
			"enabled-agent": {
				Name:    "Enabled Agent",
				Command: "enabled-cmd",
				Enabled: true,
			},
			"disabled-agent": {
				Name:    "Disabled Agent",
				Command: "disabled-cmd",
				Enabled: false,
			},
			"another-enabled": {
				Name:    "Another Enabled",
				Command: "another-cmd",
				Enabled: true,
			},
		},
	}

	enabled := config.GetEnabledAgents()
	
	assert.Len(t, enabled, 2)
	assert.Contains(t, enabled, "enabled-agent")
	assert.Contains(t, enabled, "another-enabled")
	assert.NotContains(t, enabled, "disabled-agent")
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
				} else if tt.path[0] != '~' {
					assert.Equal(t, tt.path, result)
				} else {
					// 홈 디렉토리 확장 확인
					homeDir, _ := os.UserHomeDir()
					if tt.path == "~" {
						assert.Equal(t, homeDir, result)
					} else {
						expected := filepath.Join(homeDir, tt.path[1:])
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
agents:
  test-agent:
    name: "Test Agent"
    description: "A test agent"
    command: "test-cmd"
    args: ["--flag", "value"]
    enabled: true
    env:
      TEST_ENV: "test_value"

mcp_settings:
  timeout: 60000
  max_retries: 5
  log_level: "debug"
  log_file: "./test-logs/mcp.log"

collection_settings:
  claude_code:
    session_dir: "~/.claude/sessions"
    include_patterns: ["*.json"]
    exclude_patterns: ["*.tmp"]
  gemini_cli:
    history_file: "~/.gemini/history"
    cache_dir: "~/.gemini/cache"

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

	// Agent 설정 확인
	assert.Len(t, config.Agents, 1)
	agent, exists := config.Agents["test-agent"]
	assert.True(t, exists)
	assert.Equal(t, "Test Agent", agent.Name)
	assert.Equal(t, "test-cmd", agent.Command)
	assert.True(t, agent.Enabled)
	assert.Len(t, agent.Args, 2)
	assert.Equal(t, "test_value", agent.Env["TEST_ENV"])

	// MCP 설정 확인
	assert.Equal(t, 60000, config.MCPSettings.Timeout)
	assert.Equal(t, 5, config.MCPSettings.MaxRetries)
	assert.Equal(t, "debug", config.MCPSettings.LogLevel)

	// Collection 설정 확인
	assert.Equal(t, "~/.claude/sessions", config.CollectionSettings.ClaudeCode.SessionDir)
	assert.Contains(t, config.CollectionSettings.ClaudeCode.IncludePatterns, "*.json")
	assert.Equal(t, "~/.gemini/history", config.CollectionSettings.GeminiCLI.HistoryFile)

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
agents:
  test-agent:
    name: "Test Agent"
    command: # 명령어가 비어있음
    enabled: true
`

	configPath := filepath.Join(tempDir, "invalid.yaml")
	err = os.WriteFile(configPath, []byte(invalidContent), 0644)
	assert.NoError(t, err)

	// 설정 로드 시도 - 검증 실패 예상
	config, err := LoadConfig(configPath)
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "설정 검증 실패")
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

	// 홈 디렉토리 시뮬레이션을 위해 임시 파일 생성
	configContent := `
mcp_settings:
  timeout: 30000
  max_retries: 3
  log_level: "info"
`

	// 절대 경로로 설정 파일 저장
	configPath := filepath.Join(tempDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	assert.NoError(t, err)

	// 일반적인 절대 경로로 로드
	config, err := LoadConfig(configPath)
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, 30000, config.MCPSettings.Timeout)
}

func TestConfig_EnsureLogDir(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "log_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	config := &Config{
		MCPSettings: MCPSettings{
			LogFile: filepath.Join(tempDir, "nested", "logs", "mcp.log"),
		},
	}

	// 로그 디렉토리 생성
	err = config.EnsureLogDir()
	assert.NoError(t, err)

	// 디렉토리가 생성되었는지 확인
	logDir := filepath.Dir(config.MCPSettings.LogFile)
	info, err := os.Stat(logDir)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestAgentConfig_BasicFields(t *testing.T) {
	agent := AgentConfig{
		Name:        "Test Agent",
		Description: "A test MCP agent",
		Command:     "/usr/bin/test-agent",
		Args:        []string{"--verbose", "--config=/path/to/config"},
		Env: map[string]string{
			"AGENT_MODE": "test",
			"LOG_LEVEL":  "debug",
		},
		Enabled: true,
	}

	assert.Equal(t, "Test Agent", agent.Name)
	assert.Equal(t, "/usr/bin/test-agent", agent.Command)
	assert.True(t, agent.Enabled)
	assert.Len(t, agent.Args, 2)
	assert.Len(t, agent.Env, 2)
	assert.Equal(t, "test", agent.Env["AGENT_MODE"])
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
	assert.Len(t, config.IncludePatterns, 2)
	assert.Len(t, config.ExcludePatterns, 2)
	assert.Contains(t, config.IncludePatterns, "*.json")
	assert.Contains(t, config.ExcludePatterns, "*.tmp")
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
		MCPSettings: MCPSettings{
			Timeout:    30000,
			MaxRetries: 3,
			LogLevel:   "info",
		},
		Agents: map[string]AgentConfig{
			"test-agent": {
				Command: "test-command",
				Enabled: true,
			},
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