package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"summerise-genai/internal/config"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	configShow   bool
	configInit   bool
	configValidate bool
	configPath   string
)

// configCmd는 설정 관리 명령어를 나타냅니다
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "애플리케이션 설정을 관리합니다",
	Long: `config 명령어는 summerise-genai의 설정을 관리합니다.

설정 파일 초기화, 유효성 검증, 현재 설정 확인 등의 
기능을 제공합니다.`,
	Example: `  # 현재 설정 표시
  summerise-genai config --show

  # 설정 파일 초기화
  summerise-genai config --init

  # 설정 파일 유효성 검증
  summerise-genai config --validate

  # 특정 경로의 설정 파일 검증
  summerise-genai config --validate --path ./my-config.yaml`,
	RunE: runConfig,
}

func init() {
	rootCmd.AddCommand(configCmd)

	// 플래그 정의
	configCmd.Flags().BoolVar(&configShow, "show", false, 
		"현재 설정을 표시합니다")
	configCmd.Flags().BoolVar(&configInit, "init", false, 
		"기본 설정 파일을 생성합니다")
	configCmd.Flags().BoolVar(&configValidate, "validate", false, 
		"설정 파일의 유효성을 검증합니다")
	configCmd.Flags().StringVar(&configPath, "path", "", 
		"설정 파일 경로를 지정합니다")

	// 플래그 검증 - 하나만 선택 가능
	configCmd.MarkFlagsMutuallyExclusive("show", "init", "validate")
}

func runConfig(cmd *cobra.Command, args []string) error {
	// 기본 동작 - 아무 플래그가 없으면 현재 설정 표시
	if !configShow && !configInit && !configValidate {
		configShow = true
	}

	targetPath := cfgFile
	if configPath != "" {
		targetPath = configPath
	}

	if configInit {
		return initializeConfig(targetPath)
	}

	if configValidate {
		return validateConfig(targetPath)
	}

	if configShow {
		return showConfig(targetPath)
	}

	return nil
}

func initializeConfig(path string) error {
	if verbose {
		fmt.Printf("설정 파일을 초기화합니다: %s\n", path)
	}

	// 기본 설정 생성
	defaultConfig := createDefaultConfig()

	// 디렉토리 생성
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("설정 디렉토리 생성 실패: %w", err)
	}

	// 기존 파일 존재 여부 확인
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("경고: 설정 파일이 이미 존재합니다: %s\n", path)
		fmt.Print("덮어쓰시겠습니까? (y/N): ")
		
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("설정 초기화가 취소되었습니다.")
			return nil
		}
	}

	// YAML 파일로 저장
	data, err := yaml.Marshal(defaultConfig)
	if err != nil {
		return fmt.Errorf("설정 직렬화 실패: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("설정 파일 쓰기 실패: %w", err)
	}

	fmt.Printf("✅ 설정 파일이 생성되었습니다: %s\n", path)
	
	if verbose {
		fmt.Println("\n생성된 설정 파일 내용:")
		fmt.Println(string(data))
	}

	return nil
}

func validateConfig(path string) error {
	if verbose {
		fmt.Printf("설정 파일을 검증합니다: %s\n", path)
	}

	// 파일 존재 여부 확인
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("설정 파일이 존재하지 않습니다: %s", path)
	}

	// 설정 로드 및 검증
	cfg, err := config.LoadConfig(path)
	if err != nil {
		return fmt.Errorf("설정 검증 실패: %w", err)
	}

	// 세부 검증 수행
	validationResults := performDetailedValidation(cfg)

	// 결과 출력
	fmt.Printf("✅ 설정 파일 검증 완료: %s\n", path)
	fmt.Printf("📊 검증 결과:\n")
	
	for category, results := range validationResults {
		fmt.Printf("  %s:\n", category)
		for _, result := range results {
			status := "✅"
			if !result.Valid {
				status = "❌"
			}
			fmt.Printf("    %s %s\n", status, result.Message)
		}
		fmt.Println()
	}

	// 경고 또는 오류가 있는지 확인
	hasErrors := false
	for _, results := range validationResults {
		for _, result := range results {
			if !result.Valid {
				hasErrors = true
				break
			}
		}
	}

	if hasErrors {
		return fmt.Errorf("설정 검증 중 오류가 발견되었습니다")
	}

	return nil
}

