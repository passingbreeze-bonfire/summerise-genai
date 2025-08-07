package service

import (
	"context"
	"fmt"
	"time"

	"ssamai/internal/collector"
	"ssamai/internal/config"
	"ssamai/internal/interfaces"
	"ssamai/pkg/models"
)

// CollectService는 데이터 수집의 전체 비즈니스 로직을 담당하는 서비스입니다.
// ISP 적용: 실제 필요한 인터페이스만 의존
type CollectService struct {
	processor interfaces.DataProcessor
	exporter  interfaces.DataExporter
	// 검증용 인터페이스들 (ISP: 검증이 필요한 경우에만 사용)
	processorValidator interfaces.ProcessorValidator
	exporterValidator  interfaces.ExporterValidator
	// config는 collector factory에서 필요하므로 구체 타입을 사용 (일부 DIP 완화)
	config    *config.Config
}

// NewCollectService는 새로운 수집 서비스를 생성합니다.
// ISP 적용: 필요한 기능별로 인터페이스를 분리하여 주입받음
func NewCollectService(
	p interfaces.DataProcessor, 
	e interfaces.DataExporter, 
	pv interfaces.ProcessorValidator,
	ev interfaces.ExporterValidator,
	cfg *config.Config) *CollectService {
	return &CollectService{
		processor:          p,
		exporter:           e,
		processorValidator: pv,
		exporterValidator:  ev,
		config:             cfg,
	}
}

// Execute는 데이터 수집 과정을 조율합니다. (SRP 적용: 조율 책임만 담당)
func (s *CollectService) Execute(ctx context.Context, collectConfig *models.CollectionConfig) (*models.CollectionResult, error) {
	// 1. 결과 초기화 (SRP: 초기화 책임 분리)
	result := s.initializeCollectionResult(collectConfig)
	
	// 2. 설정 준비 (SRP: 설정 관리 책임 분리)
	collectorConfigs, err := s.prepareCollectorConfigs()
	if err != nil {
		return nil, fmt.Errorf("설정 준비 실패: %w", err)
	}
	
	// 3. 데이터 수집 실행 (SRP: 수집 조율 책임 분리)
	err = s.executeCollection(ctx, collectConfig, collectorConfigs, result)
	if err != nil {
		return nil, fmt.Errorf("데이터 수집 실행 실패: %w", err)
	}
	
	// 4. 결과 완성 (SRP: 결과 완성 책임 분리)
	s.finalizeCollectionResult(result)
	
	return result, nil
}

// initializeCollectionResult는 수집 결과를 초기화합니다. (SRP: 초기화 전용)
func (s *CollectService) initializeCollectionResult(collectConfig *models.CollectionConfig) *models.CollectionResult {
	return &models.CollectionResult{
		Sources:     collectConfig.Sources,
		CollectedAt: time.Now(),
		Sessions:    make([]models.SessionData, 0),
		Errors:      make([]string, 0),
	}
}

// prepareCollectorConfigs는 컬렉터 설정을 준비합니다. (SRP: 설정 준비 전용)
func (s *CollectService) prepareCollectorConfigs() (map[models.CollectionSource]interface{}, error) {
	return s.getCollectorConfigs()
}

// executeCollection은 실제 데이터 수집을 실행합니다. (SRP: 수집 실행 전용)
func (s *CollectService) executeCollection(
	ctx context.Context, 
	collectConfig *models.CollectionConfig,
	collectorConfigs map[models.CollectionSource]interface{},
	result *models.CollectionResult) error {
	
	for _, source := range collectConfig.Sources {
		// Context 취소 확인 (SRP: 취소 처리 책임)
		if err := s.checkContextCancellation(ctx); err != nil {
			return err
		}

		// 소스별 수집 및 에러 처리 (SRP: 수집과 에러 처리 책임 분리)
		sessions, err := s.collectFromSource(ctx, source, collectConfig, collectorConfigs)
		s.handleCollectionResult(source, sessions, err, result)
	}
	
	return nil
}

