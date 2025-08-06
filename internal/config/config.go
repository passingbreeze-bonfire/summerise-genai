package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config는 애플리케이션 전체 설정을 나타냅니다
type Config struct {
	Agents            map[string]AgentConfig   `yaml:"agents"`
	MCPSettings       MCPSettings              `yaml:"mcp_settings"`
	CollectionSettings CollectionSettings      `yaml:"collection_settings"`
	OutputSettings    OutputSettings           `yaml:"output_settings"`
}

// AgentConfig는 MCP 에이전트 설정을 나타냅니다
type AgentConfig struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Command     string            `yaml:"command"`
	Args        []string          `yaml:"args"`
	Env         map[string]string `yaml:"env,omitempty"`
	Enabled     bool              `yaml:"enabled"`
}

// MCPSettings는 MCP 서버 설정을 나타냅니다
type MCPSettings struct {
	Timeout    int    `yaml:"timeout"`
	MaxRetries int    `yaml:"max_retries"`
	LogLevel   string `yaml:"log_level"`
	LogFile    string `yaml:"log_file"`
}

// CollectionSettings는 데이터 수집 설정을 나타냅니다
type CollectionSettings struct {
	ClaudeCode CLIToolConfig `yaml:"claude_code"`
	GeminiCLI  CLIToolConfig `yaml:"gemini_cli"`
	AmazonQ    CLIToolConfig `yaml:"amazon_q"`
}

// CLIToolConfig는 개별 CLI 도구의 설정을 나타냅니다
type CLIToolConfig struct {
	SessionDir      string   `yaml:"session_dir,omitempty"`
	HistoryFile     string   `yaml:"history_file,omitempty"`
	ConfigDir       string   `yaml:"config_dir,omitempty"`
	LogsDir         string   `yaml:"logs_dir,omitempty"`
	CacheDir        string   `yaml:"cache_dir,omitempty"`
	IncludePatterns []string `yaml:"include_patterns"`
	ExcludePatterns []string `yaml:"exclude_patterns"`
}

// OutputSettings는 출력 설정을 나타냅니다
type OutputSettings struct {
	TemplateDir       string `yaml:"template_dir"`
	DefaultTemplate   string `yaml:"default_template"`
	IncludeMetadata   bool   `yaml:"include_metadata"`
	IncludeTimestamps bool   `yaml:"include_timestamps"`
	FormatCodeBlocks  bool   `yaml:"format_code_blocks"`
	GenerateTOC       bool   `yaml:"generate_toc"`
}

// LoadConfig는 설정 파일을 로드합니다
func LoadConfig(configPath string) (*Config, error) {
	// 경로 확장 (~ 처리)
	if configPath[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("홈 디렉토리를 찾을 수 없습니다: %w", err)
		}
		configPath = filepath.Join(home, configPath[1:])
	}

	// 설정 파일 읽기
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("설정 파일을 읽을 수 없습니다 (%s): %w", configPath, err)
	}

	// YAML 파싱
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("설정 파일 파싱 오류: %w", err)
	}

	// 설정 검증
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("설정 검증 실패: %w", err)
	}

	// 기본값 설정
	config.SetDefaults()

	return &config, nil
}

// Validate는 설정의 유효성을 검증합니다
func (c *Config) Validate() error {
	// MCP 설정 검증
	if c.MCPSettings.Timeout <= 0 {
		return fmt.Errorf("MCP timeout은 0보다 커야 합니다")
	}
	
	if c.MCPSettings.MaxRetries < 0 {
		return fmt.Errorf("MCP max_retries는 0 이상이어야 합니다")
	}

	// 에이전트 설정 검증
	for name, agent := range c.Agents {
		if agent.Command == "" {
			return fmt.Errorf("에이전트 '%s'의 command가 비어있습니다", name)
		}
	}

	// 로그 레벨 검증
	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if !validLogLevels[c.MCPSettings.LogLevel] {
		return fmt.Errorf("유효하지 않은 로그 레벨: %s", c.MCPSettings.LogLevel)
	}

	return nil
}

// SetDefaults는 기본값을 설정합니다
func (c *Config) SetDefaults() {
	// MCP 설정 기본값
	if c.MCPSettings.Timeout == 0 {
		c.MCPSettings.Timeout = 30000
	}
	if c.MCPSettings.MaxRetries == 0 {
		c.MCPSettings.MaxRetries = 3
	}
	if c.MCPSettings.LogLevel == "" {
		c.MCPSettings.LogLevel = "info"
	}
	if c.MCPSettings.LogFile == "" {
		c.MCPSettings.LogFile = "./logs/mcp-agents.log"
	}

	// 출력 설정 기본값
	if c.OutputSettings.TemplateDir == "" {
		c.OutputSettings.TemplateDir = "./templates"
	}
	if c.OutputSettings.DefaultTemplate == "" {
		c.OutputSettings.DefaultTemplate = "comprehensive"
	}
}

// GetEnabledAgents는 활성화된 에이전트 목록을 반환합니다
func (c *Config) GetEnabledAgents() map[string]AgentConfig {
	enabled := make(map[string]AgentConfig)
	for name, agent := range c.Agents {
		if agent.Enabled {
			enabled[name] = agent
		}
	}
	return enabled
}

// ExpandPath는 경로의 ~ 기호를 확장합니다
func ExpandPath(path string) (string, error) {
	if path == "" || path[0] != '~' {
		return path, nil
	}
	
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("홈 디렉토리를 찾을 수 없습니다: %w", err)
	}
	
	return filepath.Join(home, path[1:]), nil
}

// EnsureLogDir는 로그 디렉토리를 생성합니다
func (c *Config) EnsureLogDir() error {
	logDir := filepath.Dir(c.MCPSettings.LogFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("로그 디렉토리 생성 실패: %w", err)
	}
	return nil
}