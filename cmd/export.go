package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ssamai/internal/config"
	"ssamai/internal/exporter"
	"ssamai/internal/processor"
	"ssamai/internal/service"
	"ssamai/pkg/models"

	"github.com/spf13/cobra"
)

var (
	exportTemplate    string
	exportNoTOC       bool
	exportNoMeta      bool
	exportNoTimestamp bool
	exportCustomFields map[string]string
	exportDataFile    string
	exportOutputFile  string
)

// NewExportCmd는 서비스 레이어를 주입받아 export 명령어를 생성합니다.
func NewExportCmd(exportSvc *service.ExportService) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "수집된 데이터를 마크다운 파일로 내보냅니다",
		Long: `export 명령어는 collect 명령어로 수집된 데이터를 구조화된
마크다운 문서로 변환하여 저장합니다.

다양한 템플릿과 옵션을 지원하여 필요에 맞는 형태의
문서를 생성할 수 있습니다.`,
		Example: `  # 기본 설정으로 마크다운 내보내기
  ssamai export --output ./summary.md

  # 특정 템플릿 사용하여 내보내기
  ssamai export --template technical --output ./tech-summary.md

  # 메타데이터와 목차 제외하고 내보내기
  ssamai export --no-toc --no-meta --output ./simple.md

  # 사용자 정의 필드 추가하여 내보내기
  ssamai export --custom project=MyProject --custom version=1.0 --output ./project-summary.md

  # 저장된 데이터 파일에서 내보내기
  ssamai export --data ./collected-data.json --output ./from-file.md`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExportWithService(cmd, args, exportSvc)
		},
	}

	// 플래그 정의
	cmd.Flags().StringVar(&exportOutputFile, "output", "", 
		"출력 마크다운 파일 경로 (필수)")
	cmd.Flags().StringVarP(&exportTemplate, "template", "t", "", 
		"사용할 마크다운 템플릿 (기본값: comprehensive)")
	cmd.Flags().BoolVar(&exportNoTOC, "no-toc", false, 
		"목차(Table of Contents) 생성 제외")
	cmd.Flags().BoolVar(&exportNoMeta, "no-meta", false, 
		"메타데이터 정보 제외")
	cmd.Flags().BoolVar(&exportNoTimestamp, "no-timestamp", false, 
		"타임스탬프 정보 제외")
	cmd.Flags().StringToStringVar(&exportCustomFields, "custom", map[string]string{}, 
		"사용자 정의 메타데이터 필드 (key=value 형식)")
	cmd.Flags().StringVarP(&exportDataFile, "data", "d", "", 
		"저장된 데이터 파일에서 읽어서 내보내기")

	// 필수 플래그
	cmd.MarkFlagRequired("output")
	
	return cmd
}

// runExportWithService는 서비스를 사용하여 내보내기를 실행합니다
func runExportWithService(cmd *cobra.Command, args []string, exportSvc *service.ExportService) error {
	if verbose {
		fmt.Println("마크다운 내보내기를 시작합니다...")
	}

	// 설정 로드 (필요시)
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("설정 로드 실패: %w", err)
	}

	// 내보내기 설정 구성
	exportConfig, err := buildExportConfig(cfg)
	if err != nil {
		return fmt.Errorf("내보내기 설정 구성 실패: %w", err)
	}

	if verbose {
		fmt.Printf("내보내기 설정: 템플릿=%s, 출력=%s\n", 
			exportConfig.Template, exportConfig.OutputPath)
	}

	// 서비스의 ExportFromFile 메서드 호출
	err = exportSvc.ExportFromFile(cmd.Context(), exportDataFile, exportOutputFile, exportConfig)
	if err != nil {
		return fmt.Errorf("마크다운 내보내기 실패: %w", err)
	}

	if verbose {
		fmt.Printf("마크다운 파일 생성 완료: %s\n", exportOutputFile)
	}

	return nil
}

