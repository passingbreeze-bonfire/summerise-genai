package models

import (
	"time"
)

// CollectionSource는 데이터 수집 소스를 나타냅니다
type CollectionSource string

const (
	SourceClaudeCode CollectionSource = "claude_code"
	SourceGeminiCLI  CollectionSource = "gemini_cli"
	SourceAmazonQ    CollectionSource = "amazon_q"
)

// SessionData는 AI 도구의 세션 데이터를 나타냅니다
type SessionData struct {
	ID          string            `json:"id" yaml:"id"`
	Source      CollectionSource  `json:"source" yaml:"source"`
	Timestamp   time.Time         `json:"timestamp" yaml:"timestamp"`
	Title       string            `json:"title,omitempty" yaml:"title,omitempty"`
	Messages    []Message         `json:"messages" yaml:"messages"`
	Metadata    map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Files       []FileReference   `json:"files,omitempty" yaml:"files,omitempty"`
	Commands    []Command         `json:"commands,omitempty" yaml:"commands,omitempty"`
}

// Message는 대화 메시지를 나타냅니다
type Message struct {
	ID        string            `json:"id" yaml:"id"`
	Role      string            `json:"role" yaml:"role"` // user, assistant, system
	Content   string            `json:"content" yaml:"content"`
	Timestamp time.Time         `json:"timestamp" yaml:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// FileReference는 파일 참조 정보를 나타냅니다
type FileReference struct {
	Path        string    `json:"path" yaml:"path"`
	Name        string    `json:"name" yaml:"name"`
	Size        int64     `json:"size" yaml:"size"`
	ModTime     time.Time `json:"mod_time" yaml:"mod_time"`
	ContentType string    `json:"content_type,omitempty" yaml:"content_type,omitempty"`
	Hash        string    `json:"hash,omitempty" yaml:"hash,omitempty"`
}

// Command는 실행된 명령어 정보를 나타냅니다
type Command struct {
	ID          string            `json:"id" yaml:"id"`
	Command     string            `json:"command" yaml:"command"`
	Args        []string          `json:"args,omitempty" yaml:"args,omitempty"`
	Output      string            `json:"output,omitempty" yaml:"output,omitempty"`
	Error       string            `json:"error,omitempty" yaml:"error,omitempty"`
	ExitCode    int               `json:"exit_code" yaml:"exit_code"`
	Timestamp   time.Time         `json:"timestamp" yaml:"timestamp"`
	Duration    time.Duration     `json:"duration" yaml:"duration"`
	Environment map[string]string `json:"environment,omitempty" yaml:"environment,omitempty"`
}

// CollectionConfig는 데이터 수집 설정을 나타냅니다
type CollectionConfig struct {
	Sources       []CollectionSource `json:"sources" yaml:"sources"`
	IncludeFiles  bool               `json:"include_files" yaml:"include_files"`
	IncludeCommands bool             `json:"include_commands" yaml:"include_commands"`
	DateRange     *DateRange         `json:"date_range,omitempty" yaml:"date_range,omitempty"`
	OutputPath    string             `json:"output_path" yaml:"output_path"`
	Template      string             `json:"template" yaml:"template"`
}

// DateRange는 날짜 범위를 나타냅니다
type DateRange struct {
	Start time.Time `json:"start" yaml:"start"`
	End   time.Time `json:"end" yaml:"end"`
}

// ExportConfig는 마크다운 내보내기 설정을 나타냅니다
type ExportConfig struct {
	Template         string            `json:"template" yaml:"template"`
	OutputPath       string            `json:"output_path" yaml:"output_path"`
	IncludeMetadata  bool              `json:"include_metadata" yaml:"include_metadata"`
	IncludeTimestamps bool             `json:"include_timestamps" yaml:"include_timestamps"`
	FormatCodeBlocks bool              `json:"format_code_blocks" yaml:"format_code_blocks"`
	GenerateTOC      bool              `json:"generate_toc" yaml:"generate_toc"`
	CustomFields     map[string]string `json:"custom_fields,omitempty" yaml:"custom_fields,omitempty"`
}

// CollectionResult는 데이터 수집 결과를 나타냅니다
type CollectionResult struct {
	Sessions    []SessionData     `json:"sessions" yaml:"sessions"`
	TotalCount  int               `json:"total_count" yaml:"total_count"`
	Sources     []CollectionSource `json:"sources" yaml:"sources"`
	CollectedAt time.Time         `json:"collected_at" yaml:"collected_at"`
	Duration    time.Duration     `json:"duration" yaml:"duration"`
	Errors      []string          `json:"errors,omitempty" yaml:"errors,omitempty"`
}

