package collector

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ssamai/internal/config"
	"ssamai/pkg/models"
)

// MockFileReader는 테스트용 파일 리더
type MockFileReader struct {
	files map[string][]byte
	stats map[string]os.FileInfo
}

// MockFileInfo는 테스트용 파일 정보
type MockFileInfo struct {
	name    string
	size    int64
	modTime time.Time
	isDir   bool
}

func (m MockFileInfo) Name() string       { return m.name }
func (m MockFileInfo) Size() int64        { return m.size }
func (m MockFileInfo) Mode() os.FileMode  { return 0644 }
func (m MockFileInfo) ModTime() time.Time { return m.modTime }
func (m MockFileInfo) IsDir() bool        { return m.isDir }
func (m MockFileInfo) Sys() interface{}   { return nil }

func NewMockFileReader() *MockFileReader {
	return &MockFileReader{
		files: make(map[string][]byte),
		stats: make(map[string]os.FileInfo),
	}
}

func (m *MockFileReader) ReadFile(filename string) ([]byte, error) {
	if data, exists := m.files[filename]; exists {
		return data, nil
	}
	return nil, os.ErrNotExist
}

func (m *MockFileReader) Stat(filename string) (os.FileInfo, error) {
	if stat, exists := m.stats[filename]; exists {
		return stat, nil
	}
	return nil, os.ErrNotExist
}