func runExport(cmd *cobra.Command, args []string) error {
	if verbose {
		fmt.Println("마크다운 내보내기를 시작합니다...")
	}

	// 설정 로드
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		return fmt.Errorf("설정 로드 실패: %w", err)
	}

	// 내보내기 설정 구성
	exportConfig, err := buildExportConfig(cfg)
	if err != nil {
		return fmt.Errorf("내보내기 설정 구성 실패: %w", err)
	}

	if verbose {
		fmt.Printf("내보내기 설정: 템플릿=%s, 출력=%s\n", 
			exportConfig.Template, exportConfig.OutputPath)
	}

	// 데이터 로드
	var collectionResult *models.CollectionResult
	if exportDataFile != "" {
		// 파일에서 데이터 로드
		collectionResult, err = loadDataFromFile(exportDataFile)
		if err != nil {
			return fmt.Errorf("데이터 파일 로드 실패: %w", err)
		}
	} else {
		// 최신 수집된 데이터 로드 (임시로 더미 데이터 사용)
		collectionResult, err = loadLatestCollectedData()
		if err != nil {
			return fmt.Errorf("최신 수집 데이터 로드 실패: %w", err)
		}
	}

	if len(collectionResult.Sessions) == 0 {
		return fmt.Errorf("내보낼 데이터가 없습니다. 먼저 collect 명령어를 실행하세요")
	}

	// 데이터 처리
	dataProcessor := processor.NewProcessor(exportConfig)
	processedDataInterface, err := dataProcessor.Process(context.Background(), collectionResult.Sessions)
	if err != nil {
		return fmt.Errorf("데이터 처리 실패: %w", err)
	}

	// processedData로 타입 캐스팅
	processedData, ok := processedDataInterface.(processor.ProcessedData)
	if !ok {
		return fmt.Errorf("데이터 처리 결과 타입 변환 실패")
	}

	if verbose {
		fmt.Printf("처리된 데이터: 세션 %d개, 소스 %d개\n",
			len(processedData.Sessions), len(processedData.SourceGroups))
	}


	// 마크다운 내보내기
	markdownExporter := exporter.NewMarkdownExporter(exportConfig)
	if err := markdownExporter.Export(context.Background(), processedData); err != nil {
		return fmt.Errorf("마크다운 내보내기 실패: %w", err)
	}

	// 결과 출력
	printExportResult(exportConfig, collectionResult, &processedData)

	return nil
}

func buildExportConfig(cfg *config.Config) (*models.ExportConfig, error) {
	exportCfg := &models.ExportConfig{
		OutputPath:        exportOutputFile,
		IncludeMetadata:   !exportNoMeta,
		IncludeTimestamps: !exportNoTimestamp,
		FormatCodeBlocks:  cfg.OutputSettings.FormatCodeBlocks,
		GenerateTOC:       cfg.OutputSettings.GenerateTOC && !exportNoTOC,
		CustomFields:      exportCustomFields,
	}

	// 템플릿 설정
	if exportTemplate != "" {
		exportCfg.Template = exportTemplate
	} else {
		exportCfg.Template = cfg.OutputSettings.DefaultTemplate
	}

	// 출력 파일 경로 검증
	if exportCfg.OutputPath == "" {
		return nil, fmt.Errorf("출력 파일 경로가 지정되지 않았습니다")
	}

	// 파일 확장자 확인 및 추가
	if filepath.Ext(exportCfg.OutputPath) == "" {
		exportCfg.OutputPath += ".md"
	}

	return exportCfg, nil
}

func loadDataFromFile(dataFile string) (*models.CollectionResult, error) {
	if verbose {
		fmt.Printf("데이터 파일에서 로드 중: %s\n", dataFile)
	}

	data, err := os.ReadFile(dataFile)
	if err != nil {
		return nil, fmt.Errorf("데이터 파일을 읽을 수 없습니다: %w", err)
	}

	var result models.CollectionResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("데이터 파일 형식이 올바르지 않습니다: %w", err)
	}

	return &result, nil
}

