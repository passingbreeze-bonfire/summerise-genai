package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCollectionSource(t *testing.T) {
	tests := []struct {
		name     string
		source   CollectionSource
		expected string
	}{
		{
			name:     "Claude Code source",
			source:   SourceClaudeCode,
			expected: "claude_code",
		},
		{
			name:     "Gemini CLI source",
			source:   SourceGeminiCLI,
			expected: "gemini_cli",
		},
		{
			name:     "Amazon Q source",
			source:   SourceAmazonQ,
			expected: "amazon_q",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.source))
		})
	}
}

func TestSessionData_JSONSerialization(t *testing.T) {
	now := time.Now()
	
	session := SessionData{
		ID:        "test-session-123",
		Source:    SourceClaudeCode,
		Timestamp: now,
		Title:     "Test Session",
		Messages: []Message{
			{
				ID:        "msg-1",
				Role:      "user",
				Content:   "Hello",
				Timestamp: now,
				Metadata:  map[string]string{"type": "text"},
			},
		},
		Metadata: map[string]string{
			"version": "1.0",
			"project": "test",
		},
		Files: []FileReference{
			{
				Path:        "/test/file.go",
				Name:        "file.go",
				Size:        1024,
				ModTime:     now,
				ContentType: "text/plain",
				Hash:        "abc123",
			},
		},
		Commands: []Command{
			{
				ID:        "cmd-1",
				Command:   "go",
				Args:      []string{"test", "./..."},
				Output:    "ok",
				ExitCode:  0,
				Timestamp: now,
				Duration:  time.Second,
			},
		},
	}

	// JSON 직렬화
	jsonData, err := json.Marshal(session)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// JSON 역직렬화
	var decodedSession SessionData
	err = json.Unmarshal(jsonData, &decodedSession)
	assert.NoError(t, err)

	// 검증
	assert.Equal(t, session.ID, decodedSession.ID)
	assert.Equal(t, session.Source, decodedSession.Source)
	assert.Equal(t, session.Title, decodedSession.Title)
	assert.Len(t, decodedSession.Messages, 1)
	assert.Equal(t, session.Messages[0].ID, decodedSession.Messages[0].ID)
	assert.Equal(t, session.Messages[0].Role, decodedSession.Messages[0].Role)
	assert.Equal(t, session.Messages[0].Content, decodedSession.Messages[0].Content)
	assert.Len(t, decodedSession.Files, 1)
	assert.Equal(t, session.Files[0].Path, decodedSession.Files[0].Path)
	assert.Len(t, decodedSession.Commands, 1)
	assert.Equal(t, session.Commands[0].Command, decodedSession.Commands[0].Command)
}

func TestMessage_ValidRoles(t *testing.T) {
	validRoles := []string{"user", "assistant", "system"}
	now := time.Now()

	for _, role := range validRoles {
		t.Run("role_"+role, func(t *testing.T) {
			message := Message{
				ID:        "msg-test",
				Role:      role,
				Content:   "Test content",
				Timestamp: now,
			}

			jsonData, err := json.Marshal(message)
			assert.NoError(t, err)

			var decodedMessage Message
			err = json.Unmarshal(jsonData, &decodedMessage)
			assert.NoError(t, err)
			assert.Equal(t, role, decodedMessage.Role)
		})
	}
}

func TestFileReference_BasicFields(t *testing.T) {
	now := time.Now()
	
	file := FileReference{
		Path:        "/path/to/file.go",
		Name:        "file.go",
		Size:        2048,
		ModTime:     now,
		ContentType: "text/plain",
		Hash:        "sha256:abc123",
	}

	assert.Equal(t, "/path/to/file.go", file.Path)
	assert.Equal(t, "file.go", file.Name)
	assert.Equal(t, int64(2048), file.Size)
	assert.Equal(t, now, file.ModTime)
	assert.Equal(t, "text/plain", file.ContentType)
	assert.Equal(t, "sha256:abc123", file.Hash)

	// JSON 직렬화/역직렬화 테스트
	jsonData, err := json.Marshal(file)
	assert.NoError(t, err)

	var decodedFile FileReference
	err = json.Unmarshal(jsonData, &decodedFile)
	assert.NoError(t, err)
	assert.Equal(t, file.Path, decodedFile.Path)
	assert.Equal(t, file.Size, decodedFile.Size)
}

