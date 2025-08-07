package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"ssamai/internal/config"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	configShow     bool
	configInit     bool
	configValidate bool
	configPath     string
)

// NewConfigCmd는 설정 관리 명령어를 생성합니다
func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "애플리케이션 설정을 관리합니다",
		Long: `config 명령어는 ssamai의 설정을 관리합니다.

설정 파일 초기화, 유효성 검증, 현재 설정 확인 등의 
기능을 제공합니다.`,
		Example: `  # 현재 설정 표시
  ssamai config --show

  # 설정 파일 초기화
  ssamai config --init

  # 설정 파일 유효성 검증
  ssamai config --validate

  # 특정 경로의 설정 파일 검증
  ssamai config --validate --path ./my-config.yaml`,
		RunE: runConfig,
	}

	// 플래그 정의
	cmd.Flags().BoolVar(&configShow, "show", false,
		"현재 설정을 표시합니다")
	cmd.Flags().BoolVar(&configInit, "init", false,
		"기본 설정 파일을 생성합니다")
	cmd.Flags().BoolVar(&configValidate, "validate", false,
		"설정 파일 유효성을 검증합니다")
	cmd.Flags().StringVar(&configPath, "path", "",
		"설정 파일 경로 (기본값: 자동 탐지)")

	// 플래그 조합 검증
	cmd.MarkFlagsMutuallyExclusive("show", "init")
	cmd.MarkFlagsMutuallyExclusive("show", "validate")
	cmd.MarkFlagsMutuallyExclusive("init", "validate")
	
	return cmd
}

func runConfig(cmd *cobra.Command, args []string) error {
	if configShow {
		return showConfig()
	} else if configInit {
		return initConfigFile()
	} else if configValidate {
		return validateConfig()
	}

	// 기본 동작: 도움말 표시
	return cmd.Help()
}

func showConfig() error {
	path := getConfigPath()

	if verbose {
		fmt.Printf("설정 파일 로드 중: %s\n", path)
	}

	// 설정 로드
	cfg, err := config.LoadConfig(path)
	if err != nil {
		return fmt.Errorf("설정 로드 실패: %w", err)
	}

	fmt.Printf("📋 현재 설정 파일: %s\n\n", path)

	// 수집 설정 표시
	fmt.Println("📊 데이터 수집 설정:")
	fmt.Printf("  - Claude Code: %s\n", cfg.CollectionSettings.ClaudeCode.ConfigDir)
	fmt.Printf("  - Gemini CLI: %s\n", cfg.CollectionSettings.GeminiCLI.ConfigDir)
	fmt.Printf("  - Amazon Q: %s\n", cfg.CollectionSettings.AmazonQ.ConfigDir)
	fmt.Println()

	// 출력 설정 표시
	fmt.Println("📄 출력 설정:")
	fmt.Printf("  - 기본 템플릿: %s\n", cfg.OutputSettings.DefaultTemplate)
	fmt.Printf("  - 메타데이터 포함: %v\n", cfg.OutputSettings.IncludeMetadata)
	fmt.Printf("  - 타임스탬프 포함: %v\n", cfg.OutputSettings.IncludeTimestamps)
	fmt.Printf("  - 코드 블록 포맷팅: %v\n", cfg.OutputSettings.FormatCodeBlocks)
	fmt.Printf("  - 목차 생성: %v\n", cfg.OutputSettings.GenerateTOC)

	return nil
}

func initConfigFile() error {
	path := getConfigPath()

	// 파일 존재 확인
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("⚠️ 설정 파일이 이미 존재합니다: %s\n", path)
		return nil
	}

	// 기본 설정 생성
	cfg := createDefaultConfig()

	// 디렉토리 생성
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("설정 디렉토리 생성 실패: %w", err)
	}

	// YAML로 마샬링
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("설정 마샬링 실패: %w", err)
	}

	// 파일 작성
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("설정 파일 작성 실패: %w", err)
	}

	fmt.Printf("✅ 기본 설정 파일이 생성되었습니다: %s\n", path)
	return nil
}

func validateConfig() error {
	path := getConfigPath()

	if verbose {
		fmt.Printf("설정 파일 검증 중: %s\n", path)
	}

	// 설정 로드 및 검증
	cfg, err := config.LoadConfig(path)
	if err != nil {
		fmt.Printf("❌ 설정 검증 실패: %v\n", err)
		return err
	}

	fmt.Printf("✅ 설정이 유효합니다: %s\n", path)
	
	if verbose {
		fmt.Printf("  - 수집 소스 설정: 3개\n")
		fmt.Printf("  - 출력 템플릿: %s\n", cfg.OutputSettings.DefaultTemplate)
	}

	return nil
}

func getConfigPath() string {
	if configPath != "" {
		return configPath
	}
	return cfgFile
}

func createDefaultConfig() *config.Config {
	return &config.Config{
		CollectionSettings: config.CollectionSettings{
			ClaudeCode: config.CLIToolConfig{
				ConfigDir:       "~/.claude",
				SessionDir:      "~/.claude/sessions",
				HistoryFile:     "~/.claude/history.json",
				IncludePatterns: []string{"*.json", "*.md", "*.log"},
				ExcludePatterns: []string{"*.tmp", "*.cache"},
			},
			GeminiCLI: config.CLIToolConfig{
				ConfigDir:       "~/.config/gemini",
				HistoryFile:     "~/.config/gemini/history.json",
				LogsDir:         "~/.config/gemini/logs",
				IncludePatterns: []string{"*.json", "*.log", "*.yaml"},
				ExcludePatterns: []string{"*.tmp"},
			},
			AmazonQ: config.CLIToolConfig{
				ConfigDir:       "~/.aws/amazonq",
				HistoryFile:     "~/.aws/amazonq/history.json",
				CacheDir:        "~/.aws/amazonq/cache",
				IncludePatterns: []string{"*.json", "*.log"},
				ExcludePatterns: []string{"*.tmp"},
			},
		},
		OutputSettings: config.OutputSettings{
			TemplateDir:       "./templates",
			DefaultTemplate:   "comprehensive",
			IncludeMetadata:   true,
			IncludeTimestamps: true,
			FormatCodeBlocks:  true,
			GenerateTOC:       true,
		},
	}
}