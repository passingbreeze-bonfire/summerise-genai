package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	cfgFile    string
	outputPath string
	verbose    bool
)

// rootCmd는 애플리케이션의 기본 명령어를 나타냅니다
var rootCmd = &cobra.Command{
	Use:   "summerise-genai",
	Short: "AI CLI 도구들의 작업 내용을 수집하고 마크다운으로 정리하는 도구",
	Long: `summerise-genai는 Claude Code, Gemini CLI, Amazon Q CLI에서 작업한 내용들을
모두 수집하여 구조화된 마크다운 파일로 저장하는 자동화 도구입니다.

이 도구는 다음 기능을 제공합니다:
- 다중 AI CLI 도구의 세션 데이터 수집
- Gemini CLI와 협업하여 코드 품질 향상
- 구조화된 마크다운 문서 생성
- MCP 에이전트를 통한 확장 기능`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			return
		}
	},
}

// Execute는 root 명령어를 실행합니다
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "명령어 실행 중 오류가 발생했습니다: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// 전역 플래그 정의
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "설정 파일 경로 (기본값: ./configs/agents.yaml)")
	rootCmd.PersistentFlags().StringVarP(&outputPath, "output", "o", "./output", "출력 디렉토리 경로")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "상세 출력 모드")

	// 로컬 플래그 정의
	rootCmd.Flags().BoolP("version", "", false, "버전 정보 출력")
}

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
		"./configs/agents.yaml",
		filepath.Join(home, ".summerise-genai", "config.yaml"),
		"/etc/summerise-genai/config.yaml",
	}

	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			cfgFile = path
			break
		}
	}

	if cfgFile == "" {
		cfgFile = "./configs/agents.yaml" // 기본값 설정
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