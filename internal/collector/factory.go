package collector

import (
	"context"
	"fmt"
	"time"

	"ssamai/pkg/models"
)

// CollectorConstructor는 Collector를 생성하는 함수 타입입니다.
type CollectorConstructor func(config interface{}) models.Collector

var registry = make(map[models.CollectionSource]CollectorConstructor)

// Register는 새로운 Collector 생성자를 팩토리에 등록합니다.
func Register(source models.CollectionSource, constructor CollectorConstructor) {
	registry[source] = constructor
}

// GetCollector는 소스에 맞는 Collector 인스턴스를 반환합니다.
func GetCollector(source models.CollectionSource, config interface{}) (models.Collector, error) {
	constructor, ok := registry[source]
	if !ok {
		return nil, fmt.Errorf("no collector registered for source: %s", source)
	}
	return constructor(config), nil
}

// ListRegisteredSources는 등록된 모든 소스들을 반환합니다.
func ListRegisteredSources() []models.CollectionSource {
	sources := make([]models.CollectionSource, 0, len(registry))
	for source := range registry {
		sources = append(sources, source)
	}
	return sources
}

// IsRegistered는 특정 소스가 등록되어 있는지 확인합니다.
func IsRegistered(source models.CollectionSource) bool {
	_, ok := registry[source]
	return ok
}

// CollectAllSources는 등록된 모든 collector에서 데이터를 수집합니다.
func CollectAllSources(ctx context.Context, collectionConfig *models.CollectionConfig, configs map[models.CollectionSource]interface{}) (*models.CollectionResult, error) {
	result := &models.CollectionResult{
		Sources:     make([]models.CollectionSource, 0),
		Sessions:    make([]models.SessionData, 0),
		Errors:      make([]string, 0),
		CollectedAt: time.Now(),
	}

	for _, source := range collectionConfig.Sources {
		config, exists := configs[source]
		if !exists {
			errMsg := fmt.Sprintf("소스 '%s'에 대한 설정이 없습니다", source)
			result.Errors = append(result.Errors, errMsg)
			continue
		}

		collector, err := GetCollector(source, config)
		if err != nil {
			errMsg := fmt.Sprintf("소스 '%s'의 collector 생성 실패: %v", source, err)
			result.Errors = append(result.Errors, errMsg)
			continue
		}

		sessions, err := collector.Collect(ctx, collectionConfig)
		if err != nil {
			errMsg := fmt.Sprintf("소스 '%s'에서 데이터 수집 실패: %v", source, err)
			result.Errors = append(result.Errors, errMsg)
			continue
		}

		result.Sessions = append(result.Sessions, sessions...)
		result.Sources = append(result.Sources, source)
	}

	result.TotalCount = len(result.Sessions)
	return result, nil
}