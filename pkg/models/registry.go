package models

import (
	"context"
	"fmt"
	"sync"
)

// DefaultCollectorRegistry는 기본 수집기 레지스트리 구현입니다
type DefaultCollectorRegistry struct {
	mu        sync.RWMutex
	factories map[CollectionSource]CollectorFactory
}

// NewCollectorRegistry는 새로운 수집기 레지스트리를 생성합니다
func NewCollectorRegistry() CollectorRegistry {
	return &DefaultCollectorRegistry{
		factories: make(map[CollectionSource]CollectorFactory),
	}
}

// Register는 수집기를 등록합니다
func (r *DefaultCollectorRegistry) Register(source CollectionSource, factory CollectorFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[source] = factory
}

// Get은 지정된 소스의 수집기를 반환합니다
func (r *DefaultCollectorRegistry) Get(source CollectionSource) (Collector, error) {
	r.mu.RLock()
	factory, exists := r.factories[source]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("수집기를 찾을 수 없습니다: %s", source)
	}

	return factory.Create(nil) // 기본 설정으로 생성
}

// GetAll은 등록된 모든 수집기들을 반환합니다
func (r *DefaultCollectorRegistry) GetAll() map[CollectionSource]Collector {
	r.mu.RLock()
	defer r.mu.RUnlock()

	collectors := make(map[CollectionSource]Collector)
	for source, factory := range r.factories {
		if collector, err := factory.Create(nil); err == nil {
			collectors[source] = collector
		}
	}

	return collectors
}

// ListSources는 등록된 모든 소스들을 반환합니다
func (r *DefaultCollectorRegistry) ListSources() []CollectionSource {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sources := make([]CollectionSource, 0, len(r.factories))
	for source := range r.factories {
		sources = append(sources, source)
	}

	return sources
}

// DefaultProcessorRegistry는 기본 처리기 레지스트리 구현입니다
type DefaultProcessorRegistry struct {
	mu        sync.RWMutex
	factories map[string]ProcessorFactory
}

// NewProcessorRegistry는 새로운 처리기 레지스트리를 생성합니다
func NewProcessorRegistry() ProcessorRegistry {
	return &DefaultProcessorRegistry{
		factories: make(map[string]ProcessorFactory),
	}
}

// Register는 처리기를 등록합니다
func (r *DefaultProcessorRegistry) Register(name string, factory ProcessorFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[name] = factory
}

// Get은 지정된 이름의 처리기를 반환합니다
func (r *DefaultProcessorRegistry) Get(name string) (Processor, error) {
	r.mu.RLock()
	factory, exists := r.factories[name]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("처리기를 찾을 수 없습니다: %s", name)
	}

	return factory.Create(nil) // 기본 설정으로 생성
}

// ListProcessors는 등록된 모든 처리기 이름들을 반환합니다
func (r *DefaultProcessorRegistry) ListProcessors() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}

	return names
}

// DefaultExporterRegistry는 기본 내보내기 도구 레지스트리 구현입니다
type DefaultExporterRegistry struct {
	mu        sync.RWMutex
	factories map[string]ExporterFactory
}

// NewExporterRegistry는 새로운 내보내기 도구 레지스트리를 생성합니다
func NewExporterRegistry() ExporterRegistry {
	return &DefaultExporterRegistry{
		factories: make(map[string]ExporterFactory),
	}
}

// Register는 내보내기 도구를 등록합니다
func (r *DefaultExporterRegistry) Register(format string, factory ExporterFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[format] = factory
}

// Get은 지정된 형식의 내보내기 도구를 반환합니다
func (r *DefaultExporterRegistry) Get(format string) (Exporter, error) {
	r.mu.RLock()
	factory, exists := r.factories[format]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("내보내기 도구를 찾을 수 없습니다: %s", format)
	}

	return factory.Create(nil) // 기본 설정으로 생성
}

// ListFormats는 등록된 모든 형식들을 반환합니다
func (r *DefaultExporterRegistry) ListFormats() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	formats := make([]string, 0, len(r.factories))
	for format := range r.factories {
		formats = append(formats, format)
	}

	return formats
}

// 팩토리 구현 예제들

// ClaudeCodeCollectorFactory는 Claude Code 수집기 팩토리 구현입니다
type ClaudeCodeCollectorFactory struct{}

// Create는 설정을 바탕으로 수집기를 생성합니다
func (f *ClaudeCodeCollectorFactory) Create(config interface{}) (Collector, error) {
	// config 타입 검증 및 변환 로직이 필요
	// 여기서는 간단한 예제로 nil 체크만 수행
	if config == nil {
		return nil, fmt.Errorf("Claude Code 수집기 설정이 필요합니다")
	}
	
	// 실제 구현에서는 config를 적절한 타입으로 변환하고
	// 해당 설정으로 수집기를 생성해야 합니다
	return nil, fmt.Errorf("구현 필요: Claude Code 수집기 생성")
}

// GetConfigType은 이 팩토리가 요구하는 설정 타입을 반환합니다
func (f *ClaudeCodeCollectorFactory) GetConfigType() interface{} {
	// 실제 구현에서는 설정 구조체 타입을 반환
	return nil
}

// DefaultProcessorFactory는 기본 처리기 팩토리 구현입니다
type DefaultProcessorFactory struct{}

// Create는 설정을 바탕으로 처리기를 생성합니다
func (f *DefaultProcessorFactory) Create(config interface{}) (Processor, error) {
	// 실제 구현에서는 config를 적절한 타입으로 변환하고
	// 해당 설정으로 처리기를 생성해야 합니다
	return nil, fmt.Errorf("구현 필요: 처리기 생성")
}

