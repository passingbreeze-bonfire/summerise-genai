package collector

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"ssamai/internal/config"
	"ssamai/pkg/models"
)

// init 함수는 패키지 로드 시 자동으로 호출되어 팩토리에 등록합니다.
func init() {
	Register(models.SourceGeminiCLI, func(configInterface interface{}) models.Collector {
		cfg, ok := configInterface.(config.CLIToolConfig)
		if !ok {
			// 기본 설정으로 생성
			cfg = config.CLIToolConfig{}
		}
		return NewImprovedGeminiCLICollector(cfg)
	})
}

const (
	// 파일 처리 관련 상수
	maxFileSize        = 100 * 1024 * 1024 // 100MB
	bufferSize         = 64 * 1024         // 64KB
	maxWorkers         = 10                // 최대 워커 수
	defaultTimeout     = 30 * time.Second  // 기본 타임아웃
	maxJSONDepth       = 100               // JSON 파싱 최대 깊이
	maxMessagesPerFile = 10000             // 파일당 최대 메시지 수
)

// GeminiCLICollectorInterface는 테스트 가능성을 위한 인터페이스
type GeminiCLICollectorInterface interface {
	Collect(ctx context.Context, collectConfig *models.CollectionConfig) ([]models.SessionData, error)
	GetSource() models.CollectionSource
	Validate() error
	GetSupportedFormats() []string
}

// FileReader는 파일 읽기를 위한 인터페이스 (테스트용)
type FileReader interface {
	ReadFile(filename string) ([]byte, error)
	Stat(filename string) (os.FileInfo, error)
	WalkDir(root string, fn fs.WalkDirFunc) error
}

// DefaultFileReader는 FileReader의 기본 구현
type DefaultFileReader struct{}

func (r *DefaultFileReader) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

func (r *DefaultFileReader) Stat(filename string) (os.FileInfo, error) {
	return os.Stat(filename)
}

func (r *DefaultFileReader) WalkDir(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, fn)
}

// ImprovedGeminiCLICollector는 개선된 Gemini CLI 수집기
type ImprovedGeminiCLICollector struct {
	config     config.CLIToolConfig
	fileReader FileReader
	logger     Logger // 추가된 로거 인터페이스
}

// Logger는 로깅을 위한 인터페이스
type Logger interface {
	Printf(format string, v ...interface{})
	Errorf(format string, v ...interface{})
	Warnf(format string, v ...interface{})
}

// DefaultLogger는 Logger의 기본 구현
type DefaultLogger struct{}

func (l *DefaultLogger) Printf(format string, v ...interface{}) {
	fmt.Printf(format, v...)
}

func (l *DefaultLogger) Errorf(format string, v ...interface{}) {
	fmt.Printf("ERROR: "+format, v...)
}

func (l *DefaultLogger) Warnf(format string, v ...interface{}) {
	fmt.Printf("WARN: "+format, v...)
}

// NewImprovedGeminiCLICollector는 개선된 collector 생성자
func NewImprovedGeminiCLICollector(config config.CLIToolConfig) *ImprovedGeminiCLICollector {
	return &ImprovedGeminiCLICollector{
		config:     config,
		fileReader: &DefaultFileReader{},
		logger:     &DefaultLogger{},
	}
}

// WithFileReader는 테스트용 파일 리더 의존성 주입
func (g *ImprovedGeminiCLICollector) WithFileReader(reader FileReader) *ImprovedGeminiCLICollector {
	g.fileReader = reader
	return g
}

// WithLogger는 로거 의존성 주입
func (g *ImprovedGeminiCLICollector) WithLogger(logger Logger) *ImprovedGeminiCLICollector {
	g.logger = logger
	return g
}

