package models

import (
	"context"
	"io"
)

// Collector는 다양한 AI CLI 도구에서 데이터를 수집하는 인터페이스입니다
type Collector interface {
	// Collect는 설정된 소스에서 세션 데이터를 수집합니다
	Collect(ctx context.Context, config *CollectionConfig) ([]SessionData, error)
	
	// GetSource는 이 수집기가 처리하는 소스 타입을 반환합니다
	GetSource() CollectionSource
	
	// Validate는 수집기 설정이 유효한지 검증합니다
	Validate() error
	
	// GetSupportedFormats는 수집기가 지원하는 데이터 형식들을 반환합니다
	GetSupportedFormats() []string
}

// StreamingCollector는 스트리밍 방식으로 데이터를 수집할 수 있는 수집기 인터페이스입니다
type StreamingCollector interface {
	Collector
	
	// CollectStream은 스트리밍 방식으로 세션 데이터를 수집합니다
	CollectStream(ctx context.Context, config *CollectionConfig, output chan<- SessionData) error
}

// Processor는 수집된 데이터를 처리하고 변환하는 인터페이스입니다
type Processor interface {
	// Process는 세션 데이터를 처리하여 구조화된 형태로 변환합니다
	Process(ctx context.Context, sessions []SessionData) (interface{}, error)
	
	// Validate는 처리기 설정이 유효한지 검증합니다
	Validate() error
	
	// GetSupportedOutputFormats는 지원하는 출력 형식들을 반환합니다
	GetSupportedOutputFormats() []string
}

// StreamingProcessor는 스트리밍 방식으로 데이터를 처리할 수 있는 처리기 인터페이스입니다
type StreamingProcessor interface {
	Processor
	
	// ProcessStream은 스트리밍 방식으로 데이터를 처리합니다
	ProcessStream(ctx context.Context, input <-chan SessionData, output chan<- interface{}) error
}

// Exporter는 처리된 데이터를 다양한 형식으로 내보내는 인터페이스입니다
type Exporter interface {
	// Export는 처리된 데이터를 지정된 형식으로 내보냅니다
	Export(ctx context.Context, data interface{}) error
	
	// ExportToWriter는 처리된 데이터를 Writer에 출력합니다
	ExportToWriter(ctx context.Context, data interface{}, writer io.Writer) error
	
	// GetFormat은 내보내기 형식을 반환합니다
	GetFormat() string
	
	// Validate는 내보내기 설정이 유효한지 검증합니다
	Validate() error
	
	// GetSupportedTemplates는 지원하는 템플릿들을 반환합니다
	GetSupportedTemplates() []string
}

// StreamingExporter는 스트리밍 방식으로 데이터를 내보낼 수 있는 내보내기 인터페이스입니다
type StreamingExporter interface {
	Exporter
	
	// ExportStream은 스트리밍 방식으로 데이터를 내보냅니다
	ExportStream(ctx context.Context, input <-chan interface{}, writer io.Writer) error
}

// Pipeline은 전체 데이터 처리 파이프라인을 관리하는 인터페이스입니다
type Pipeline interface {
	// Execute는 전체 파이프라인을 실행합니다
	Execute(ctx context.Context, config *PipelineConfig) error
	
	// AddCollector는 파이프라인에 수집기를 추가합니다
	AddCollector(collector Collector)
	
	// SetProcessor는 파이프라인의 처리기를 설정합니다
	SetProcessor(processor Processor)
	
	// AddExporter는 파이프라인에 내보내기를 추가합니다
	AddExporter(exporter Exporter)
	
	// Validate는 파이프라인 설정이 유효한지 검증합니다
	Validate() error
}

// CollectorRegistry는 수집기들을 관리하는 레지스트리 인터페이스입니다
type CollectorRegistry interface {
	// Register는 수집기를 등록합니다
	Register(source CollectionSource, factory CollectorFactory)
	
	// Get은 지정된 소스의 수집기를 반환합니다
	Get(source CollectionSource) (Collector, error)
	
	// GetAll은 등록된 모든 수집기들을 반환합니다
	GetAll() map[CollectionSource]Collector
	
	// ListSources는 등록된 모든 소스들을 반환합니다
	ListSources() []CollectionSource
}

// ProcessorRegistry는 처리기들을 관리하는 레지스트리 인터페이스입니다
type ProcessorRegistry interface {
	// Register는 처리기를 등록합니다
	Register(name string, factory ProcessorFactory)
	
	// Get은 지정된 이름의 처리기를 반환합니다
	Get(name string) (Processor, error)
	
	// ListProcessors는 등록된 모든 처리기 이름들을 반환합니다
	ListProcessors() []string
}

