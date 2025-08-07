package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"ssamai/internal/collector"
	"ssamai/internal/config"
	"ssamai/internal/service"
	"ssamai/pkg/models"

	"github.com/spf13/cobra"
)

var (
	collectSources   []string
	collectAll       bool
	collectDateFrom  string
	collectDateTo    string
	collectIncludeFiles bool
	collectIncludeCmds  bool
)

// NewCollectCmd는 서비스 레이어를 주입받아 collect 명령어를 생성합니다.
func NewCollectCmd(collectSvc *service.CollectService) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "collect",
		Short: "AI CLI 도구들의 데이터를 수집합니다",
		Long: `collect 명령어는 Claude Code, Gemini CLI, Amazon Q CLI에서
작업한 세션 데이터, 히스토리, 로그 등을 수집합니다.

수집된 데이터는 구조화된 형태로 저장되어 후에 마크다운으로
내보낼 수 있습니다.`,
		Example: `  # 모든 소스에서 데이터 수집
  ssamai collect --all

  # 특정 소스만 수집
  ssamai collect --sources claude_code,gemini_cli

  # 날짜 범위 지정하여 수집
  ssamai collect --all --from 2024-01-01 --to 2024-01-31

  # 파일과 명령어 정보 포함하여 수집
  ssamai collect --all --include-files --include-commands`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCollectWithService(cmd, args, collectSvc)
		},
	}

	// 플래그 정의
	cmd.Flags().StringSliceVarP(&collectSources, "sources", "s", []string{}, 
		"수집할 데이터 소스 (claude_code, gemini_cli, amazon_q)")
	cmd.Flags().BoolVarP(&collectAll, "all", "a", false, 
		"모든 데이터 소스에서 수집")
	cmd.Flags().StringVar(&collectDateFrom, "from", "", 
		"수집 시작 날짜 (YYYY-MM-DD 형식)")
	cmd.Flags().StringVar(&collectDateTo, "to", "", 
		"수집 종료 날짜 (YYYY-MM-DD 형식)")
	cmd.Flags().BoolVar(&collectIncludeFiles, "include-files", false,
		"파일 참조 정보 포함")
	cmd.Flags().BoolVar(&collectIncludeCmds, "include-commands", false,
		"실행된 명령어 정보 포함")

	// 플래그 검증
	cmd.MarkFlagsMutuallyExclusive("all", "sources")
	
	return cmd
}

// runCollectWithService는 서비스를 사용하여 수집을 실행합니다
func runCollectWithService(cmd *cobra.Command, args []string, collectSvc *service.CollectService) error {
	if verbose {
		fmt.Println("데이터 수집을 시작합니다...")
	}

	// 설정 로드 (필요시)
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("설정 로드 실패: %w", err)
	}

	// 수집 설정 구성
	collectConfig, err := buildCollectionConfig(cfg)
	if err != nil {
		return fmt.Errorf("수집 설정 구성 실패: %w", err)
	}

	if verbose {
		fmt.Printf("수집 설정: %+v\n", collectConfig)
	}

	// 서비스의 Execute 메서드 호출
	result, err := collectSvc.Execute(cmd.Context(), collectConfig)
	if err != nil {
		return fmt.Errorf("데이터 수집 실패: %w", err)
	}

	// 수집된 데이터를 파일로 저장
	if err := saveCollectedData(result); err != nil {
		if verbose {
			fmt.Printf("경고: 데이터 저장 실패 - %v\n", err)
		}
		// 저장 실패는 치명적 오류가 아니므로 계속 진행
	}

	// 결과 출력
	printCollectionResult(result)

	return nil
}