// GetConfigType은 이 팩토리가 요구하는 설정 타입을 반환합니다
func (f *DefaultProcessorFactory) GetConfigType() interface{} {
	return nil
}

// MarkdownExporterFactory는 마크다운 내보내기 도구 팩토리 구현입니다
type MarkdownExporterFactory struct{}

// Create는 설정을 바탕으로 내보내기 도구를 생성합니다
func (f *MarkdownExporterFactory) Create(config interface{}) (Exporter, error) {
	// 실제 구현에서는 config를 적절한 타입으로 변환하고
	// 해당 설정으로 내보내기 도구를 생성해야 합니다
	return nil, fmt.Errorf("구현 필요: 마크다운 내보내기 도구 생성")
}

// GetConfigType은 이 팩토리가 요구하는 설정 타입을 반환합니다
func (f *MarkdownExporterFactory) GetConfigType() interface{} {
	return nil
}

// DefaultPluginManager는 기본 플러그인 관리자 구현입니다
type DefaultPluginManager struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
}

// NewPluginManager는 새로운 플러그인 관리자를 생성합니다
func NewPluginManager() PluginManager {
	return &DefaultPluginManager{
		plugins: make(map[string]Plugin),
	}
}

// LoadPlugin은 플러그인을 로드합니다
func (pm *DefaultPluginManager) LoadPlugin(path string) (Plugin, error) {
	// 실제 구현에서는 동적 라이브러리 로딩 로직이 필요
	return nil, fmt.Errorf("구현 필요: 플러그인 로딩")
}

// RegisterPlugin은 플러그인을 등록합니다
func (pm *DefaultPluginManager) RegisterPlugin(plugin Plugin) error {
	if plugin == nil {
		return fmt.Errorf("플러그인이 nil입니다")
	}

	name := plugin.GetName()
	if name == "" {
		return fmt.Errorf("플러그인 이름이 비어있습니다")
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.plugins[name]; exists {
		return fmt.Errorf("플러그인이 이미 등록되어 있습니다: %s", name)
	}

	pm.plugins[name] = plugin
	return nil
}

// UnloadPlugin은 플러그인을 언로드합니다
func (pm *DefaultPluginManager) UnloadPlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("플러그인을 찾을 수 없습니다: %s", name)
	}

	// 플러그인 정리
	if err := plugin.Cleanup(); err != nil {
		return fmt.Errorf("플러그인 정리 실패: %w", err)
	}

	delete(pm.plugins, name)
	return nil
}

// ListPlugins는 로드된 플러그인 목록을 반환합니다
func (pm *DefaultPluginManager) ListPlugins() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	names := make([]string, 0, len(pm.plugins))
	for name := range pm.plugins {
		names = append(names, name)
	}

	return names
}

// GetPlugin은 지정된 이름의 플러그인을 반환합니다
func (pm *DefaultPluginManager) GetPlugin(name string) (Plugin, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return nil, fmt.Errorf("플러그인을 찾을 수 없습니다: %s", name)
	}

	return plugin, nil
}

// 기본 파이프라인 구현

// DefaultPipeline은 기본 파이프라인 구현입니다
type DefaultPipeline struct {
	collectors []Collector
	processor  Processor
	exporters  []Exporter
}

// NewPipeline은 새로운 파이프라인을 생성합니다
func NewPipeline() Pipeline {
	return &DefaultPipeline{
		collectors: make([]Collector, 0),
		exporters:  make([]Exporter, 0),
	}
}

// Execute는 전체 파이프라인을 실행합니다
func (p *DefaultPipeline) Execute(ctx context.Context, config *PipelineConfig) error {
	// 실제 파이프라인 실행 로직 구현 필요
	return fmt.Errorf("구현 필요: 파이프라인 실행")
}

// AddCollector는 파이프라인에 수집기를 추가합니다
func (p *DefaultPipeline) AddCollector(collector Collector) {
	p.collectors = append(p.collectors, collector)
}

// SetProcessor는 파이프라인의 처리기를 설정합니다
func (p *DefaultPipeline) SetProcessor(processor Processor) {
	p.processor = processor
}

// AddExporter는 파이프라인에 내보내기를 추가합니다
func (p *DefaultPipeline) AddExporter(exporter Exporter) {
	p.exporters = append(p.exporters, exporter)
}

// Validate는 파이프라인 설정이 유효한지 검증합니다
func (p *DefaultPipeline) Validate() error {
	if len(p.collectors) == 0 {
		return fmt.Errorf("최소 하나의 수집기가 필요합니다")
	}

	if p.processor == nil {
		return fmt.Errorf("처리기가 설정되지 않았습니다")
	}

	if len(p.exporters) == 0 {
		return fmt.Errorf("최소 하나의 내보내기 도구가 필요합니다")
	}

	// 각 구성 요소 검증
	for _, collector := range p.collectors {
		if err := collector.Validate(); err != nil {
			return fmt.Errorf("수집기 검증 실패: %w", err)
		}
	}

	if err := p.processor.Validate(); err != nil {
		return fmt.Errorf("처리기 검증 실패: %w", err)
	}

	for _, exporter := range p.exporters {
		if err := exporter.Validate(); err != nil {
			return fmt.Errorf("내보내기 도구 검증 실패: %w", err)
		}
	}

	return nil
}