func loadLatestCollectedData() (*models.CollectionResult, error) {
	if verbose {
		fmt.Println("최신 수집 데이터를 로드하는 중...")
	}

	// 데이터 디렉토리 경로
	dataDir := filepath.Join(".", ".ssamai", "data")

	// 1. 먼저 latest.json 파일 확인
	latestPath := filepath.Join(dataDir, "latest.json")
	if _, err := os.Stat(latestPath); err == nil {
		if verbose {
			fmt.Printf("최신 데이터 파일 발견: %s\n", latestPath)
		}
		return loadDataFromFile(latestPath)
	}

	// 2. latest.json이 없으면 가장 최근 파일 찾기
	latestFile, err := findLatestDataFile(dataDir)
	if err == nil && latestFile != "" {
		if verbose {
			fmt.Printf("가장 최신 데이터 파일 발견: %s\n", latestFile)
		}
		return loadDataFromFile(latestFile)
	}

	// 3. 실제 데이터 파일이 없으면 폴백 처리
	if verbose {
		fmt.Println("수집된 데이터 파일이 없습니다. 더미 데이터를 생성합니다.")
		fmt.Println("실제 데이터를 원한다면 먼저 'collect' 명령어를 실행하세요.")
	}

	// 더미 데이터 생성 (기존 로직 유지)
	now := time.Now()
	result := &models.CollectionResult{
		Sessions: []models.SessionData{
			{
				ID:        "claude-session-export-demo",
				Source:    models.SourceClaudeCode,
				Timestamp: now.Add(-2 * time.Hour),
				Title:     "코드 리팩토링 세션 (더미 데이터)",
				Messages: []models.Message{
					{
						ID:        "msg-1",
						Role:      "user",
						Content:   "다음 Go 코드를 리팩토링해주세요:\n\n```go\nfunc main() {\n    fmt.Println(\"Hello World\")\n}\n```",
						Timestamp: now.Add(-2 * time.Hour),
					},
					{
						ID:        "msg-2",
						Role:      "assistant",
						Content:   "코드를 다음과 같이 개선할 수 있습니다:\n\n```go\npackage main\n\nimport \"fmt\"\n\nfunc main() {\n    message := \"Hello World\"\n    fmt.Println(message)\n}\n```\n\n개선사항:\n- 패키지 선언 추가\n- import 명시\n- 변수를 통한 메시지 관리",
						Timestamp: now.Add(-2*time.Hour + 5*time.Minute),
					},
				},
				Commands: []models.Command{
					{
						ID:        "cmd-1",
						Command:   "go",
						Args:      []string{"build", "main.go"},
						Output:    "",
						ExitCode:  0,
						Timestamp: now.Add(-2*time.Hour + 10*time.Minute),
						Duration:  2 * time.Second,
					},
				},
				Files: []models.FileReference{
					{
						Path:        "./main.go",
						Name:        "main.go",
						Size:        156,
						ModTime:     now.Add(-2 * time.Hour),
						ContentType: "text/x-go",
					},
				},
				Metadata: map[string]string{
					"fallback": "true",
					"reason":   "실제 수집 데이터가 없음",
				},
			},
			{
				ID:        "gemini-session-export-demo",
				Source:    models.SourceGeminiCLI,
				Timestamp: now.Add(-1 * time.Hour),
				Title:     "API 설계 검토 (더미 데이터)",
				Messages: []models.Message{
					{
						ID:        "msg-3",
						Role:      "user",
						Content:   "REST API 설계를 검토해주세요. 사용자 인증과 데이터 조회 엔드포인트가 필요합니다.",
						Timestamp: now.Add(-1 * time.Hour),
					},
					{
						ID:        "msg-4",
						Role:      "assistant",
						Content:   "REST API 설계 제안:\n\n**인증 관련**\n- POST /auth/login - 로그인\n- POST /auth/logout - 로그아웃\n- GET /auth/me - 현재 사용자 정보\n\n**데이터 조회**\n- GET /api/v1/users - 사용자 목록\n- GET /api/v1/users/{id} - 특정 사용자\n\n**고려사항**\n- JWT 토큰 기반 인증\n- Rate limiting 적용\n- API 버저닝",
						Timestamp: now.Add(-1*time.Hour + 3*time.Minute),
					},
				},
				Metadata: map[string]string{
					"fallback": "true",
					"reason":   "실제 수집 데이터가 없음",
				},
			},
			{
				ID:        "amazonq-session-export-demo",
				Source:    models.SourceAmazonQ,
				Timestamp: now.Add(-30 * time.Minute),
				Title:     "AWS 인프라 최적화 (더미 데이터)",
				Messages: []models.Message{
					{
						ID:        "msg-5",
						Role:      "user",
						Content:   "현재 ECS 클러스터의 비용을 줄이고 싶습니다. 어떤 방법이 있을까요?",
						Timestamp: now.Add(-30 * time.Minute),
					},
					{
						ID:        "msg-6",
						Role:      "assistant",
						Content:   "ECS 비용 최적화 방안:\n\n1. **Spot Instances 사용**\n   - 최대 90% 비용 절감 가능\n   - 중단 가능한 워크로드에 적합\n\n2. **Auto Scaling 설정**\n   - CPU/메모리 사용률 기반 스케일링\n   - 트래픽 패턴에 맞춘 스케줄링\n\n3. **Reserved Instances**\n   - 장기간 사용 시 최대 75% 할인\n   - 예측 가능한 워크로드에 적합\n\n4. **컨테이너 최적화**\n   - 리소스 요청/제한 적절히 설정\n   - 멀티스테이지 빌드로 이미지 크기 최소화",
						Timestamp: now.Add(-30*time.Minute + 2*time.Minute),
					},
				},
				Metadata: map[string]string{
					"fallback": "true",
					"reason":   "실제 수집 데이터가 없음",
				},
			},
		},
		TotalCount:  3,
		Sources:     []models.CollectionSource{models.SourceClaudeCode, models.SourceGeminiCLI, models.SourceAmazonQ},
		CollectedAt: now,
		Duration:    5 * time.Second,
		Errors:      []string{"실제 수집 데이터가 없어 더미 데이터를 사용합니다."},
	}

	return result, nil
}

