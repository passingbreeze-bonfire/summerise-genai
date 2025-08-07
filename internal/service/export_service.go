package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"ssamai/internal/interfaces"
	"ssamai/pkg/models"
)

// ExportService는 데이터 내보내기의 비즈니스 로직을 담당하는 서비스입니다.
// ISP 적용: 실제 필요한 인터페이스만 의존
type ExportService struct {
	processor interfaces.DataProcessor
	exporter  interfaces.DataExporter
}

// NewExportService는 새로운 내보내기 서비스를 생성합니다.
func NewExportService(p interfaces.DataProcessor, e interfaces.DataExporter) *ExportService {
	return &ExportService{
		processor: p,
		exporter:  e,
	}
}

// ExportFromFile은 저장된 데이터 파일을 읽어서 내보냅니다.
func (s *ExportService) ExportFromFile(ctx context.Context, inputPath, outputPath string, exportConfig *models.ExportConfig) error {
	// 입력 파일 읽기
	data, err := s.loadCollectedData(inputPath)
	if err != nil {
		return fmt.Errorf("데이터 로드 실패: %w", err)
	}

	// 데이터 처리
	if s.processor != nil {
		processedData, err := s.processor.Process(ctx, data.Sessions)
		if err != nil {
			return fmt.Errorf("데이터 처리 실패: %w", err)
		}

		// 내보내기 설정 업데이트
		if exportConfig.OutputPath == "" {
			exportConfig.OutputPath = outputPath
		}

		// 데이터 내보내기
		if s.exporter != nil {
			return s.exporter.Export(ctx, processedData)
		}
	}

	return fmt.Errorf("processor 또는 exporter가 설정되지 않았습니다")
}

// ExportFromResult는 수집 결과를 직접 내보냅니다.
func (s *ExportService) ExportFromResult(ctx context.Context, result *models.CollectionResult, exportConfig *models.ExportConfig) error {
	// 데이터 처리
	if s.processor != nil {
		processedData, err := s.processor.Process(ctx, result.Sessions)
		if err != nil {
			return fmt.Errorf("데이터 처리 실패: %w", err)
		}

		// 데이터 내보내기
		if s.exporter != nil {
			return s.exporter.Export(ctx, processedData)
		}
	}

	return fmt.Errorf("processor 또는 exporter가 설정되지 않았습니다")
}

// loadCollectedData는 저장된 수집 데이터를 로드합니다.
func (s *ExportService) loadCollectedData(inputPath string) (*models.CollectionResult, error) {
	// 파일 경로 처리
	var filePath string
	
	if inputPath == "" || inputPath == "latest" {
		// 최신 데이터 파일 사용
		dataDir := filepath.Join(".", ".ssamai", "data")
		filePath = filepath.Join(dataDir, "latest.json")
	} else {
		filePath = inputPath
	}

	// 파일 존재 여부 확인
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("데이터 파일이 존재하지 않습니다: %s", filePath)
	}

	// TODO: JSON 파일 읽기 및 파싱 구현
	// 현재는 빈 결과 반환
	return &models.CollectionResult{
		Sessions: make([]models.SessionData, 0),
	}, nil
}

// GetAvailableDataFiles는 사용 가능한 데이터 파일 목록을 반환합니다.
func (s *ExportService) GetAvailableDataFiles() ([]string, error) {
	dataDir := filepath.Join(".", ".ssamai", "data")
	
	// 디렉토리 존재 여부 확인
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	// 디렉토리 내 JSON 파일 목록
	files, err := filepath.Glob(filepath.Join(dataDir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("파일 목록 가져오기 실패: %w", err)
	}

	return files, nil
}

// ValidateExportConfig는 내보내기 설정을 검증합니다.
func (s *ExportService) ValidateExportConfig(config *models.ExportConfig) error {
	if config == nil {
		return fmt.Errorf("내보내기 설정이 없습니다")
	}

	if config.OutputPath == "" {
		return fmt.Errorf("출력 경로가 지정되지 않았습니다")
	}

	// 출력 디렉토리가 존재하는지 확인하고 없으면 생성
	outputDir := filepath.Dir(config.OutputPath)
	if outputDir != "." {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("출력 디렉토리 생성 실패: %w", err)
		}
	}

	return nil
}