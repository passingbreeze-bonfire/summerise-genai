package interfaces

import (
	"context"

	"ssamai/pkg/models"
)

// DataProcessor는 데이터 처리 핵심 기능을 담당하는 인터페이스입니다 (ISP 적용)
type DataProcessor interface {
	// Process는 세션 데이터를 처리하여 구조화된 형태로 변환합니다
	Process(ctx context.Context, sessions []models.SessionData) (interface{}, error)
}

// ProcessorInfo는 처리기 메타데이터 조회를 담당하는 인터페이스입니다 (ISP 적용)
type ProcessorInfo interface {
	// GetSupportedOutputFormats는 지원하는 출력 형식들을 반환합니다
	GetSupportedOutputFormats() []string
}

// ProcessorValidator는 처리기 설정 검증을 담당하는 인터페이스입니다 (ISP 적용)
type ProcessorValidator interface {
	// Validate는 처리기 설정이 유효한지 검증합니다
	Validate() error
}

// FullDataProcessor는 모든 처리기 기능을 결합한 인터페이스입니다 (편의용)
type FullDataProcessor interface {
	DataProcessor
	ProcessorInfo
	ProcessorValidator
}

// DataExporter는 데이터 내보내기 핵심 기능을 담당하는 인터페이스입니다 (ISP 적용)
type DataExporter interface {
	// Export는 처리된 데이터를 내보냅니다
	Export(ctx context.Context, data interface{}) error
}

// ExporterInfo는 내보내기 메타데이터 조회를 담당하는 인터페이스입니다 (ISP 적용)
type ExporterInfo interface {
	// GetFormat은 내보내기 형식을 반환합니다
	GetFormat() string
	
	// GetSupportedTemplates는 지원하는 템플릿들을 반환합니다
	GetSupportedTemplates() []string
}

// ExporterValidator는 내보내기 설정 검증을 담당하는 인터페이스입니다 (ISP 적용)
type ExporterValidator interface {
	// Validate는 내보내기 설정이 유효한지 검증합니다
	Validate() error
}

// FullDataExporter는 모든 내보내기 기능을 결합한 인터페이스입니다 (편의용)
type FullDataExporter interface {
	DataExporter
	ExporterInfo
	ExporterValidator
}