// ExporterRegistry는 내보내기 도구들을 관리하는 레지스트리 인터페이스입니다
type ExporterRegistry interface {
	// Register는 내보내기 도구를 등록합니다
	Register(format string, factory ExporterFactory)
	
	// Get은 지정된 형식의 내보내기 도구를 반환합니다
	Get(format string) (Exporter, error)
	
	// ListFormats는 등록된 모든 형식들을 반환합니다
	ListFormats() []string
}

// Filter는 데이터 필터링을 위한 인터페이스입니다
type Filter interface {
	// Apply는 세션 데이터에 필터를 적용합니다
	Apply(sessions []SessionData) []SessionData
	
	// Validate는 필터 설정이 유효한지 검증합니다
	Validate() error
}

// Transformer는 데이터 변환을 위한 인터페이스입니다
type Transformer interface {
	// Transform은 세션 데이터를 변환합니다
	Transform(ctx context.Context, sessions []SessionData) ([]SessionData, error)
	
	// Validate는 변환기 설정이 유효한지 검증합니다
	Validate() error
}

// Validator는 데이터 검증을 위한 인터페이스입니다
type Validator interface {
	// ValidateSession은 개별 세션 데이터를 검증합니다
	ValidateSession(session SessionData) []ValidationError
	
	// ValidateCollection은 수집된 데이터 전체를 검증합니다
	ValidateCollection(sessions []SessionData) []ValidationError
}

// ErrorHandler는 에러 처리를 위한 인터페이스입니다
type ErrorHandler interface {
	// HandleError는 에러를 처리합니다
	HandleError(ctx context.Context, err error, metadata map[string]interface{})
	
	// ShouldRetry는 에러 발생 시 재시도 여부를 결정합니다
	ShouldRetry(err error, attemptCount int) bool
	
	// GetRetryDelay는 재시도 대기 시간을 반환합니다
	GetRetryDelay(attemptCount int) int64 // milliseconds
}

// ProgressReporter는 진행상황 보고를 위한 인터페이스입니다
type ProgressReporter interface {
	// ReportProgress는 진행상황을 보고합니다
	ReportProgress(current, total int, message string)
	
	// ReportError는 에러를 보고합니다
	ReportError(err error)
	
	// Complete는 작업 완료를 보고합니다
	Complete()
}

// Factory 인터페이스들 - 의존성 주입을 위한 팩토리 패턴

// CollectorFactory는 수집기 생성을 위한 팩토리 인터페이스입니다
type CollectorFactory interface {
	// Create는 설정을 바탕으로 수집기를 생성합니다
	Create(config interface{}) (Collector, error)
	
	// GetConfigType은 이 팩토리가 요구하는 설정 타입을 반환합니다
	GetConfigType() interface{}
}

// ProcessorFactory는 처리기 생성을 위한 팩토리 인터페이스입니다
type ProcessorFactory interface {
	// Create는 설정을 바탕으로 처리기를 생성합니다
	Create(config interface{}) (Processor, error)
	
	// GetConfigType은 이 팩토리가 요구하는 설정 타입을 반환합니다
	GetConfigType() interface{}
}

// ExporterFactory는 내보내기 도구 생성을 위한 팩토리 인터페이스입니다
type ExporterFactory interface {
	// Create는 설정을 바탕으로 내보내기 도구를 생성합니다
	Create(config interface{}) (Exporter, error)
	
	// GetConfigType은 이 팩토리가 요구하는 설정 타입을 반환합니다
	GetConfigType() interface{}
}

// Plugin 관련 인터페이스들 - 확장성을 위한 플러그인 시스템

// Plugin는 기본 플러그인 인터페이스입니다
type Plugin interface {
	// GetName은 플러그인 이름을 반환합니다
	GetName() string
	
	// GetVersion은 플러그인 버전을 반환합니다
	GetVersion() string
	
	// Initialize는 플러그인을 초기화합니다
	Initialize(config interface{}) error
	
	// Cleanup은 플러그인 리소스를 정리합니다
	Cleanup() error
}

// CollectorPlugin는 수집기 플러그인 인터페이스입니다
type CollectorPlugin interface {
	Plugin
	
	// CreateCollector는 수집기를 생성합니다
	CreateCollector(config interface{}) (Collector, error)
	
	// GetSupportedSources는 지원하는 소스들을 반환합니다
	GetSupportedSources() []CollectionSource
}

