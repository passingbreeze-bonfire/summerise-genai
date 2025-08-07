package main

import (
	"log"

	"ssamai/cmd"
	"ssamai/internal/config"
	"ssamai/internal/exporter"
	"ssamai/internal/processor"
	"ssamai/internal/service"
	"ssamai/pkg/models"

	// Collector 패키지들을 blank import하여 팩토리에 자동 등록
	_ "ssamai/internal/collector"
)

func main() {
	// 1. 설정 로드
	cfg, err := config.LoadConfig("")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// 2. 의존성 객체 생성 (Exporter, Processor 등)
	// OutputSettings를 ExportConfig로 변환
	exportConfig := &models.ExportConfig{
		Template:          cfg.OutputSettings.DefaultTemplate,
		OutputPath:        "", // CLI에서 지정
		IncludeMetadata:   cfg.OutputSettings.IncludeMetadata,
		IncludeTimestamps: cfg.OutputSettings.IncludeTimestamps,
		FormatCodeBlocks:  cfg.OutputSettings.FormatCodeBlocks,
		GenerateTOC:       cfg.OutputSettings.GenerateTOC,
	}
	
	markdownExporter := exporter.NewMarkdownExporter(exportConfig)
	dataProcessor := processor.NewProcessor(exportConfig)

	// 3. 서비스 계층 객체 생성 (ISP 적용: 필요한 인터페이스만 주입)
	collectSvc := service.NewCollectService(
		dataProcessor,        // DataProcessor 인터페이스
		markdownExporter,     // DataExporter 인터페이스
		dataProcessor,        // ProcessorValidator 인터페이스 (같은 객체가 여러 인터페이스 구현)
		markdownExporter,     // ExporterValidator 인터페이스
		cfg)
	exportSvc := service.NewExportService(dataProcessor, markdownExporter)

	// 4. 루트 명령어 생성 및 서비스 주입
	rootCmd := cmd.NewRootCmd(collectSvc, exportSvc)

	// 5. 애플리케이션 실행
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("command execution failed: %v", err)
	}
}