// Collect는 컨텍스트 관리와 에러 처리가 개선된 수집 메서드
func (g *ImprovedGeminiCLICollector) Collect(ctx context.Context, collectConfig *models.CollectionConfig) ([]models.SessionData, error) {
	if collectConfig == nil {
		return nil, fmt.Errorf("collection config is nil")
	}

	// 타임아웃이 설정된 컨텍스트 생성
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	// 설정 디렉토리 검증
	if err := g.validateConfigDirectory(); err != nil {
		return nil, fmt.Errorf("config directory validation failed: %w", err)
	}

	var allSessions []models.SessionData
	var mu sync.Mutex
	var wg sync.WaitGroup
	errs := make([]error, 0)
	var errMu sync.Mutex

	// 에러 수집 함수
	addError := func(err error) {
		errMu.Lock()
		errs = append(errs, err)
		errMu.Unlock()
	}

	// 히스토리 파일 처리
	if g.config.HistoryFile != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sessions, err := g.collectFromHistoryWithRetry(ctx, collectConfig)
			if err != nil {
				addError(fmt.Errorf("history collection failed: %w", err))
				return
			}
			mu.Lock()
			allSessions = append(allSessions, sessions...)
			mu.Unlock()
		}()
	}

	// 세션 디렉토리 처리
	if g.config.SessionDir != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sessions, err := g.collectFromSessionDirConcurrent(ctx, collectConfig)
			if err != nil {
				addError(fmt.Errorf("session directory collection failed: %w", err))
				return
			}
			mu.Lock()
			allSessions = append(allSessions, sessions...)
			mu.Unlock()
		}()
	}

	wg.Wait()

	// 에러가 있으면 경고 로그 출력
	for _, err := range errs {
		g.logger.Warnf("Collection warning: %v\n", err)
	}

	// 날짜 필터링
	if collectConfig.DateRange != nil {
		allSessions = g.filterByDateRange(allSessions, collectConfig.DateRange)
	}

	return allSessions, nil
}

// validateConfigDirectory는 설정 디렉토리 유효성 검사
func (g *ImprovedGeminiCLICollector) validateConfigDirectory() error {
	configDir, err := config.ExpandPath(g.config.ConfigDir)
	if err != nil {
		return fmt.Errorf("failed to expand config directory path: %w", err)
	}

	if _, err := g.fileReader.Stat(configDir); os.IsNotExist(err) {
		return fmt.Errorf("gemini CLI config directory does not exist: %s", configDir)
	}

	return nil
}

// collectFromHistoryWithRetry는 재시도 로직이 있는 히스토리 수집
func (g *ImprovedGeminiCLICollector) collectFromHistoryWithRetry(ctx context.Context, collectConfig *models.CollectionConfig) ([]models.SessionData, error) {
	historyPath, err := config.ExpandPath(g.config.HistoryFile)
	if err != nil {
		return nil, fmt.Errorf("failed to expand history file path: %w", err)
	}

	// 파일 크기 확인
	info, err := g.fileReader.Stat(historyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat history file: %w", err)
	}

	if info.Size() > maxFileSize {
		return nil, fmt.Errorf("history file too large: %d bytes (max: %d)", info.Size(), maxFileSize)
	}

	// 스트리밍 방식으로 파일 읽기
	return g.parseHistoryFileStreaming(ctx, historyPath, collectConfig)
}

// parseHistoryFileStreaming은 메모리 효율적인 히스토리 파일 파싱
func (g *ImprovedGeminiCLICollector) parseHistoryFileStreaming(ctx context.Context, filePath string, collectConfig *models.CollectionConfig) ([]models.SessionData, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open history file: %w", err)
	}
	defer file.Close()

	var sessions []models.SessionData
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, bufferSize), bufferSize)
	
	lineNum := 0
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		session, err := g.parseHistoryLine(line, lineNum)
		if err != nil {
			g.logger.Warnf("Failed to parse history line %d: %v", lineNum, err)
			continue
		}

		if session != nil {
			sessions = append(sessions, *session)
		}

		// 메모리 사용량 제한
		if len(sessions) >= maxMessagesPerFile {
			g.logger.Warnf("Reached maximum messages per file limit: %d", maxMessagesPerFile)
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading history file: %w", err)
	}

	return sessions, nil
}

// parseHistoryLine은 안전한 히스토리 라인 파싱
func (g *ImprovedGeminiCLICollector) parseHistoryLine(line string, lineNum int) (*models.SessionData, error) {
	// JSON 파싱 시도
	if strings.HasPrefix(line, "{") {
		return g.parseJSONHistoryEntry(line, lineNum)
	}

	// 텍스트로 처리
	return g.parseTextHistoryEntry(line, lineNum), nil
}

// parseJSONHistoryEntry는 안전한 JSON 히스토리 엔트리 파싱
func (g *ImprovedGeminiCLICollector) parseJSONHistoryEntry(line string, lineNum int) (*models.SessionData, error) {
	var entry GeminiHistoryEntry
	decoder := json.NewDecoder(strings.NewReader(line))
	decoder.DisallowUnknownFields() // 알 수 없는 필드 거부

	if err := decoder.Decode(&entry); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return g.convertHistoryEntryToSession(entry, lineNum), nil
}

