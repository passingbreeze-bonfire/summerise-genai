package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config는 애플리케이션 전체 설정을 나타냅니다
type Config struct {
	CollectionSettings CollectionSettings `yaml:"collection_settings"`
	OutputSettings     OutputSettings     `yaml:"output_settings"`
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
	// 빈 경로일 경우 기본 설정 반환
	if configPath == "" {
		config := createDefaultConfig()
		config.SetDefaults()
		return config, nil
	}
	
	// 경로 확장 (~ 처리)
	if len(configPath) > 0 && configPath[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("홈 디렉토리를 찾을 수 없습니다: %w", err)
		}
		configPath = filepath.Join(home, configPath[1:])
	}

	// 설정 파일 읽기
	data, err := os.ReadFile(configPath)
	if err != nil {
		// 파일이 없으면 기본 설정 반환
		if os.IsNotExist(err) {
			config := createDefaultConfig()
			config.SetDefaults()
			return config, nil
		}
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
	// 기본 검증 로직 추가 가능
	return nil
}

// createDefaultConfig는 기본 설정을 생성합니다
func createDefaultConfig() *Config {
	return &Config{
		CollectionSettings: CollectionSettings{
			ClaudeCode: CLIToolConfig{
				ConfigDir:       "~/.claude",
				SessionDir:      "~/.claude/sessions",
				HistoryFile:     "~/.claude/history.json",
				IncludePatterns: []string{"*.json", "*.md", "*.log"},
				ExcludePatterns: []string{"*.tmp", "*.cache"},
			},
			GeminiCLI: CLIToolConfig{
				ConfigDir:       "~/.config/gemini",
				HistoryFile:     "~/.config/gemini/history.json",
				LogsDir:         "~/.config/gemini/logs",
				IncludePatterns: []string{"*.json", "*.log", "*.yaml"},
				ExcludePatterns: []string{"*.tmp"},
			},
			AmazonQ: CLIToolConfig{
				ConfigDir:       "~/.aws/amazonq",
				HistoryFile:     "~/.aws/amazonq/history.json",
				CacheDir:        "~/.aws/amazonq/cache",
				IncludePatterns: []string{"*.json", "*.log"},
				ExcludePatterns: []string{"*.tmp"},
			},
		},
		OutputSettings: OutputSettings{
			TemplateDir:       "./templates",
			DefaultTemplate:   "comprehensive",
			IncludeMetadata:   true,
			IncludeTimestamps: true,
			FormatCodeBlocks:  true,
			GenerateTOC:       true,
		},
	}
}

// SetDefaults는 기본값을 설정합니다
func (c *Config) SetDefaults() {
	// 출력 설정 기본값
	if c.OutputSettings.TemplateDir == "" {
		c.OutputSettings.TemplateDir = "./templates"
	}
	if c.OutputSettings.DefaultTemplate == "" {
		c.OutputSettings.DefaultTemplate = "comprehensive"
	}
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
