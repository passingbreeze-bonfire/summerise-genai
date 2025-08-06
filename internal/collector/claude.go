package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"summerise-genai/internal/config"
	"summerise-genai/pkg/models"
)

// ClaudeCodeCollector는 Claude Code 데이터 수집기를 나타냅니다
type ClaudeCodeCollector struct {
	config config.CLIToolConfig
}

// NewClaudeCodeCollector는 새로운 Claude Code 데이터 수집기를 생성합니다
func NewClaudeCodeCollector(cfg config.CLIToolConfig) *ClaudeCodeCollector {
	return &ClaudeCodeCollector{
		config: cfg,
	}
}

// Collect는 Claude Code에서 세션 데이터를 수집합니다 (인터페이스 호환)
func (c *ClaudeCodeCollector) Collect(ctx context.Context, collectConfig *models.CollectionConfig) ([]models.SessionData, error) {
	// context 취소 확인
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	var sessions []models.SessionData

	// 설정 디렉토리 확장
	configDir, err := config.ExpandPath(c.config.ConfigDir)
	if err != nil {
		return nil, fmt.Errorf("설정 디렉토리 경로 확장 실패: %w", err)
	}

	// Claude Code 설정 디렉토리 존재 여부 확인
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("Claude Code 설정 디렉토리가 존재하지 않습니다: %s", configDir)
	}

	// 히스토리 파일에서 세션 수집
	if c.config.HistoryFile != "" {
		historySessions, err := c.collectFromHistory(ctx, collectConfig)
		if err != nil {
			// 히스토리 파일이 없어도 계속 진행
			fmt.Printf("경고: 히스토리 파일 수집 실패: %v\n", err)
		} else {
			sessions = append(sessions, historySessions...)
		}
	}

	// context 취소 확인
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// 세션 디렉토리에서 개별 세션 파일 수집
	if c.config.SessionDir != "" {
		sessionSessions, err := c.collectFromSessionDir(ctx, collectConfig)
		if err != nil {
			// 세션 디렉토리가 없어도 계속 진행
			fmt.Printf("경고: 세션 디렉토리 수집 실패: %v\n", err)
		} else {
			sessions = append(sessions, sessionSessions...)
		}
	}

	// 날짜 필터링
	if collectConfig.DateRange != nil {
		sessions = c.filterByDateRange(sessions, collectConfig.DateRange)
	}

	return sessions, nil
}

// GetSource는 이 수집기가 처리하는 소스 타입을 반환합니다
func (c *ClaudeCodeCollector) GetSource() models.CollectionSource {
	return models.SourceClaudeCode
}

// Validate는 수집기 설정이 유효한지 검증합니다
func (c *ClaudeCodeCollector) Validate() error {
	if c.config.ConfigDir == "" {
		return fmt.Errorf("설정 디렉토리가 지정되지 않았습니다")
	}

	// 경로 확장 시도
	configDir, err := config.ExpandPath(c.config.ConfigDir)
	if err != nil {
		return fmt.Errorf("설정 디렉토리 경로 확장 실패: %w", err)
	}

	// 디렉토리 존재 여부 확인
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return fmt.Errorf("설정 디렉토리가 존재하지 않습니다: %s", configDir)
	}

	return nil
}

// GetSupportedFormats는 수집기가 지원하는 데이터 형식들을 반환합니다
func (c *ClaudeCodeCollector) GetSupportedFormats() []string {
	return []string{"json", "text"}
}

// collectFromHistory는 히스토리 파일에서 세션을 수집합니다
func (c *ClaudeCodeCollector) collectFromHistory(ctx context.Context, collectConfig *models.CollectionConfig) ([]models.SessionData, error) {
	// context 취소 확인
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	historyPath, err := config.ExpandPath(c.config.HistoryFile)
	if err != nil {
		return nil, fmt.Errorf("히스토리 파일 경로 확장 실패: %w", err)
	}

	// 파일 존재 여부 확인
	if _, err := os.Stat(historyPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("히스토리 파일이 존재하지 않습니다: %s", historyPath)
	}

	// 파일 읽기
	data, err := os.ReadFile(historyPath)
	if err != nil {
		return nil, fmt.Errorf("히스토리 파일 읽기 실패: %w", err)
	}

	// context 취소 확인
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// JSON 구조 추정 및 파싱
	var historyData map[string]interface{}
	if err := json.Unmarshal(data, &historyData); err != nil {
		return nil, fmt.Errorf("히스토리 파일 JSON 파싱 실패: %w", err)
	}

	// 세션 데이터 추출 및 변환
	sessions := c.parseHistoryData(historyData)

	return sessions, nil
}