// runCollect는 기존 함수 (호환성 유지)
func runCollect(cmd *cobra.Command, args []string) error {
	if verbose {
		fmt.Println("데이터 수집을 시작합니다...")
	}

	// 설정 로드
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("설정 로드 실패: %w", err)
	}

	// 수집 설정 구성
	collectConfig, err := buildCollectionConfig(cfg)
	if err != nil {
		return fmt.Errorf("수집 설정 구성 실패: %w", err)
	}

	if verbose {
		fmt.Printf("수집 설정: %+v\n", collectConfig)
	}

	// 데이터 수집 실행
	result, err := executeCollection(collectConfig)
	if err != nil {
		return fmt.Errorf("데이터 수집 실패: %w", err)
	}

	// 수집된 데이터를 파일로 저장
	if err := saveCollectedData(result); err != nil {
		if verbose {
			fmt.Printf("경고: 데이터 저장 실패 - %v\n", err)
		}
		// 저장 실패는 치명적 오류가 아니므로 계속 진행
	}

	// 결과 출력
	printCollectionResult(result)

	return nil
}

// saveCollectedData는 수집된 데이터를 파일로 저장합니다
func saveCollectedData(result *models.CollectionResult) error {
	// 데이터 저장 디렉토리 생성
	dataDir := filepath.Join(".", ".ssamai", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("데이터 디렉토리 생성 실패: %w", err)
	}

	// 파일명 생성 (타임스탬프 기반)
	timestamp := result.CollectedAt.Format("20060102-150405")
	filename := fmt.Sprintf("collection-%s.json", timestamp)
	filePath := filepath.Join(dataDir, filename)

	// JSON 데이터 생성
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON 직렬화 실패: %w", err)
	}

	// 파일 저장
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("파일 저장 실패: %w", err)
	}

	if verbose {
		fmt.Printf("수집 데이터 저장 완료: %s\n", filePath)
	}

	// 최신 데이터 심볼릭 링크 또는 파일 생성
	latestPath := filepath.Join(dataDir, "latest.json")
	// 기존 파일이 있으면 삭제
	if _, err := os.Stat(latestPath); err == nil {
		os.Remove(latestPath)
	}
	
	// 최신 데이터 복사 (심볼릭 링크 대신 복사 사용 - 더 안전함)
	if err := os.WriteFile(latestPath, data, 0644); err != nil {
		if verbose {
			fmt.Printf("경고: 최신 데이터 링크 생성 실패 - %v\n", err)
		}
	}

	return nil
}

// getDataDirectory는 데이터 저장 디렉토리 경로를 반환합니다
func getDataDirectory() string {
	return filepath.Join(".", ".ssamai", "data")
}

func buildCollectionConfig(cfg *config.Config) (*models.CollectionConfig, error) {
	collectCfg := &models.CollectionConfig{
		IncludeFiles:    collectIncludeFiles,
		IncludeCommands: collectIncludeCmds,
		OutputPath:      outputPath,
		Template:        cfg.OutputSettings.DefaultTemplate,
	}

	// 소스 결정
	if collectAll {
		collectCfg.Sources = []models.CollectionSource{
			models.SourceClaudeCode,
			models.SourceGeminiCLI,
			models.SourceAmazonQ,
		}
	} else if len(collectSources) > 0 {
		sources := make([]models.CollectionSource, 0, len(collectSources))
		for _, source := range collectSources {
			switch source {
			case "claude_code":
				sources = append(sources, models.SourceClaudeCode)
			case "gemini_cli":
				sources = append(sources, models.SourceGeminiCLI)
			case "amazon_q":
				sources = append(sources, models.SourceAmazonQ)
			default:
				return nil, fmt.Errorf("알 수 없는 데이터 소스: %s", source)
			}
		}
		collectCfg.Sources = sources
	} else {
		return nil, fmt.Errorf("--all 또는 --sources 플래그를 지정해야 합니다")
	}

	// 날짜 범위 설정
	if collectDateFrom != "" || collectDateTo != "" {
		dateRange := &models.DateRange{}
		
		if collectDateFrom != "" {
			from, err := time.Parse("2006-01-02", collectDateFrom)
			if err != nil {
				return nil, fmt.Errorf("시작 날짜 형식 오류: %w", err)
			}
			dateRange.Start = from
		}
		
		if collectDateTo != "" {
			to, err := time.Parse("2006-01-02", collectDateTo)
			if err != nil {
				return nil, fmt.Errorf("종료 날짜 형식 오류: %w", err)
			}
			dateRange.End = to.Add(24 * time.Hour - time.Second) // 해당 날짜의 끝까지
		}
		
		collectCfg.DateRange = dateRange
	}

	return collectCfg, nil
}