// ProcessorPlugin는 처리기 플러그인 인터페이스입니다
type ProcessorPlugin interface {
	Plugin
	
	// CreateProcessor는 처리기를 생성합니다
	CreateProcessor(config interface{}) (Processor, error)
	
	// GetSupportedTypes는 지원하는 처리 타입들을 반환합니다
	GetSupportedTypes() []string
}

// ExporterPlugin는 내보내기 플러그인 인터페이스입니다
type ExporterPlugin interface {
	Plugin
	
	// CreateExporter는 내보내기 도구를 생성합니다
	CreateExporter(config interface{}) (Exporter, error)
	
	// GetSupportedFormats는 지원하는 형식들을 반환합니다
	GetSupportedFormats() []string
}

// PluginManager는 플러그인 관리를 위한 인터페이스입니다
type PluginManager interface {
	// LoadPlugin은 플러그인을 로드합니다
	LoadPlugin(path string) (Plugin, error)
	
	// RegisterPlugin은 플러그인을 등록합니다
	RegisterPlugin(plugin Plugin) error
	
	// UnloadPlugin은 플러그인을 언로드합니다
	UnloadPlugin(name string) error
	
	// ListPlugins는 로드된 플러그인 목록을 반환합니다
	ListPlugins() []string
	
	// GetPlugin은 지정된 이름의 플러그인을 반환합니다
	GetPlugin(name string) (Plugin, error)
}

// 추가 데이터 타입들

// ProcessedData는 처리된 데이터를 나타내는 인터페이스 호환 구조체입니다
// 기존 processor 패키지의 ProcessedData와 호환성을 위해 별도 정의하지 않음

// PipelineConfig는 파이프라인 설정을 나타냅니다
type PipelineConfig struct {
	CollectionConfig *CollectionConfig `json:"collection" yaml:"collection"`
	ProcessorConfig  interface{}       `json:"processor" yaml:"processor"`
	ExportConfig     *ExportConfig     `json:"export" yaml:"export"`
	
	// 파이프라인 실행 옵션
	Parallel         bool              `json:"parallel" yaml:"parallel"`
	MaxWorkers       int               `json:"max_workers" yaml:"max_workers"`
	TimeoutSeconds   int               `json:"timeout_seconds" yaml:"timeout_seconds"`
	RetryAttempts    int               `json:"retry_attempts" yaml:"retry_attempts"`
	
	// 로깅 및 모니터링
	EnableProgress   bool              `json:"enable_progress" yaml:"enable_progress"`
	EnableMetrics    bool              `json:"enable_metrics" yaml:"enable_metrics"`
	LogLevel         string            `json:"log_level" yaml:"log_level"`
}

// ValidationError는 검증 에러를 나타냅니다
type ValidationError struct {
	Field   string `json:"field" yaml:"field"`
	Value   string `json:"value" yaml:"value"`
	Message string `json:"message" yaml:"message"`
	Code    string `json:"code" yaml:"code"`
}

// Error는 ValidationError가 error 인터페이스를 구현하도록 합니다
func (ve ValidationError) Error() string {
	return ve.Message
}

// Metadata는 메타데이터 정보를 나타냅니다
type Metadata struct {
	Version     string            `json:"version" yaml:"version"`
	CreatedAt   string            `json:"created_at" yaml:"created_at"`
	CreatedBy   string            `json:"created_by" yaml:"created_by"`
	Source      string            `json:"source" yaml:"source"`
	Format      string            `json:"format" yaml:"format"`
	Custom      map[string]string `json:"custom,omitempty" yaml:"custom,omitempty"`
}

// Metrics는 성능 메트릭 정보를 나타냅니다
type Metrics struct {
	StartTime        string            `json:"start_time" yaml:"start_time"`
	EndTime          string            `json:"end_time" yaml:"end_time"`
	Duration         int64             `json:"duration_ms" yaml:"duration_ms"` // milliseconds
	ProcessedItems   int               `json:"processed_items" yaml:"processed_items"`
	ErrorCount       int               `json:"error_count" yaml:"error_count"`
	RetryCount       int               `json:"retry_count" yaml:"retry_count"`
	MemoryUsed       int64             `json:"memory_used_bytes" yaml:"memory_used_bytes"`
	CustomMetrics    map[string]int64  `json:"custom_metrics,omitempty" yaml:"custom_metrics,omitempty"`
}