func TestCommand_SuccessfulExecution(t *testing.T) {
	now := time.Now()
	
	cmd := Command{
		ID:        "cmd-success",
		Command:   "echo",
		Args:      []string{"hello", "world"},
		Output:    "hello world\n",
		Error:     "",
		ExitCode:  0,
		Timestamp: now,
		Duration:  100 * time.Millisecond,
		Environment: map[string]string{
			"PATH": "/usr/bin:/bin",
			"HOME": "/home/user",
		},
	}

	assert.Equal(t, 0, cmd.ExitCode)
	assert.Empty(t, cmd.Error)
	assert.NotEmpty(t, cmd.Output)
	assert.Equal(t, 2, len(cmd.Args))
}

func TestCommand_FailedExecution(t *testing.T) {
	now := time.Now()
	
	cmd := Command{
		ID:        "cmd-failed",
		Command:   "invalid-command",
		Args:      []string{},
		Output:    "",
		Error:     "command not found",
		ExitCode:  127,
		Timestamp: now,
		Duration:  10 * time.Millisecond,
	}

	assert.NotEqual(t, 0, cmd.ExitCode)
	assert.NotEmpty(t, cmd.Error)
	assert.Empty(t, cmd.Output)
}

func TestDateRange_Validation(t *testing.T) {
	tests := []struct {
		name        string
		dateRange   DateRange
		description string
	}{
		{
			name: "valid range",
			dateRange: DateRange{
				Start: time.Now().Add(-24 * time.Hour),
				End:   time.Now(),
			},
			description: "Start should be before End",
		},
		{
			name: "same date range",
			dateRange: DateRange{
				Start: time.Now(),
				End:   time.Now(),
			},
			description: "Start and End can be the same",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// JSON 직렬화/역직렬화 테스트
			jsonData, err := json.Marshal(tt.dateRange)
			assert.NoError(t, err, tt.description)

			var decodedRange DateRange
			err = json.Unmarshal(jsonData, &decodedRange)
			assert.NoError(t, err, tt.description)
			
			// 시간이 정확히 보존되는지 확인 (나노초까지)
			assert.True(t, tt.dateRange.Start.Equal(decodedRange.Start), tt.description)
			assert.True(t, tt.dateRange.End.Equal(decodedRange.End), tt.description)
		})
	}
}

func TestCollectionConfig_DefaultValues(t *testing.T) {
	config := CollectionConfig{
		Sources:         []CollectionSource{SourceClaudeCode, SourceGeminiCLI},
		IncludeFiles:    true,
		IncludeCommands: false,
		OutputPath:      "./output.md",
		Template:        "default",
	}

	assert.Len(t, config.Sources, 2)
	assert.Contains(t, config.Sources, SourceClaudeCode)
	assert.Contains(t, config.Sources, SourceGeminiCLI)
	assert.True(t, config.IncludeFiles)
	assert.False(t, config.IncludeCommands)
	assert.Equal(t, "./output.md", config.OutputPath)
	assert.Equal(t, "default", config.Template)
}

func TestCollectionConfig_WithDateRange(t *testing.T) {
	start := time.Now().Add(-7 * 24 * time.Hour)
	end := time.Now()
	
	config := CollectionConfig{
		Sources: []CollectionSource{SourceClaudeCode},
		DateRange: &DateRange{
			Start: start,
			End:   end,
		},
	}

	assert.NotNil(t, config.DateRange)
	assert.True(t, config.DateRange.Start.Equal(start))
	assert.True(t, config.DateRange.End.Equal(end))

	// JSON 직렬화 테스트
	jsonData, err := json.Marshal(config)
	assert.NoError(t, err)

	var decodedConfig CollectionConfig
	err = json.Unmarshal(jsonData, &decodedConfig)
	assert.NoError(t, err)
	assert.NotNil(t, decodedConfig.DateRange)
}

func TestExportConfig_AllFieldsSet(t *testing.T) {
	config := ExportConfig{
		Template:          "comprehensive",
		OutputPath:        "./exports/summary.md",
		IncludeMetadata:   true,
		IncludeTimestamps: true,
		FormatCodeBlocks:  true,
		GenerateTOC:       true,
		CustomFields: map[string]string{
			"author":  "AI Assistant",
			"version": "1.0.0",
		},
	}

	assert.Equal(t, "comprehensive", config.Template)
	assert.Equal(t, "./exports/summary.md", config.OutputPath)
	assert.True(t, config.IncludeMetadata)
	assert.True(t, config.IncludeTimestamps)
	assert.True(t, config.FormatCodeBlocks)
	assert.True(t, config.GenerateTOC)
	assert.Len(t, config.CustomFields, 2)
	assert.Equal(t, "AI Assistant", config.CustomFields["author"])
}