func executeCollection(cfg *models.CollectionConfig) (*models.CollectionResult, error) {
	startTime := time.Now()
	result := &models.CollectionResult{
		Sources:     cfg.Sources,
		CollectedAt: startTime,
		Sessions:    make([]models.SessionData, 0),
		Errors:      make([]string, 0),
	}

	if verbose {
		fmt.Printf("수집 대상 소스: %v\n", cfg.Sources)
	}

	// 각 소스별로 데이터 수집
	for _, source := range cfg.Sources {
		if verbose {
			fmt.Printf("소스 '%s'에서 데이터 수집 중...\n", source)
		}

		sessions, err := collectFromSource(source, cfg)
		if err != nil {
			errMsg := fmt.Sprintf("소스 '%s' 수집 실패: %v", source, err)
			result.Errors = append(result.Errors, errMsg)
			log.Printf("경고: %s\n", errMsg)
			continue
		}

		result.Sessions = append(result.Sessions, sessions...)
		if verbose {
			fmt.Printf("소스 '%s'에서 %d개 세션 수집 완료\n", source, len(sessions))
		}
	}

	result.TotalCount = len(result.Sessions)
	result.Duration = time.Since(startTime)

	return result, nil
}

func collectFromSource(source models.CollectionSource, cfg *models.CollectionConfig) ([]models.SessionData, error) {
	// 현재는 더미 데이터를 반환합니다
	// 실제 구현에서는 각 소스별 collector를 호출할 것입니다
	
	switch source {
	case models.SourceClaudeCode:
		return collectClaudeCodeData(cfg)
	case models.SourceGeminiCLI:
		return collectGeminiCLIData(cfg)
	case models.SourceAmazonQ:
		return collectAmazonQData(cfg)
	default:
		return nil, fmt.Errorf("지원하지 않는 소스: %s", source)
	}
}

func collectClaudeCodeData(cfg *models.CollectionConfig) ([]models.SessionData, error) {
	if verbose {
		fmt.Println("  Claude Code 데이터 수집기 호출")
	}
	
	// 설정 로드
	appConfig, err := config.LoadConfig(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("설정 로드 실패: %w", err)
	}
	
	// Claude Code 수집기 생성
	claudeCollector := collector.NewClaudeCodeCollector(appConfig.CollectionSettings.ClaudeCode)
	
	// 실제 데이터 수집
	sessions, err := claudeCollector.Collect(context.Background(), cfg)
	if err != nil {
		// 실제 수집 실패 시 더미 데이터로 폴백
		if verbose {
			fmt.Printf("  실제 수집 실패, 더미 데이터 사용: %v\n", err)
		}
		
		// 더미 데이터 반환
		return []models.SessionData{
			{
				ID:        "claude-session-fallback",
				Source:    models.SourceClaudeCode,
				Timestamp: time.Now().Add(-1 * time.Hour),
				Title:     "Claude Code 예시 세션 (실제 데이터 없음)",
				Messages: []models.Message{
					{
						ID:        "msg-1",
						Role:      "user",
						Content:   "Claude Code가 설치되어 있지 않거나 설정 디렉토리를 찾을 수 없습니다.",
						Timestamp: time.Now().Add(-1 * time.Hour),
					},
				},
				Metadata: map[string]string{
					"fallback": "true",
					"reason":   err.Error(),
				},
			},
		}, nil
	}
	
	if verbose {
		fmt.Printf("  Claude Code에서 %d개 세션 수집 완료\n", len(sessions))
	}
	
	return sessions, nil
}