func (m *MockFileReader) WalkDir(root string, fn fs.WalkDirFunc) error {
	for path := range m.files {
		if strings.HasPrefix(path, root) {
			info := m.stats[path]
			if err := fn(path, fs.FileInfoToDirEntry(info), nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *MockFileReader) AddFile(path string, content []byte) {
	m.files[path] = content
	m.stats[path] = MockFileInfo{
		name:    filepath.Base(path),
		size:    int64(len(content)),
		modTime: time.Now(),
		isDir:   false,
	}
}

func (m *MockFileReader) AddDir(path string) {
	m.stats[path] = MockFileInfo{
		name:    filepath.Base(path),
		size:    0,
		modTime: time.Now(),
		isDir:   true,
	}
}

// MockLogger는 테스트용 로거
type MockLogger struct {
	logs []string
}

func (m *MockLogger) Printf(format string, v ...interface{}) {
	m.logs = append(m.logs, "INFO: "+fmt.Sprintf(format, v...))
}

func (m *MockLogger) Errorf(format string, v ...interface{}) {
	m.logs = append(m.logs, "ERROR: "+fmt.Sprintf(format, v...))
}

func (m *MockLogger) Warnf(format string, v ...interface{}) {
	m.logs = append(m.logs, "WARN: "+fmt.Sprintf(format, v...))
}

func TestNewImprovedGeminiCLICollector(t *testing.T) {
	config := config.CLIToolConfig{
		ConfigDir: "/test/config",
	}

	collector := NewImprovedGeminiCLICollector(config)

	if collector == nil {
		t.Fatal("collector should not be nil")
	}

	if collector.config.ConfigDir != "/test/config" {
		t.Errorf("expected config dir '/test/config', got '%s'", collector.config.ConfigDir)
	}

	if collector.GetSource() != models.SourceGeminiCLI {
		t.Errorf("expected source %s, got %s", models.SourceGeminiCLI, collector.GetSource())
	}
}

func TestWithFileReader(t *testing.T) {
	config := config.CLIToolConfig{}
	mockReader := NewMockFileReader()
	
	collector := NewImprovedGeminiCLICollector(config).WithFileReader(mockReader)

	if collector.fileReader != mockReader {
		t.Error("fileReader was not set correctly")
	}
}

func TestWithLogger(t *testing.T) {
	config := config.CLIToolConfig{}
	mockLogger := &MockLogger{}
	
	collector := NewImprovedGeminiCLICollector(config).WithLogger(mockLogger)

	if collector.logger != mockLogger {
		t.Error("logger was not set correctly")
	}
}

func TestCollectWithNilConfig(t *testing.T) {
	collector := NewImprovedGeminiCLICollector(config.CLIToolConfig{})
	
	_, err := collector.Collect(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil collection config")
	}
}

func TestCollectFromHistoryWithValidJSON(t *testing.T) {
	mockReader := NewMockFileReader()
	mockLogger := &MockLogger{}
	
	// 테스트 히스토리 파일 생성
	historyContent := `{"id":"test-1","command":"gemini","prompt":"Hello","response":"Hi there","timestamp":"2024-01-01T10:00:00Z","model":"gemini-pro"}
{"id":"test-2","command":"gemini","prompt":"What is Go?","response":"Go is a programming language","timestamp":"2024-01-01T11:00:00Z","model":"gemini-pro"}`
	
	historyPath := "/test/history.jsonl"
	configDir := "/test"
	mockReader.AddFile(historyPath, []byte(historyContent))
	mockReader.AddDir(configDir)

	collector := NewImprovedGeminiCLICollector(config.CLIToolConfig{
		ConfigDir:   configDir,
		HistoryFile: historyPath,
	}).WithFileReader(mockReader).WithLogger(mockLogger)

	collectConfig := &models.CollectionConfig{
		Sources: []models.CollectionSource{models.SourceGeminiCLI},
	}

	sessions, err := collector.Collect(context.Background(), collectConfig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	// 첫 번째 세션 검증
	session1 := sessions[0]
	if session1.ID != "test-1" {
		t.Errorf("expected ID 'test-1', got '%s'", session1.ID)
	}
	if session1.Source != models.SourceGeminiCLI {
		t.Errorf("expected source %s, got %s", models.SourceGeminiCLI, session1.Source)
	}
	if len(session1.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(session1.Messages))
	}

	// 메시지 검증
	if session1.Messages[0].Role != "user" || session1.Messages[0].Content != "Hello" {
		t.Errorf("unexpected first message: role=%s, content=%s", session1.Messages[0].Role, session1.Messages[0].Content)
	}
	if session1.Messages[1].Role != "assistant" || session1.Messages[1].Content != "Hi there" {
		t.Errorf("unexpected second message: role=%s, content=%s", session1.Messages[1].Role, session1.Messages[1].Content)
	}
}

func TestCollectFromHistoryWithTextFormat(t *testing.T) {
	mockReader := NewMockFileReader()
	mockLogger := &MockLogger{}
	
	// 텍스트 형태의 히스토리 파일
	historyContent := `What is the weather today?
How do I install Go?
Tell me about machine learning`
	
	historyPath := "/test/history.txt"
	configDir := "/test"
	mockReader.AddFile(historyPath, []byte(historyContent))
	mockReader.AddDir(configDir)

	collector := NewImprovedGeminiCLICollector(config.CLIToolConfig{
		ConfigDir:   configDir,
		HistoryFile: historyPath,
	}).WithFileReader(mockReader).WithLogger(mockLogger)

	collectConfig := &models.CollectionConfig{
		Sources: []models.CollectionSource{models.SourceGeminiCLI},
	}

	sessions, err := collector.Collect(context.Background(), collectConfig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(sessions))
	}

	// 첫 번째 세션의 메시지 내용 확인
	if sessions[0].Messages[0].Content != "What is the weather today?" {
		t.Errorf("unexpected message content: %s", sessions[0].Messages[0].Content)
	}
}

func TestCollectFromSessionDirectory(t *testing.T) {
	mockReader := NewMockFileReader()
	mockLogger := &MockLogger{}
	
	// 세션 파일들 생성
	session1JSON := `{
		"id": "session-1",
		"title": "Test Session 1",
		"created_at": "2024-01-01T10:00:00Z",
		"model": "gemini-pro",
		"messages": [
			{
				"id": "msg-1",
				"role": "user",
				"content": "Hello Gemini",
				"timestamp": "2024-01-01T10:00:00Z"
			},
			{
				"id": "msg-2",
				"role": "assistant",
				"content": "Hello! How can I help you?",
				"timestamp": "2024-01-01T10:00:05Z"
			}
		]
	}`

	session2JSON := `{
		"id": "session-2",
		"title": "Test Session 2",
		"created_at": "2024-01-01T11:00:00Z",
		"model": "gemini-pro",
		"messages": [
			{
				"id": "msg-3",
				"role": "user",
				"parts": [{"type": "text", "text": "What is AI?"}],
				"timestamp": "2024-01-01T11:00:00Z"
			}
		]
	}`

	sessionDir := "/test/sessions"
	mockReader.AddDir(sessionDir)
	mockReader.AddFile(filepath.Join(sessionDir, "session1.json"), []byte(session1JSON))
	mockReader.AddFile(filepath.Join(sessionDir, "session2.json"), []byte(session2JSON))
	mockReader.AddDir("/test")

	collector := NewImprovedGeminiCLICollector(config.CLIToolConfig{
		ConfigDir:  "/test",
		SessionDir: sessionDir,
	}).WithFileReader(mockReader).WithLogger(mockLogger)

	collectConfig := &models.CollectionConfig{
		Sources: []models.CollectionSource{models.SourceGeminiCLI},
	}

	sessions, err := collector.Collect(context.Background(), collectConfig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	// 첫 번째 세션 검증
	session1 := sessions[0]
	if session1.ID != "session-1" {
		t.Errorf("expected ID 'session-1', got '%s'", session1.ID)
	}
	if session1.Title != "Test Session 1" {
		t.Errorf("expected title 'Test Session 1', got '%s'", session1.Title)
	}
	if len(session1.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(session1.Messages))
	}

	// 두 번째 세션에서 parts 처리 확인
	session2 := sessions[1]
	if len(session2.Messages) != 1 {
		t.Fatalf("expected 1 message in session2, got %d", len(session2.Messages))
	}
	if session2.Messages[0].Content != "What is AI?" {
		t.Errorf("expected content 'What is AI?', got '%s'", session2.Messages[0].Content)
	}
}

func TestDateRangeFiltering(t *testing.T) {
	mockReader := NewMockFileReader()
	mockLogger := &MockLogger{}
	
	// 다양한 날짜의 히스토리 생성
	historyContent := `{"id":"old","command":"gemini","prompt":"Old question","timestamp":"2023-12-01T10:00:00Z"}
{"id":"recent","command":"gemini","prompt":"Recent question","timestamp":"2024-01-15T10:00:00Z"}
{"id":"very-recent","command":"gemini","prompt":"Very recent question","timestamp":"2024-02-01T10:00:00Z"}`
	
	historyPath := "/test/history.jsonl"
	configDir := "/test"
	mockReader.AddFile(historyPath, []byte(historyContent))
	mockReader.AddDir(configDir)

	collector := NewImprovedGeminiCLICollector(config.CLIToolConfig{
		ConfigDir:   configDir,
		HistoryFile: historyPath,
	}).WithFileReader(mockReader).WithLogger(mockLogger)

	// 2024년 1월만 필터링
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
	
	collectConfig := &models.CollectionConfig{
		Sources: []models.CollectionSource{models.SourceGeminiCLI},
		DateRange: &models.DateRange{
			Start: startDate,
			End:   endDate,
		},
	}

	sessions, err := collector.Collect(context.Background(), collectConfig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("expected 1 session after date filtering, got %d", len(sessions))
	}

	if sessions[0].ID != "recent" {
		t.Errorf("expected filtered session ID 'recent', got '%s'", sessions[0].ID)
	}
}

func TestContextCancellation(t *testing.T) {
	mockReader := NewMockFileReader()
	mockLogger := &MockLogger{}
	
	configDir := "/test"
	historyPath := "/test/history.jsonl"
	
	// 테스트 파일과 디렉토리 추가
	mockReader.AddDir(configDir)
	mockReader.AddFile(historyPath, []byte(`{"id":"test","prompt":"test","timestamp":"2024-01-01T10:00:00Z"}`))
	
	// 취소될 수 있는 컨텍스트 생성
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 즉시 취소

	collector := NewImprovedGeminiCLICollector(config.CLIToolConfig{
		ConfigDir:   configDir,
		HistoryFile: historyPath,
	}).WithFileReader(mockReader).WithLogger(mockLogger)

	collectConfig := &models.CollectionConfig{
		Sources: []models.CollectionSource{models.SourceGeminiCLI},
	}

	_, err := collector.Collect(ctx, collectConfig)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

func TestValidateConfigDirectory(t *testing.T) {
	mockReader := NewMockFileReader()
	
	tests := []struct {
		name        string
		configDir   string
		setupMock   func(*MockFileReader)
		expectError bool
	}{
		{
			name:      "valid directory",
			configDir: "/test",
			setupMock: func(m *MockFileReader) {
				m.AddDir("/test")
			},
			expectError: false,
		},
		{
			name:        "non-existent directory",
			configDir:   "/nonexistent",
			setupMock:   func(m *MockFileReader) {},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mockReader)
			
			collector := NewImprovedGeminiCLICollector(config.CLIToolConfig{
				ConfigDir: tt.configDir,
			}).WithFileReader(mockReader)

			err := collector.Validate()
			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestGetSupportedFormats(t *testing.T) {
	collector := NewImprovedGeminiCLICollector(config.CLIToolConfig{})
	
	formats := collector.GetSupportedFormats()
	expectedFormats := []string{"json", "text", "jsonl"}
	
	if len(formats) != len(expectedFormats) {
		t.Fatalf("expected %d formats, got %d", len(expectedFormats), len(formats))
	}
	
	for i, expected := range expectedFormats {
		if formats[i] != expected {
			t.Errorf("expected format %s at index %d, got %s", expected, i, formats[i])
		}
	}
}

func TestExtractTitleFromPrompt(t *testing.T) {
	collector := NewImprovedGeminiCLICollector(config.CLIToolConfig{})
	
	tests := []struct {
		prompt   string
		expected string
	}{
		{"Hello world", "Hello world"},
		{"", "Gemini CLI Session"},
		{"First line\nSecond line", "First line"},
		{strings.Repeat("a", 150), strings.Repeat("a", 97) + "..."},
		{"   \n  \n  ", "Gemini CLI Session"},
	}
	
	for _, tt := range tests {
		result := collector.extractTitleFromPrompt(tt.prompt)
		if result != tt.expected {
			t.Errorf("extractTitleFromPrompt(%q) = %q, expected %q", tt.prompt, result, tt.expected)
		}
	}
}

// 벤치마크 테스트
func BenchmarkCollectFromHistory(b *testing.B) {
	mockReader := NewMockFileReader()
	mockLogger := &MockLogger{}
	
	// 큰 히스토리 파일 생성
	var historyLines []string
	for i := 0; i < 1000; i++ {
		historyLines = append(historyLines, fmt.Sprintf(
			`{"id":"test-%d","command":"gemini","prompt":"Question %d","response":"Answer %d","timestamp":"2024-01-01T10:00:00Z"}`,
			i, i, i))
	}
	historyContent := strings.Join(historyLines, "\n")
	
	historyPath := "/test/history.jsonl"
	mockReader.AddFile(historyPath, []byte(historyContent))
	mockReader.AddDir("/test")

	collector := NewImprovedGeminiCLICollector(config.CLIToolConfig{
		ConfigDir:   "/test",
		HistoryFile: historyPath,
	}).WithFileReader(mockReader).WithLogger(mockLogger)

	collectConfig := &models.CollectionConfig{
		Sources: []models.CollectionSource{models.SourceGeminiCLI},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := collector.Collect(context.Background(), collectConfig)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}