// collectFromSessionDir는 세션 디렉토리에서 개별 세션 파일들을 수집합니다
func (c *ClaudeCodeCollector) collectFromSessionDir(ctx context.Context, collectConfig *models.CollectionConfig) ([]models.SessionData, error) {
	sessionDir, err := config.ExpandPath(c.config.SessionDir)
	if err != nil {
		return nil, fmt.Errorf("세션 디렉토리 경로 확장 실패: %w", err)
	}

	// 디렉토리 존재 여부 확인
	if _, err := os.Stat(sessionDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("세션 디렉토리가 존재하지 않습니다: %s", sessionDir)
	}

	var sessions []models.SessionData

	// 디렉토리 순회하여 세션 파일 찾기
	err = filepath.Walk(sessionDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// context 취소 확인
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 디렉토리는 건너뛰기
		if info.IsDir() {
			return nil
		}

		// 파일 패턴 매칭
		if !c.matchesIncludePattern(path) {
			return nil
		}

		if c.matchesExcludePattern(path) {
			return nil
		}

		// 세션 파일 파싱
		sessionData, err := c.parseSessionFile(path)
		if err != nil {
			// 개별 파일 파싱 실패는 로그만 남기고 계속 진행
			fmt.Printf("세션 파일 파싱 실패 (건너뜀): %s - %v\n", path, err)
			return nil
		}

		if sessionData != nil {
			sessions = append(sessions, *sessionData)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("세션 디렉토리 순회 실패: %w", err)
	}

	return sessions, nil
}

// parseHistoryData는 히스토리 데이터를 파싱하여 세션 데이터로 변환합니다
func (c *ClaudeCodeCollector) parseHistoryData(historyData map[string]interface{}) []models.SessionData {
	var sessions []models.SessionData

	// 히스토리 데이터 구조를 추정하고 파싱
	// 실제 Claude Code의 히스토리 형식에 맞게 조정 필요
	
	if sessionsData, ok := historyData["sessions"].([]interface{}); ok {
		for _, sessionInterface := range sessionsData {
			if sessionMap, ok := sessionInterface.(map[string]interface{}); ok {
				session := c.parseSessionMap(sessionMap)
				if session != nil {
					sessions = append(sessions, *session)
				}
			}
		}
	}

	// 대체 구조 - conversations, chats 등의 키도 확인
	alternativeKeys := []string{"conversations", "chats", "history", "data"}
	for _, key := range alternativeKeys {
		if data, ok := historyData[key].([]interface{}); ok {
			for _, item := range data {
				if itemMap, ok := item.(map[string]interface{}); ok {
					session := c.parseSessionMap(itemMap)
					if session != nil {
						sessions = append(sessions, *session)
					}
				}
			}
		}
	}

	return sessions
}

// parseSessionFile은 개별 세션 파일을 파싱합니다
func (c *ClaudeCodeCollector) parseSessionFile(filePath string) (*models.SessionData, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("파일 읽기 실패: %w", err)
	}

	// JSON 파싱 시도
	var sessionData map[string]interface{}
	if err := json.Unmarshal(data, &sessionData); err != nil {
		// JSON이 아닌 경우 텍스트 파일로 처리
		return c.parseTextSession(filePath, string(data))
	}

	return c.parseSessionMap(sessionData), nil
}