// findLatestDataFile은 데이터 디렉토리에서 가장 최신 데이터 파일을 찾습니다
func findLatestDataFile(dataDir string) (string, error) {
	// 디렉토리 존재 확인
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		return "", fmt.Errorf("데이터 디렉토리가 존재하지 않습니다: %s", dataDir)
	}

	// 디렉토리 읽기
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return "", fmt.Errorf("데이터 디렉토리 읽기 실패: %w", err)
	}

	var latestFile string
	var latestTime time.Time

	// collection-*.json 패턴의 파일들 검사
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// latest.json은 제외하고 collection-*.json 패턴만 검사
		if name == "latest.json" || !strings.HasPrefix(name, "collection-") || !strings.HasSuffix(name, ".json") {
			continue
		}

		// 파일 수정시간 확인
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if latestFile == "" || info.ModTime().After(latestTime) {
			latestFile = filepath.Join(dataDir, name)
			latestTime = info.ModTime()
		}
	}

	if latestFile == "" {
		return "", fmt.Errorf("수집 데이터 파일을 찾을 수 없습니다")
	}

	return latestFile, nil
}

func saveDataToFile(result *models.CollectionResult, filename string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON 직렬화 실패: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("파일 저장 실패: %w", err)
	}

	return nil
}

func printExportResult(cfg *models.ExportConfig, collectionResult *models.CollectionResult, processedData *processor.ProcessedData) {
	fmt.Println("\n=== 마크다운 내보내기 완료 ===")
	fmt.Printf("출력 파일: %s\n", cfg.OutputPath)
	fmt.Printf("템플릿: %s\n", cfg.Template)
	fmt.Printf("처리된 세션: %d개\n", len(processedData.Sessions))
	fmt.Printf("소스별 분포:\n")
	
	for source, sessions := range processedData.SourceGroups {
		sourceName := ""
		switch source {
		case models.SourceClaudeCode:
			sourceName = "Claude Code"
		case models.SourceGeminiCLI:
			sourceName = "Gemini CLI"
		case models.SourceAmazonQ:
			sourceName = "Amazon Q"
		}
		fmt.Printf("  - %s: %d개 세션\n", sourceName, len(sessions))
	}

	// 파일 크기 정보
	if info, err := os.Stat(cfg.OutputPath); err == nil {
		fmt.Printf("생성된 파일 크기: %d bytes\n", info.Size())
	}

	fmt.Printf("\n생성된 마크다운 파일을 확인하세요: %s\n", cfg.OutputPath)
	
	// 옵션 정보
	fmt.Println("\n포함된 옵션:")
	if cfg.GenerateTOC {
		fmt.Println("  ✓ 목차 생성")
	} else {
		fmt.Println("  ✗ 목차 제외")
	}
	
	if cfg.IncludeMetadata {
		fmt.Println("  ✓ 메타데이터 포함")
	} else {
		fmt.Println("  ✗ 메타데이터 제외")
	}
	
	if cfg.IncludeTimestamps {
		fmt.Println("  ✓ 타임스탬프 포함")
	} else {
		fmt.Println("  ✗ 타임스탬프 제외")
	}
	
	if len(cfg.CustomFields) > 0 {
		fmt.Printf("  ✓ 사용자 정의 필드: %d개\n", len(cfg.CustomFields))
	}
}