// GeminiHistoryEntry는 Gemini CLI 히스토리 엔트리 구조체
type GeminiHistoryEntry struct {
	ID        string                 `json:"id"`
	Command   string                 `json:"command"`
	Prompt    string                 `json:"prompt"`
	Response  string                 `json:"response"`
	Timestamp string                 `json:"timestamp"`
	Model     string                 `json:"model"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// GeminiSessionData는 Gemini CLI 세션 데이터 구조체
type GeminiSessionData struct {
	ID           string                   `json:"id"`
	Title        string                   `json:"title"`
	CreatedAt    string                   `json:"created_at"`
	UpdatedAt    string                   `json:"updated_at"`
	Model        string                   `json:"model"`
	Messages     []GeminiMessage          `json:"messages"`
	Metadata     map[string]interface{}   `json:"metadata"`
	Settings     *GeminiSessionSettings   `json:"settings"`
}

// GeminiMessage는 Gemini CLI 메시지 구조체
type GeminiMessage struct {
	ID        string                 `json:"id"`
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	Parts     []GeminiMessagePart    `json:"parts"`
	Timestamp string                 `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// GeminiMessagePart는 Gemini 메시지 파트 구조체
type GeminiMessagePart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// GeminiSessionSettings는 Gemini 세션 설정 구조체
type GeminiSessionSettings struct {
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
}

// convertHistoryEntryToSession은 히스토리 엔트리를 세션으로 변환
func (g *ImprovedGeminiCLICollector) convertHistoryEntryToSession(entry GeminiHistoryEntry, index int) *models.SessionData {
	sessionID := entry.ID
	if sessionID == "" {
		sessionID = fmt.Sprintf("gemini-cli-history-%d", index)
	}

	session := &models.SessionData{
		ID:        sessionID,
		Source:    models.SourceGeminiCLI,
		Timestamp: time.Now(),
		Title:     g.extractTitleFromPrompt(entry.Prompt),
		Messages:  make([]models.Message, 0, 2),
		Metadata:  make(map[string]string),
	}

	// 타임스탬프 파싱
	if entry.Timestamp != "" {
		if timestamp, err := time.Parse(time.RFC3339, entry.Timestamp); err == nil {
			session.Timestamp = timestamp
		}
	}

	// 메타데이터 설정
	session.Metadata["model"] = entry.Model
	session.Metadata["command"] = entry.Command
	session.Metadata["source_type"] = "gemini_cli_history"

	// 사용자 메시지 추가
	if entry.Prompt != "" {
		userMsg := models.Message{
			ID:        fmt.Sprintf("%s-user", sessionID),
			Role:      "user",
			Content:   entry.Prompt,
			Timestamp: session.Timestamp,
			Metadata:  make(map[string]string),
		}
		session.Messages = append(session.Messages, userMsg)
	}

	// 어시스턴트 메시지 추가
	if entry.Response != "" {
		assistantMsg := models.Message{
			ID:        fmt.Sprintf("%s-assistant", sessionID),
			Role:      "assistant",
			Content:   entry.Response,
			Timestamp: session.Timestamp.Add(1 * time.Second),
			Metadata:  make(map[string]string),
		}
		session.Messages = append(session.Messages, assistantMsg)
	}

	return session
}

// parseTextHistoryEntry는 텍스트 히스토리 엔트리 파싱
func (g *ImprovedGeminiCLICollector) parseTextHistoryEntry(line string, lineNum int) *models.SessionData {
	if len(strings.TrimSpace(line)) == 0 {
		return nil
	}

	sessionID := fmt.Sprintf("gemini-cli-text-%d", lineNum)
	return &models.SessionData{
		ID:        sessionID,
		Source:    models.SourceGeminiCLI,
		Timestamp: time.Now(),
		Title:     "Gemini CLI History Entry",
		Messages: []models.Message{
			{
				ID:        fmt.Sprintf("%s-user", sessionID),
				Role:      "user",
				Content:   line,
				Timestamp: time.Now(),
				Metadata:  map[string]string{"source_type": "gemini_cli_text"},
			},
		},
		Metadata: map[string]string{
			"source_type":  "gemini_cli_history",
			"entry_number": fmt.Sprintf("%d", lineNum),
		},
	}
}

// collectFromSessionDirConcurrent는 동시성 처리가 개선된 세션 디렉토리 수집
func (g *ImprovedGeminiCLICollector) collectFromSessionDirConcurrent(ctx context.Context, collectConfig *models.CollectionConfig) ([]models.SessionData, error) {
	sessionDirPath, err := config.ExpandPath(g.config.SessionDir)
	if err != nil {
		return nil, fmt.Errorf("failed to expand session directory path: %w", err)
	}

	// 파일 목록 수집
	var filePaths []string
	err = g.fileReader.WalkDir(sessionDirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}

		filePaths = append(filePaths, path)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk session directory: %w", err)
	}

	// 워커 수 결정
	numWorkers := min(maxWorkers, len(filePaths), runtime.NumCPU())
	if numWorkers == 0 {
		return []models.SessionData{}, nil
	}

	// 채널 생성
	fileChan := make(chan string, len(filePaths))
	resultChan := make(chan *models.SessionData, len(filePaths))
	errorChan := make(chan error, len(filePaths))

	// 워커 시작
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go g.sessionFileWorker(ctx, &wg, fileChan, resultChan, errorChan, collectConfig)
	}

	// 파일 경로들을 채널에 전송
	go func() {
		defer close(fileChan)
		for _, path := range filePaths {
			select {
			case fileChan <- path:
			case <-ctx.Done():
				return
			}
		}
	}()

	// 워커들이 완료되면 채널들을 닫음
	go func() {
		wg.Wait()
		close(resultChan)
		close(errorChan)
	}()

	// 결과 수집
	var sessions []models.SessionData
	var errors []error

	for {
		select {
		case session, ok := <-resultChan:
			if !ok {
				resultChan = nil
			} else if session != nil {
				sessions = append(sessions, *session)
			}
		case err, ok := <-errorChan:
			if !ok {
				errorChan = nil
			} else if err != nil {
				errors = append(errors, err)
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		if resultChan == nil && errorChan == nil {
			break
		}
	}

	// 에러 로깅
	for _, err := range errors {
		g.logger.Warnf("Session file processing error: %v", err)
	}

	return sessions, nil
}

// sessionFileWorker는 세션 파일 처리 워커
func (g *ImprovedGeminiCLICollector) sessionFileWorker(ctx context.Context, wg *sync.WaitGroup, fileChan <-chan string, resultChan chan<- *models.SessionData, errorChan chan<- error, collectConfig *models.CollectionConfig) {
	defer wg.Done()

	for {
		select {
		case filePath, ok := <-fileChan:
			if !ok {
				return
			}

			session, err := g.parseSessionFileSafe(filePath, collectConfig)
			if err != nil {
				errorChan <- fmt.Errorf("failed to parse session file %s: %w", filePath, err)
				continue
			}

			resultChan <- session

		case <-ctx.Done():
			return
		}
	}
}

// parseSessionFileSafe는 안전한 세션 파일 파싱
func (g *ImprovedGeminiCLICollector) parseSessionFileSafe(path string, collectConfig *models.CollectionConfig) (*models.SessionData, error) {
	// 파일 크기 확인
	info, err := g.fileReader.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if info.Size() > maxFileSize {
		return nil, fmt.Errorf("file too large: %d bytes", info.Size())
	}

	// 파일 읽기
	data, err := g.fileReader.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// JSON 파싱
	var sessionData GeminiSessionData
	if err := json.Unmarshal(data, &sessionData); err != nil {
		// JSON 파싱 실패 시 텍스트로 처리
		return g.parseTextSession(string(data), path), nil
	}

	return g.convertGeminiSessionToModel(sessionData, path), nil
}

// convertGeminiSessionToModel은 Gemini 세션 데이터를 모델로 변환
func (g *ImprovedGeminiCLICollector) convertGeminiSessionToModel(geminiSession GeminiSessionData, filePath string) *models.SessionData {
	session := &models.SessionData{
		ID:        geminiSession.ID,
		Source:    models.SourceGeminiCLI,
		Timestamp: time.Now(),
		Title:     geminiSession.Title,
		Messages:  make([]models.Message, 0, len(geminiSession.Messages)),
		Metadata:  make(map[string]string),
	}

	// ID 설정
	if session.ID == "" {
		session.ID = fmt.Sprintf("gemini-cli-%s", filepath.Base(filePath))
	}

	// 타임스탬프 파싱
	if geminiSession.CreatedAt != "" {
		if timestamp, err := time.Parse(time.RFC3339, geminiSession.CreatedAt); err == nil {
			session.Timestamp = timestamp
		}
	}

	// 메타데이터 설정
	session.Metadata["file_path"] = filePath
	session.Metadata["model"] = geminiSession.Model
	session.Metadata["source_type"] = "gemini_cli_session"

	// 메시지 변환
	for _, geminiMsg := range geminiSession.Messages {
		msg := models.Message{
			ID:        geminiMsg.ID,
			Role:      geminiMsg.Role,
			Content:   g.extractContentFromGeminiMessage(geminiMsg),
			Timestamp: session.Timestamp,
			Metadata:  make(map[string]string),
		}

		// 메시지 타임스탬프 파싱
		if geminiMsg.Timestamp != "" {
			if msgTime, err := time.Parse(time.RFC3339, geminiMsg.Timestamp); err == nil {
				msg.Timestamp = msgTime
			}
		}

		session.Messages = append(session.Messages, msg)
	}

	return session
}

// extractContentFromGeminiMessage는 Gemini 메시지에서 컨텐츠 추출
func (g *ImprovedGeminiCLICollector) extractContentFromGeminiMessage(msg GeminiMessage) string {
	if msg.Content != "" {
		return msg.Content
	}

	// Parts에서 텍스트 추출
	var contents []string
	for _, part := range msg.Parts {
		if part.Type == "text" && part.Text != "" {
			contents = append(contents, part.Text)
		}
	}

	return strings.Join(contents, "\n")
}

// parseTextSession은 텍스트 세션 파싱
func (g *ImprovedGeminiCLICollector) parseTextSession(content string, path string) *models.SessionData {
	fileName := filepath.Base(path)
	sessionID := fmt.Sprintf("gemini-cli-text-%s", strings.TrimSuffix(fileName, filepath.Ext(fileName)))

	return &models.SessionData{
		ID:        sessionID,
		Source:    models.SourceGeminiCLI,
		Timestamp: time.Now(),
		Title:     fmt.Sprintf("Gemini CLI Session: %s", fileName),
		Messages: []models.Message{
			{
				ID:        fmt.Sprintf("%s-content", sessionID),
				Role:      "user",
				Content:   content,
				Timestamp: time.Now(),
				Metadata:  map[string]string{"source_type": "gemini_cli_text"},
			},
		},
		Metadata: map[string]string{
			"file_path":   path,
			"source_type": "gemini_cli_text",
		},
	}
}

// extractTitleFromPrompt는 프롬프트에서 제목 추출
func (g *ImprovedGeminiCLICollector) extractTitleFromPrompt(prompt string) string {
	if len(prompt) == 0 {
		return "Gemini CLI Session"
	}

	// 첫 줄만 사용
	lines := strings.Split(prompt, "\n")
	title := strings.TrimSpace(lines[0])

	// 길이 제한
	if len(title) > 100 {
		title = title[:97] + "..."
	}

	if title == "" {
		return "Gemini CLI Session"
	}

	return title
}

// filterByDateRange는 날짜 범위 필터링
func (g *ImprovedGeminiCLICollector) filterByDateRange(sessions []models.SessionData, dateRange *models.DateRange) []models.SessionData {
	if dateRange == nil {
		return sessions
	}

	filtered := make([]models.SessionData, 0, len(sessions))
	for _, session := range sessions {
		if g.isWithinDateRange(session.Timestamp, dateRange) {
			filtered = append(filtered, session)
		}
	}

	return filtered
}

// isWithinDateRange는 날짜가 범위 내에 있는지 확인
func (g *ImprovedGeminiCLICollector) isWithinDateRange(timestamp time.Time, dateRange *models.DateRange) bool {
	if !dateRange.Start.IsZero() && timestamp.Before(dateRange.Start) {
		return false
	}
	if !dateRange.End.IsZero() && timestamp.After(dateRange.End) {
		return false
	}
	return true
}

// GetSource는 소스 타입 반환
func (g *ImprovedGeminiCLICollector) GetSource() models.CollectionSource {
	return models.SourceGeminiCLI
}

// Validate는 설정 검증
func (g *ImprovedGeminiCLICollector) Validate() error {
	return g.validateConfigDirectory()
}

// GetSupportedFormats는 지원 형식 반환
func (g *ImprovedGeminiCLICollector) GetSupportedFormats() []string {
	return []string{"json", "text", "jsonl"}
}

// min은 정수의 최솟값 반환 (Go 1.21 이전 버전 호환)
func min(a ...int) int {
	if len(a) == 0 {
		return 0
	}
	result := a[0]
	for _, v := range a[1:] {
		if v < result {
			result = v
		}
	}
	return result
}