package collector

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ssamai/internal/config"
	"ssamai/pkg/models"
)

// MockAmazonQFileReader는 테스트용 파일 리더
type MockAmazonQFileReader struct {
	files   map[string][]byte
	dirs    map[string]bool
	errors  map[string]error
}

func NewMockAmazonQFileReader() *MockAmazonQFileReader {
	return &MockAmazonQFileReader{
		files:  make(map[string][]byte),
		dirs:   make(map[string]bool),
		errors: make(map[string]error),
	}
}

func (m *MockAmazonQFileReader) AddFile(path string, content []byte) {
	m.files[path] = content
}

func (m *MockAmazonQFileReader) AddDir(path string) {
	m.dirs[path] = true
}

func (m *MockAmazonQFileReader) AddError(path string, err error) {
	m.errors[path] = err
}

func (m *MockAmazonQFileReader) ReadFile(filename string) ([]byte, error) {
	if err, exists := m.errors[filename]; exists {
		return nil, err
	}
	if content, exists := m.files[filename]; exists {
		return content, nil
	}
	return nil, os.ErrNotExist
}

func (m *MockAmazonQFileReader) Stat(filename string) (os.FileInfo, error) {
	if err, exists := m.errors[filename]; exists {
		return nil, err
	}
	if _, exists := m.files[filename]; exists {
		return &mockFileInfo{
			name: filepath.Base(filename),
			size: int64(len(m.files[filename])),
		}, nil
	}
	if _, exists := m.dirs[filename]; exists {
		return &mockFileInfo{
			name:  filepath.Base(filename),
			isDir: true,
		}, nil
	}
	return nil, os.ErrNotExist
}