// parseSessionMap은 세션 맵 데이터를 모델로 변환합니다
func (c *ClaudeCodeCollector) parseSessionMap(sessionMap map[string]interface{}) *models.SessionData {
	session := &models.SessionData{
		Source:   models.SourceClaudeCode,
		Messages: make([]models.Message, 0),
		Commands: make([]models.Command, 0),
		Files:    make([]models.FileReference, 0),
		Metadata: make(map[string]string),
	}

	// ID 추출
	if id, ok := sessionMap["id"].(string); ok {
		session.ID = id
	} else {
		session.ID = fmt.Sprintf("claude-session-%d", time.Now().UnixNano())
	}

	// 타임스탬프 추출
	if timestamp, ok := sessionMap["timestamp"].(string); ok {
		if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
			session.Timestamp = t
		}
	} else if createdAt, ok := sessionMap["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			session.Timestamp = t
		}
	}

	if session.Timestamp.IsZero() {
		session.Timestamp = time.Now()
	}

	// 제목 추출
	if title, ok := sessionMap["title"].(string); ok {
		session.Title = title
	} else if name, ok := sessionMap["name"].(string); ok {
		session.Title = name
	}

	// 메시지 추출
	if messages, ok := sessionMap["messages"].([]interface{}); ok {
		for i, msgInterface := range messages {
			if msgMap, ok := msgInterface.(map[string]interface{}); ok {
				message := c.parseMessage(msgMap, i)
				session.Messages = append(session.Messages, message)
			}
		}
	}

	// 메타데이터 추출
	if metadata, ok := sessionMap["metadata"].(map[string]interface{}); ok {
		for k, v := range metadata {
			if str, ok := v.(string); ok {
				session.Metadata[k] = str
			} else {
				session.Metadata[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	return session
}

// parseMessage는 메시지 데이터를 파싱합니다
func (c *ClaudeCodeCollector) parseMessage(msgMap map[string]interface{}, index int) models.Message {
	message := models.Message{
		ID:       fmt.Sprintf("msg-%d", index+1),
		Metadata: make(map[string]string),
	}

	// ID 추출
	if id, ok := msgMap["id"].(string); ok {
		message.ID = id
	}

	// Role 추출
	if role, ok := msgMap["role"].(string); ok {
		message.Role = role
	} else if sender, ok := msgMap["sender"].(string); ok {
		message.Role = sender
	} else {
		message.Role = "unknown"
	}

	// Content 추출
	if content, ok := msgMap["content"].(string); ok {
		message.Content = content
	} else if text, ok := msgMap["text"].(string); ok {
		message.Content = text
	} else if body, ok := msgMap["body"].(string); ok {
		message.Content = body
	}

	// 타임스탬프 추출
	if timestamp, ok := msgMap["timestamp"].(string); ok {
		if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
			message.Timestamp = t
		}
	}

	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}

	return message
}

// parseTextSession은 텍스트 파일을 세션으로 파싱합니다
func (c *ClaudeCodeCollector) parseTextSession(filePath, content string) (*models.SessionData, error) {
	session := &models.SessionData{
		ID:        fmt.Sprintf("claude-text-session-%d", time.Now().UnixNano()),
		Source:    models.SourceClaudeCode,
		Title:     filepath.Base(filePath),
		Timestamp: time.Now(),
		Messages:  make([]models.Message, 0),
		Metadata:  make(map[string]string),
	}

	// 파일 수정 시간을 타임스탬프로 사용
	if info, err := os.Stat(filePath); err == nil {
		session.Timestamp = info.ModTime()
	}

	// 텍스트 내용을 하나의 메시지로 처리
	message := models.Message{
		ID:        "msg-1",
		Role:      "content",
		Content:   content,
		Timestamp: session.Timestamp,
	}

	session.Messages = append(session.Messages, message)
	session.Metadata["file_path"] = filePath
	session.Metadata["file_type"] = "text"

	return session, nil
}

// matchesIncludePattern은 파일이 포함 패턴과 매칭되는지 확인합니다
func (c *ClaudeCodeCollector) matchesIncludePattern(filePath string) bool {
	if len(c.config.IncludePatterns) == 0 {
		return true
	}

	fileName := filepath.Base(filePath)
	for _, pattern := range c.config.IncludePatterns {
		if matched, _ := filepath.Match(pattern, fileName); matched {
			return true
		}
	}

	return false
}

// matchesExcludePattern은 파일이 제외 패턴과 매칭되는지 확인합니다
func (c *ClaudeCodeCollector) matchesExcludePattern(filePath string) bool {
	if len(c.config.ExcludePatterns) == 0 {
		return false
	}

	fileName := filepath.Base(filePath)
	for _, pattern := range c.config.ExcludePatterns {
		if matched, _ := filepath.Match(pattern, fileName); matched {
			return true
		}
	}

	return false
}

// filterByDateRange는 날짜 범위로 세션을 필터링합니다
func (c *ClaudeCodeCollector) filterByDateRange(sessions []models.SessionData, dateRange *models.DateRange) []models.SessionData {
	if dateRange == nil {
		return sessions
	}

	var filtered []models.SessionData
	for _, session := range sessions {
		if !dateRange.Start.IsZero() && session.Timestamp.Before(dateRange.Start) {
			continue
		}
		if !dateRange.End.IsZero() && session.Timestamp.After(dateRange.End) {
			continue
		}
		filtered = append(filtered, session)
	}

	return filtered
}