func showConfig(path string) error {
	if verbose {
		fmt.Printf("설정 파일을 표시합니다: %s\n", path)
	}

	// 설정 로드
	cfg, err := config.LoadConfig(path)
	if err != nil {
		return fmt.Errorf("설정 로드 실패: %w", err)
	}

	fmt.Printf("📋 현재 설정 파일: %s\n\n", path)

	// 에이전트 설정 표시
	fmt.Println("🤖 에이전트 설정:")
	if len(cfg.Agents) == 0 {
		fmt.Println("  설정된 에이전트가 없습니다.")
	} else {
		for name, agent := range cfg.Agents {
			status := "🟢 활성"
			if !agent.Enabled {
				status = "🔴 비활성"
			}
			fmt.Printf("  - %s: %s (%s)\n", name, agent.Name, status)
			fmt.Printf("    명령어: %s %v\n", agent.Command, agent.Args)
		}
	}
	fmt.Println()

	// MCP 설정 표시
	fmt.Println("⚙️ MCP 설정:")
	fmt.Printf("  - 타임아웃: %d ms\n", cfg.MCPSettings.Timeout)
	fmt.Printf("  - 최대 재시도: %d\n", cfg.MCPSettings.MaxRetries)
	fmt.Printf("  - 로그 레벨: %s\n", cfg.MCPSettings.LogLevel)
	fmt.Printf("  - 로그 파일: %s\n", cfg.MCPSettings.LogFile)
	fmt.Println()

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

func createDefaultConfig() *config.Config {
	return &config.Config{
		Agents: map[string]config.AgentConfig{
			"file-system-manager": {
				Name:        "File System Manager",
				Description: "파일 시스템 접근 및 관리 에이전트",
				Command:     "npx",
				Args:        []string{"@modelcontextprotocol/server-filesystem", "--allowed-dirs", "./", "/tmp", "~/"},
				Enabled:     true,
			},
			"markdown-processor": {
				Name:        "Markdown Processor",
				Description: "마크다운 생성 및 포맷팅 에이전트",
				Command:     "npx",
				Args:        []string{"@modelcontextprotocol/server-markdown"},
				Enabled:     true,
			},
		},
		MCPSettings: config.MCPSettings{
			Timeout:    30000,
			MaxRetries: 3,
			LogLevel:   "info",
			LogFile:    "./logs/mcp-agents.log",
		},
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

// ValidationResult는 검증 결과를 나타냅니다
type ValidationResult struct {
	Valid   bool
	Message string
}

func performDetailedValidation(cfg *config.Config) map[string][]ValidationResult {
	results := make(map[string][]ValidationResult)

	// 에이전트 설정 검증
	agentResults := []ValidationResult{}
	for name, agent := range cfg.Agents {
		if agent.Command == "" {
			agentResults = append(agentResults, ValidationResult{
				Valid:   false,
				Message: fmt.Sprintf("에이전트 '%s'의 명령어가 비어있습니다", name),
			})
		} else {
			agentResults = append(agentResults, ValidationResult{
				Valid:   true,
				Message: fmt.Sprintf("에이전트 '%s' 설정이 유효합니다", name),
			})
		}
	}
	results["에이전트"] = agentResults

	// MCP 설정 검증
	mcpResults := []ValidationResult{}
	if cfg.MCPSettings.Timeout <= 0 {
		mcpResults = append(mcpResults, ValidationResult{
			Valid:   false,
			Message: "MCP 타임아웃은 0보다 커야 합니다",
		})
	} else {
		mcpResults = append(mcpResults, ValidationResult{
			Valid:   true,
			Message: "MCP 타임아웃 설정이 유효합니다",
		})
	}

	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[cfg.MCPSettings.LogLevel] {
		mcpResults = append(mcpResults, ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("유효하지 않은 로그 레벨: %s", cfg.MCPSettings.LogLevel),
		})
	} else {
		mcpResults = append(mcpResults, ValidationResult{
			Valid:   true,
			Message: "로그 레벨 설정이 유효합니다",
		})
	}
	results["MCP 설정"] = mcpResults

	// 디렉토리 존재 여부 검증
	dirResults := []ValidationResult{}
	
	// 로그 디렉토리 생성 가능 여부
	logDir := filepath.Dir(cfg.MCPSettings.LogFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		dirResults = append(dirResults, ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("로그 디렉토리 생성 불가: %s", logDir),
		})
	} else {
		dirResults = append(dirResults, ValidationResult{
			Valid:   true,
			Message: fmt.Sprintf("로그 디렉토리 접근 가능: %s", logDir),
		})
	}
	
	results["디렉토리"] = dirResults

	return results
}