func (m *MockAmazonQFileReader) WalkDir(root string, fn fs.WalkDirFunc) error {
	// Mock implementation - returns files in the directory
	for path := range m.files {
		if strings.HasPrefix(path, root) {
			err := fn(path, &mockDirEntry{
				name:  filepath.Base(path),
				isDir: false,
			}, nil)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *MockAmazonQFileReader) OpenFile(name string) (*os.File, error) {
	if err, exists := m.errors[name]; exists {
		return nil, err
	}
	if _, exists := m.files[name]; exists {
		// 테스트에서는 실제 파일을 만들지 않고 파일이 존재한다고 가정
		// 실제로는 스트리밍 파싱이 되지만 테스트에서는 단순화
		return nil, nil 
	}
	return nil, os.ErrNotExist
}

// mockFileInfo implements os.FileInfo
type mockFileInfo struct {
	name  string
	size  int64
	isDir bool
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return m.size }
func (m *mockFileInfo) Mode() os.FileMode  { return 0644 }
func (m *mockFileInfo) ModTime() time.Time { return time.Now() }
func (m *mockFileInfo) IsDir() bool        { return m.isDir }
func (m *mockFileInfo) Sys() interface{}   { return nil }

// mockDirEntry implements fs.DirEntry
type mockDirEntry struct {
	name  string
	isDir bool
}

func (m *mockDirEntry) Name() string               { return m.name }
func (m *mockDirEntry) IsDir() bool                { return m.isDir }
func (m *mockDirEntry) Type() fs.FileMode          { return 0644 }
func (m *mockDirEntry) Info() (fs.FileInfo, error) { 
	return &mockFileInfo{name: m.name, isDir: m.isDir}, nil 
}

// MockAmazonQLogger는 테스트용 로거
type MockAmazonQLogger struct {
	logs []string
}

func NewMockAmazonQLogger() *MockAmazonQLogger {
	return &MockAmazonQLogger{
		logs: make([]string, 0),
	}
}

func (m *MockAmazonQLogger) Printf(format string, v ...interface{}) {
	m.logs = append(m.logs, "INFO: "+format)
}

func (m *MockAmazonQLogger) Errorf(format string, v ...interface{}) {
	m.logs = append(m.logs, "ERROR: "+format)
}

func (m *MockAmazonQLogger) Warnf(format string, v ...interface{}) {
	m.logs = append(m.logs, "WARN: "+format)
}

func (m *MockAmazonQLogger) GetLogs() []string {
	return m.logs
}

func TestNewAmazonQCollector(t *testing.T) {
	cfg := config.CLIToolConfig{
		ConfigDir: "~/.amazon-q",
	}

	collector := NewAmazonQCollector(cfg)

	if collector == nil {
		t.Fatal("Expected non-nil collector")
	}

	if collector.config.ConfigDir != cfg.ConfigDir {
		t.Errorf("Expected config dir %s, got %s", cfg.ConfigDir, collector.config.ConfigDir)
	}
}

func TestAmazonQCollector_GetSource(t *testing.T) {
	collector := NewAmazonQCollector(config.CLIToolConfig{})
	
	source := collector.GetSource()
	if source != models.SourceAmazonQ {
		t.Errorf("Expected source %s, got %s", models.SourceAmazonQ, source)
	}
}

func TestAmazonQCollector_GetSupportedFormats(t *testing.T) {
	collector := NewAmazonQCollector(config.CLIToolConfig{})
	
	formats := collector.GetSupportedFormats()
	expected := []string{"json", "text", "aws-logs", "session"}
	
	if len(formats) != len(expected) {
		t.Errorf("Expected %d formats, got %d", len(expected), len(formats))
	}
	
	for i, format := range expected {
		if formats[i] != format {
			t.Errorf("Expected format %s, got %s", format, formats[i])
		}
	}
}

func TestAmazonQCollector_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    config.CLIToolConfig
		mockFiles map[string][]byte
		wantError bool
	}{
		{
			name: "valid config with existing directory",
			config: config.CLIToolConfig{
				ConfigDir: "/test/.amazon-q",
			},
			mockFiles: map[string][]byte{},
			wantError: false,
		},
		{
			name: "empty config directory",
			config: config.CLIToolConfig{
				ConfigDir: "",
			},
			mockFiles: map[string][]byte{},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := NewAmazonQCollector(tt.config)
			mockReader := NewMockAmazonQFileReader()
			
			if tt.config.ConfigDir != "" {
				mockReader.AddDir(tt.config.ConfigDir)
			}
			
			collector.WithFileReader(mockReader)
			
			err := collector.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestAmazonQCollector_Collect_DummyData(t *testing.T) {
	cfg := config.CLIToolConfig{
		ConfigDir: "/nonexistent/.amazon-q",
	}

	collector := NewAmazonQCollector(cfg)
	mockReader := NewMockAmazonQFileReader()
	mockLogger := NewMockAmazonQLogger()
	
	collector.WithFileReader(mockReader).WithLogger(mockLogger)

	ctx := context.Background()
	collectConfig := &models.CollectionConfig{}

	sessions, err := collector.Collect(ctx, collectConfig)

	if err != nil {
		t.Errorf("Collect() error = %v, expected nil", err)
	}

	if len(sessions) == 0 {
		t.Error("Expected dummy data, got no sessions")
	}

	// 더미 데이터 검증
	for _, session := range sessions {
		if session.Source != models.SourceAmazonQ {
			t.Errorf("Expected source %s, got %s", models.SourceAmazonQ, session.Source)
		}
		
		if len(session.Messages) == 0 {
			t.Error("Expected messages in dummy session")
		}
		
		if session.Metadata["source_type"] != "amazon_q_dummy" {
			t.Error("Expected dummy data marker in metadata")
		}
	}
}

func TestAmazonQCollector_Collect_WithHistoryFile(t *testing.T) {
	cfg := config.CLIToolConfig{
		ConfigDir:   "/test/.amazon-q",
		HistoryFile: "/test/.amazon-q/history.json",
	}

	historyContent := `{"id": "test-1", "query": "How to create EC2?", "response": "Use AWS console", "timestamp": "2024-01-01T00:00:00Z", "service": "ec2", "region": "us-west-2"}`

	collector := NewAmazonQCollector(cfg)
	mockReader := NewMockAmazonQFileReader()
	mockLogger := NewMockAmazonQLogger()

	mockReader.AddDir("/test/.amazon-q")
	mockReader.AddFile("/test/.amazon-q/history.json", []byte(historyContent))

	collector.WithFileReader(mockReader).WithLogger(mockLogger)

	ctx := context.Background()
	collectConfig := &models.CollectionConfig{}

	sessions, err := collector.Collect(ctx, collectConfig)

	if err != nil {
		t.Errorf("Collect() error = %v, expected nil", err)
	}

	if len(sessions) == 0 {
		t.Error("Expected sessions from history file")
	}

	// 첫 번째 세션 검증
	session := sessions[0]
	if session.ID != "test-1" {
		t.Errorf("Expected session ID 'test-1', got '%s'", session.ID)
	}

	if len(session.Messages) != 2 {
		t.Errorf("Expected 2 messages (user + assistant), got %d", len(session.Messages))
	}

	if session.Messages[0].Role != "user" {
		t.Errorf("Expected first message to be from user, got '%s'", session.Messages[0].Role)
	}

	if session.Messages[1].Role != "assistant" {
		t.Errorf("Expected second message to be from assistant, got '%s'", session.Messages[1].Role)
	}
}

func TestAmazonQCollector_Collect_WithSessionDir(t *testing.T) {
	cfg := config.CLIToolConfig{
		ConfigDir:  "/test/.amazon-q",
		SessionDir: "/test/.amazon-q/sessions",
	}

	sessionContent := `{
		"id": "session-1",
		"title": "Test Session",
		"service": "lambda",
		"region": "us-east-1",
		"created_at": "2024-01-01T00:00:00Z",
		"messages": [
			{
				"id": "msg-1",
				"role": "user",
				"content": "How to optimize Lambda?",
				"timestamp": "2024-01-01T00:00:00Z",
				"service": "lambda"
			}
		]
	}`

	collector := NewAmazonQCollector(cfg)
	mockReader := NewMockAmazonQFileReader()
	mockLogger := NewMockAmazonQLogger()

	mockReader.AddDir("/test/.amazon-q")
	mockReader.AddDir("/test/.amazon-q/sessions")
	mockReader.AddFile("/test/.amazon-q/sessions/session1.json", []byte(sessionContent))

	collector.WithFileReader(mockReader).WithLogger(mockLogger)

	ctx := context.Background()
	collectConfig := &models.CollectionConfig{}

	sessions, err := collector.Collect(ctx, collectConfig)

	if err != nil {
		t.Errorf("Collect() error = %v, expected nil", err)
	}

	if len(sessions) == 0 {
		t.Error("Expected sessions from session directory")
	}

	// 첫 번째 세션 검증
	session := sessions[0]
	if session.ID != "session-1" {
		t.Errorf("Expected session ID 'session-1', got '%s'", session.ID)
	}

	if session.Title != "Test Session" {
		t.Errorf("Expected title 'Test Session', got '%s'", session.Title)
	}

	if session.Metadata["service"] != "lambda" {
		t.Errorf("Expected service 'lambda', got '%s'", session.Metadata["service"])
	}
}

func TestAmazonQCollector_Collect_WithDateFiltering(t *testing.T) {
	cfg := config.CLIToolConfig{
		ConfigDir:   "/test/.amazon-q",
		HistoryFile: "/test/.amazon-q/history.json",
	}

	// 과거와 현재의 히스토리 엔트리
	pastEntry := `{"id": "past-1", "query": "Old query", "response": "Old response", "timestamp": "2023-01-01T00:00:00Z"}`
	recentEntry := `{"id": "recent-1", "query": "New query", "response": "New response", "timestamp": "2024-12-01T00:00:00Z"}`
	historyContent := pastEntry + "\n" + recentEntry

	collector := NewAmazonQCollector(cfg)
	mockReader := NewMockAmazonQFileReader()
	mockLogger := NewMockAmazonQLogger()

	mockReader.AddDir("/test/.amazon-q")
	mockReader.AddFile("/test/.amazon-q/history.json", []byte(historyContent))

	collector.WithFileReader(mockReader).WithLogger(mockLogger)

	ctx := context.Background()
	
	// 2024년 이후만 필터링
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	collectConfig := &models.CollectionConfig{
		DateRange: &models.DateRange{
			Start: startDate,
		},
	}

	sessions, err := collector.Collect(ctx, collectConfig)

	if err != nil {
		t.Errorf("Collect() error = %v, expected nil", err)
	}

	// 2024년 이후 데이터만 있어야 함
	for _, session := range sessions {
		if session.Timestamp.Before(startDate) {
			t.Errorf("Session %s timestamp %v is before filter date %v", session.ID, session.Timestamp, startDate)
		}
	}
}

func TestAmazonQCollector_extractTitleFromQuery(t *testing.T) {
	collector := NewAmazonQCollector(config.CLIToolConfig{})

	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "empty query",
			query:    "",
			expected: "Amazon Q CLI Session",
		},
		{
			name:     "single line query",
			query:    "How to create EC2 instance?",
			expected: "How to create EC2 instance?",
		},
		{
			name:     "multi-line query",
			query:    "How to create EC2 instance?\nWith auto-scaling group",
			expected: "How to create EC2 instance?",
		},
		{
			name:     "very long query",
			query:    strings.Repeat("a", 150),
			expected: strings.Repeat("a", 97) + "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collector.extractTitleFromQuery(tt.query)
			if result != tt.expected {
				t.Errorf("extractTitleFromQuery() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAmazonQCollector_isAmazonQFile(t *testing.T) {
	collector := NewAmazonQCollector(config.CLIToolConfig{})

	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{
			name:     "JSON file",
			filePath: "/path/to/session.json",
			expected: true,
		},
		{
			name:     "Log file",
			filePath: "/path/to/amazon.log",
			expected: true,
		},
		{
			name:     "Amazon Q specific file",
			filePath: "/path/to/amazonq-session.txt",
			expected: true,
		},
		{
			name:     "AWS Q file",
			filePath: "/path/to/aws-q-data.json",
			expected: true,
		},
		{
			name:     "Random text file",
			filePath: "/path/to/readme.txt",
			expected: false,
		},
		{
			name:     "Binary file",
			filePath: "/path/to/image.png",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collector.isAmazonQFile(tt.filePath)
			if result != tt.expected {
				t.Errorf("isAmazonQFile(%s) = %v, want %v", tt.filePath, result, tt.expected)
			}
		})
	}
}

func TestAmazonQCollector_generateDummyData(t *testing.T) {
	collector := NewAmazonQCollector(config.CLIToolConfig{})
	
	sessions := collector.generateDummyData()
	
	if len(sessions) == 0 {
		t.Error("Expected dummy data, got empty slice")
	}
	
	for i, session := range sessions {
		if session.Source != models.SourceAmazonQ {
			t.Errorf("Session %d: expected source %s, got %s", i, models.SourceAmazonQ, session.Source)
		}
		
		if len(session.Messages) == 0 {
			t.Errorf("Session %d: expected messages, got empty slice", i)
		}
		
		if session.Metadata["source_type"] != "amazon_q_dummy" {
			t.Errorf("Session %d: expected dummy source type", i)
		}
		
		if session.Title == "" {
			t.Errorf("Session %d: expected title", i)
		}
		
		// 메시지 검증
		hasUserMessage := false
		hasAssistantMessage := false
		
		for _, msg := range session.Messages {
			if msg.Role == "user" {
				hasUserMessage = true
			}
			if msg.Role == "assistant" {
				hasAssistantMessage = true
			}
		}
		
		if !hasUserMessage {
			t.Errorf("Session %d: expected user message", i)
		}
		
		if !hasAssistantMessage {
			t.Errorf("Session %d: expected assistant message", i)
		}
	}
}