func collectGeminiCLIData(cfg *models.CollectionConfig) ([]models.SessionData, error) {
	if verbose {
		fmt.Println("  Gemini CLI 데이터 수집기 호출")
	}
	
	// 설정에서 Gemini CLI 설정 가져오기
	appConfig, err := config.LoadConfig(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("설정 로드 실패: %w", err)
	}

	// Gemini CLI collector 생성
	geminiCollector := collector.NewImprovedGeminiCLICollector(appConfig.CollectionSettings.GeminiCLI)
	
	// 실제 데이터 수집
	sessions, err := geminiCollector.Collect(context.Background(), cfg)
	if err != nil {
		if verbose {
			fmt.Printf("  실제 수집 실패, 더미 데이터 사용: %v", err)
		}
		
		// 수집 실패 시 더미 데이터 반환
		return []models.SessionData{
			{
				ID:        "gemini-session-fallback",
				Source:    models.SourceGeminiCLI,
				Timestamp: time.Now().Add(-30 * time.Minute),
				Title:     "Gemini CLI 예시 세션 (실제 데이터 없음)",
				Messages: []models.Message{
					{
						ID:        "msg-2",
						Role:      "user", 
						Content:   "Gemini CLI가 설치되어 있지 않거나 설정 디렉토리를 찾을 수 없습니다.",
						Timestamp: time.Now().Add(-30 * time.Minute),
					},
				},
				Metadata: map[string]string{
					"fallback": "true",
					"reason":   err.Error(),
				},
			},
		}, nil
	}

	if verbose {
		fmt.Printf("  개선된 Gemini CLI에서 %d개 세션 수집 완료\n", len(sessions))
	}

	return sessions, nil
}

func collectAmazonQData(cfg *models.CollectionConfig) ([]models.SessionData, error) {
	if verbose {
		fmt.Println("  Amazon Q CLI 데이터 수집기 호출")
	}
	
	// 설정 로드
	appConfig, err := config.LoadConfig(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("설정 로드 실패: %w", err)
	}
	
	// Amazon Q CLI 수집기 생성
	amazonQCollector := collector.NewAmazonQCollector(appConfig.CollectionSettings.AmazonQ)
	
	// 실제 데이터 수집
	sessions, err := amazonQCollector.Collect(context.Background(), cfg)
	if err != nil {
		// 실제 수집 실패 시 더미 데이터로 폴백
		if verbose {
			fmt.Printf("  실제 수집 실패, 더미 데이터 사용: %v\n", err)
		}
		
		// 더미 데이터 반환
		return []models.SessionData{
			{
				ID:        "amazonq-session-fallback",
				Source:    models.SourceAmazonQ,
				Timestamp: time.Now().Add(-15 * time.Minute),
				Title:     "Amazon Q CLI 예시 세션 (실제 데이터 없음)",
				Messages: []models.Message{
					{
						ID:        "msg-3",
						Role:      "user",
						Content:   "Amazon Q CLI가 설치되어 있지 않거나 설정 디렉토리를 찾을 수 없습니다.",
						Timestamp: time.Now().Add(-15 * time.Minute),
					},
				},
				Metadata: map[string]string{
					"fallback": "true",
					"reason":   err.Error(),
				},
			},
		}, nil
	}
	
	if verbose {
		fmt.Printf("  Amazon Q CLI에서 %d개 세션 수집 완료\n", len(sessions))
	}
	
	return sessions, nil
}

func printCollectionResult(result *models.CollectionResult) {
	fmt.Println("\n=== 데이터 수집 완료 ===")
	fmt.Printf("총 수집된 세션: %d개\n", result.TotalCount)
	fmt.Printf("수집 대상 소스: %v\n", result.Sources)
	fmt.Printf("수집 시간: %v\n", result.Duration.Round(time.Millisecond))
	fmt.Printf("수집 완료 시각: %s\n", result.CollectedAt.Format("2006-01-02 15:04:05"))

	if len(result.Errors) > 0 {
		fmt.Printf("\n경고 (%d개):\n", len(result.Errors))
		for i, err := range result.Errors {
			fmt.Printf("  %d. %s\n", i+1, err)
		}
	}

	if verbose && len(result.Sessions) > 0 {
		fmt.Println("\n수집된 세션 목록:")
		for _, session := range result.Sessions {
			fmt.Printf("  - %s [%s] %s (%s)\n", 
				session.ID, 
				session.Source, 
				session.Title,
				session.Timestamp.Format("01-02 15:04"))
		}
	}

	fmt.Printf("\n다음 단계: export 명령어로 마크다운 파일을 생성하세요\n")
	fmt.Printf("예: summerise-genai export --output ./summary.md\n")
}