package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"ssamai/internal/service"

	"github.com/spf13/cobra"
)

var (
	cfgFile    string
	outputPath string
	verbose    bool
)

// NewRootCmd는 서비스를 주입받아 루트 명령어를 생성합니다
func NewRootCmd(collectSvc *service.CollectService, exportSvc *service.ExportService) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "ssamai",
		Short: "AI CLI 도구들의 작업 내용을 수집하고 마크다운으로 정리하는 도구",
		Long: `ssamai는 Claude Code, Gemini CLI, Amazon Q CLI에서 작업한 내용들을
모두 수집하여 구조화된 마크다운 파일로 저장하는 자동화 도구입니다.

이 도구는 다음 기능을 제공합니다:
- 다중 AI CLI 도구의 세션 데이터 수집
- 구조화된 마크다운 문서 생성
- 데이터 필터링 및 날짜 범위 설정`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.Help()
				return
			}
		},
	}

	cobra.OnInitialize(initConfig)

	// 전역 플래그 정의
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "설정 파일 경로 (기본값: ./configs/config.yaml)")
	rootCmd.PersistentFlags().StringVarP(&outputPath, "output", "o", "./output", "출력 디렉토리 경로")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "상세 출력 모드")

	// 로컬 플래그 정의
	rootCmd.Flags().BoolP("version", "", false, "버전 정보 출력")

	// NewCollectCmd와 NewExportCmd에 의존성 주입
	rootCmd.AddCommand(NewCollectCmd(collectSvc))
	rootCmd.AddCommand(NewExportCmd(exportSvc))
	rootCmd.AddCommand(NewConfigCmd())
	
	return rootCmd
}

// Execute는 root 명령어를 실행합니다 (이전 버전과의 호환성을 위해 유지)
func Execute() {
	// 이 함수는 더 이상 사용되지 않지만 호환성을 위해 유지
	fmt.Fprintf(os.Stderr, "Execute() 함수는 더 이상 사용되지 않습니다. main.go를 확인하세요.\n")
	os.Exit(1)
}

// init 함수는 이제 NewRootCmd에서 처리되므로 제거됩니다

// initConfig는 설정을 초기화합니다
func initConfig() {
	if cfgFile != "" {
		// 사용자가 설정 파일을 지정한 경우
		return
	}

	// 기본 설정 파일 위치 찾기
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "홈 디렉토리를 찾을 수 없습니다: %v\n", err)
		os.Exit(1)
	}

	// 설정 파일 경로들 확인
	configPaths := []string{
		"./configs/config.yaml",
		filepath.Join(home, ".ssamai", "config.yaml"),
		"/etc/ssamai/config.yaml",
	}

	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			cfgFile = path
			break
		}
	}

	if cfgFile == "" {
		cfgFile = "./configs/config.yaml" // 기본값 설정
	}

	// 출력 디렉토리 생성
	if outputPath != "" {
		if err := os.MkdirAll(outputPath, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "출력 디렉토리를 생성할 수 없습니다: %v\n", err)
			os.Exit(1)
		}
	}

	if verbose {
		fmt.Printf("설정 파일: %s\n", cfgFile)
		fmt.Printf("출력 경로: %s\n", outputPath)
	}
}