func TestCollectionResult_EmptyAndPopulated(t *testing.T) {
	now := time.Now()

	// 빈 결과
	emptyResult := CollectionResult{
		Sessions:    []SessionData{},
		TotalCount:  0,
		Sources:     []CollectionSource{},
		CollectedAt: now,
		Duration:    0,
		Errors:      []string{},
	}

	assert.Empty(t, emptyResult.Sessions)
	assert.Zero(t, emptyResult.TotalCount)
	assert.Empty(t, emptyResult.Sources)
	assert.Empty(t, emptyResult.Errors)

	// 데이터가 있는 결과
	session := SessionData{
		ID:        "test-session",
		Source:    SourceClaudeCode,
		Timestamp: now,
		Title:     "Test",
		Messages:  []Message{},
	}

	populatedResult := CollectionResult{
		Sessions:    []SessionData{session},
		TotalCount:  1,
		Sources:     []CollectionSource{SourceClaudeCode},
		CollectedAt: now,
		Duration:    time.Second,
		Errors:      []string{"minor warning"},
	}

	assert.Len(t, populatedResult.Sessions, 1)
	assert.Equal(t, 1, populatedResult.TotalCount)
	assert.Len(t, populatedResult.Sources, 1)
	assert.Len(t, populatedResult.Errors, 1)
}

func TestGeminiCollaboration_BasicFields(t *testing.T) {
	now := time.Now()
	
	collab := GeminiCollaboration{
		SessionID:   "gemini-session-123",
		ReviewType:  "code_review",
		RequestText: "다음 코드를 검토해주세요",
		Response:    "코드가 잘 작성되었습니다",
		Suggestions: []string{"주석 추가", "테스트 코드 작성"},
		Priority:    "high",
		Timestamp:   now,
		Metadata: map[string]string{
			"language": "go",
			"context":  "backend",
		},
	}

	assert.Equal(t, "gemini-session-123", collab.SessionID)
	assert.Equal(t, "code_review", collab.ReviewType)
	assert.NotEmpty(t, collab.RequestText)
	assert.NotEmpty(t, collab.Response)
	assert.Len(t, collab.Suggestions, 2)
	assert.Equal(t, "high", collab.Priority)
	assert.Len(t, collab.Metadata, 2)

	// JSON 직렬화 테스트
	jsonData, err := json.Marshal(collab)
	assert.NoError(t, err)

	var decodedCollab GeminiCollaboration
	err = json.Unmarshal(jsonData, &decodedCollab)
	assert.NoError(t, err)
	assert.Equal(t, collab.SessionID, decodedCollab.SessionID)
	assert.Equal(t, len(collab.Suggestions), len(decodedCollab.Suggestions))
}

// 벤치마크 테스트
func BenchmarkSessionDataJSONMarshal(b *testing.B) {
	now := time.Now()
	session := SessionData{
		ID:        "benchmark-session",
		Source:    SourceClaudeCode,
		Timestamp: now,
		Title:     "Benchmark Test",
		Messages: make([]Message, 100),
		Metadata:  map[string]string{"test": "benchmark"},
	}

	// 100개의 메시지로 채우기
	for i := 0; i < 100; i++ {
		session.Messages[i] = Message{
			ID:        "msg-" + string(rune(i)),
			Role:      "user",
			Content:   "Test message content",
			Timestamp: now,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(session)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSessionDataJSONUnmarshal(b *testing.B) {
	now := time.Now()
	session := SessionData{
		ID:        "benchmark-session",
		Source:    SourceClaudeCode,
		Timestamp: now,
		Title:     "Benchmark Test",
		Messages:  make([]Message, 100),
		Metadata:  map[string]string{"test": "benchmark"},
	}

	for i := 0; i < 100; i++ {
		session.Messages[i] = Message{
			ID:        "msg-" + string(rune(i)),
			Role:      "user",
			Content:   "Test message content",
			Timestamp: now,
		}
	}

	jsonData, _ := json.Marshal(session)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var decodedSession SessionData
		err := json.Unmarshal(jsonData, &decodedSession)
		if err != nil {
			b.Fatal(err)
		}
	}
}