// checkContextCancellation은 컨텍스트 취소를 확인합니다. (SRP: 취소 확인 전용)
func (s *CollectService) checkContextCancellation(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// handleCollectionResult는 수집 결과를 처리합니다. (SRP: 결과 처리 전용)
func (s *CollectService) handleCollectionResult(
	source models.CollectionSource,
	sessions []models.SessionData, 
	err error, 
	result *models.CollectionResult) {
	
	if err != nil {
		errMsg := fmt.Sprintf("소스 '%s' 수집 실패: %v", source, err)
		result.Errors = append(result.Errors, errMsg)
		return
	}
	
	result.Sessions = append(result.Sessions, sessions...)
}

// finalizeCollectionResult는 수집 결과를 완성합니다. (SRP: 결과 완성 전용)
func (s *CollectService) finalizeCollectionResult(result *models.CollectionResult) {
	result.TotalCount = len(result.Sessions)
	result.Duration = time.Since(result.CollectedAt)
}

// collectFromSource는 특정 소스에서 데이터를 수집합니다.
func (s *CollectService) collectFromSource(ctx context.Context, source models.CollectionSource, collectConfig *models.CollectionConfig, configs map[models.CollectionSource]interface{}) ([]models.SessionData, error) {
	// 팩토리를 통해 Collector 가져오기
	collectorConfig, exists := configs[source]
	if !exists {
		return nil, fmt.Errorf("소스 '%s'에 대한 설정이 없습니다", source)
	}

	c, err := collector.GetCollector(source, collectorConfig)
	if err != nil {
		return nil, fmt.Errorf("collector 생성 실패: %w", err)
	}

	// 데이터 수집
	sessions, err := c.Collect(ctx, collectConfig)
	if err != nil {
		return nil, fmt.Errorf("데이터 수집 실패: %w", err)
	}

	return sessions, nil
}

// ProcessAndExport는 수집된 데이터를 처리하고 내보냅니다.
func (s *CollectService) ProcessAndExport(ctx context.Context, result *models.CollectionResult, exportConfig *models.ExportConfig) error {
	// 데이터 처리
	if s.processor != nil {
		processedData, err := s.processor.Process(ctx, result.Sessions)
		if err != nil {
			return fmt.Errorf("데이터 처리 실패: %w", err)
		}

		// 처리된 데이터를 내보내기
		if s.exporter != nil {
			return s.exporter.Export(ctx, processedData)
		}
	}

	return nil
}

// ValidateConfig는 서비스 설정을 검증합니다.
func (s *CollectService) ValidateConfig() error {
	if s.config == nil {
		return fmt.Errorf("설정이 없습니다")
	}
	
	// ISP 적용: 검증 전용 인터페이스 사용
	if err := s.processorValidator.Validate(); err != nil {
		return fmt.Errorf("프로세서 검증 실패: %w", err)
	}
	
	if err := s.exporterValidator.Validate(); err != nil {
		return fmt.Errorf("익스포터 검증 실패: %w", err)
	}
	
	return nil
}

// getCollectorConfigs는 설정에서 컬렉터 설정을 추출합니다.
func (s *CollectService) getCollectorConfigs() (map[models.CollectionSource]interface{}, error) {
	if s.config == nil {
		return nil, fmt.Errorf("설정이 없습니다")
	}
	
	return map[models.CollectionSource]interface{}{
		models.SourceClaudeCode: s.config.CollectionSettings.ClaudeCode,
		models.SourceGeminiCLI:  s.config.CollectionSettings.GeminiCLI,
		models.SourceAmazonQ:    s.config.CollectionSettings.AmazonQ,
	}, nil
}

// GetSupportedSources는 지원하는 모든 소스를 반환합니다.
func (s *CollectService) GetSupportedSources() []models.CollectionSource {
	return collector.ListRegisteredSources()
}