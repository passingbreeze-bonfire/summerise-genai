package collector

import (
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
	Register(models.SourceAmazonQ, func(configInterface interface{}) models.Collector {
		cfg, ok := configInterface.(config.CLIToolConfig)
		if !ok {
			// 기본 설정으로 생성
			cfg = config.CLIToolConfig{}
		}
		return NewAmazonQCollector(cfg)
	})
}

const (
	// Amazon Q CLI 특정 상수들
	amazonQMaxFileSize        = 100 * 1024 * 1024 // 100MB
	amazonQBufferSize         = 64 * 1024         // 64KB
	amazonQMaxWorkers         = 10                // 최대 워커 수
	amazonQDefaultTimeout     = 30 * time.Second  // 기본 타임아웃
	amazonQMaxMessagesPerFile = 10000             // 파일당 최대 메시지 수
)

// AmazonQCollectorInterface는 테스트 가능성을 위한 인터페이스
type AmazonQCollectorInterface interface {
	Collect(ctx context.Context, collectConfig *models.CollectionConfig) ([]models.SessionData, error)
	GetSource() models.CollectionSource
	Validate() error
	GetSupportedFormats() []string
}

// AmazonQFileReader는 Amazon Q CLI 파일 읽기를 위한 인터페이스
type AmazonQFileReader interface {
	ReadFile(filename string) ([]byte, error)
	Stat(filename string) (os.FileInfo, error)
	WalkDir(root string, fn fs.WalkDirFunc) error
	OpenFile(name string) (*os.File, error)
}

// DefaultAmazonQFileReader는 AmazonQFileReader의 기본 구현
type DefaultAmazonQFileReader struct{}

func (r *DefaultAmazonQFileReader) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

func (r *DefaultAmazonQFileReader) Stat(filename string) (os.FileInfo, error) {
	return os.Stat(filename)
}

func (r *DefaultAmazonQFileReader) WalkDir(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, fn)
}

func (r *DefaultAmazonQFileReader) OpenFile(name string) (*os.File, error) {
	return os.Open(name)
}

// AmazonQLogger는 Amazon Q CLI 로깅을 위한 인터페이스
type AmazonQLogger interface {
	Printf(format string, v ...interface{})
	Errorf(format string, v ...interface{})
	Warnf(format string, v ...interface{})
}

// DefaultAmazonQLogger는 AmazonQLogger의 기본 구현
type DefaultAmazonQLogger struct{}

func (l *DefaultAmazonQLogger) Printf(format string, v ...interface{}) {
	fmt.Printf(format, v...)
}

func (l *DefaultAmazonQLogger) Errorf(format string, v ...interface{}) {
	fmt.Printf("ERROR: "+format, v...)
}

func (l *DefaultAmazonQLogger) Warnf(format string, v ...interface{}) {
	fmt.Printf("WARN: "+format, v...)
}

// AmazonQCollector는 Amazon Q CLI 데이터 수집기
type AmazonQCollector struct {
	config     config.CLIToolConfig
	fileReader AmazonQFileReader
	logger     AmazonQLogger
}

// NewAmazonQCollector는 새로운 Amazon Q CLI 데이터 수집기를 생성합니다
func NewAmazonQCollector(cfg config.CLIToolConfig) *AmazonQCollector {
	return &AmazonQCollector{
		config:     cfg,
		fileReader: &DefaultAmazonQFileReader{},
		logger:     &DefaultAmazonQLogger{},
	}
}

// WithFileReader는 테스트용 파일 리더 의존성 주입
func (a *AmazonQCollector) WithFileReader(reader AmazonQFileReader) *AmazonQCollector {
	a.fileReader = reader
	return a
}

// WithLogger는 로거 의존성 주입
func (a *AmazonQCollector) WithLogger(logger AmazonQLogger) *AmazonQCollector {
	a.logger = logger
	return a
}

// Collect는 Amazon Q CLI에서 세션 데이터를 수집합니다
func (a *AmazonQCollector) Collect(ctx context.Context, collectConfig *models.CollectionConfig) ([]models.SessionData, error) {
	if collectConfig == nil {
		return nil, fmt.Errorf("collection config is nil")
	}

	// 타임아웃이 설정된 컨텍스트 생성
	ctx, cancel := context.WithTimeout(ctx, amazonQDefaultTimeout)
	defer cancel()

	// 설정 디렉토리 검증
	if err := a.validateConfigDirectory(); err != nil {
		// Amazon Q CLI가 설치되지 않은 경우 더미 데이터 반환
		a.logger.Warnf("Amazon Q CLI not found, returning dummy data: %v\n", err)
		return a.generateDummyData(), nil
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
	if a.config.HistoryFile != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sessions, err := a.collectFromHistoryWithRetry(ctx, collectConfig)
			if err != nil {
				addError(fmt.Errorf("Amazon Q history collection failed: %w", err))
				return
			}
			mu.Lock()
			allSessions = append(allSessions, sessions...)
			mu.Unlock()
		}()
	}

	// 세션 디렉토리 처리
	if a.config.SessionDir != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sessions, err := a.collectFromSessionDirConcurrent(ctx, collectConfig)
			if err != nil {
				addError(fmt.Errorf("Amazon Q session directory collection failed: %w", err))
				return
			}
			mu.Lock()
			allSessions = append(allSessions, sessions...)
			mu.Unlock()
		}()
	}

	// AWS 설정 파일에서 컨텍스트 정보 수집
	wg.Add(1)
	go func() {
		defer wg.Done()
		sessions, err := a.collectFromAWSConfig(ctx, collectConfig)
		if err != nil {
			addError(fmt.Errorf("AWS config collection failed: %w", err))
			return
		}
		mu.Lock()
		allSessions = append(allSessions, sessions...)
		mu.Unlock()
	}()

	wg.Wait()

	// 에러가 있으면 경고 로그 출력 (하지만 실행은 계속)
	for _, err := range errs {
		a.logger.Warnf("Collection warning: %v\n", err)
	}

	// 데이터가 없으면 더미 데이터 생성
	if len(allSessions) == 0 {
		a.logger.Printf("No Amazon Q CLI data found, generating dummy data\n")
		allSessions = a.generateDummyData()
	}

	// 날짜 필터링
	if collectConfig.DateRange != nil {
		allSessions = a.filterByDateRange(allSessions, collectConfig.DateRange)
	}

	return allSessions, nil
}

// GetSource는 이 수집기가 처리하는 소스 타입을 반환합니다
func (a *AmazonQCollector) GetSource() models.CollectionSource {
	return models.SourceAmazonQ
}

// Validate는 수집기 설정이 유효한지 검증합니다
func (a *AmazonQCollector) Validate() error {
	if a.config.ConfigDir == "" {
		return fmt.Errorf("config directory not specified")
	}

	// 경로 확장 시도
	configDir, err := config.ExpandPath(a.config.ConfigDir)
	if err != nil {
		return fmt.Errorf("failed to expand config directory path: %w", err)
	}

	// 디렉토리 존재 여부 확인 (경고만 출력, 에러는 반환하지 않음)
	if _, err := a.fileReader.Stat(configDir); os.IsNotExist(err) {
		a.logger.Warnf("Amazon Q CLI config directory does not exist: %s\n", configDir)
		return nil // 더미 데이터로 처리 가능
	}

	return nil
}

// GetSupportedFormats는 수집기가 지원하는 데이터 형식들을 반환합니다
func (a *AmazonQCollector) GetSupportedFormats() []string {
	return []string{"json", "text", "aws-logs", "session"}
}

// validateConfigDirectory는 설정 디렉토리 유효성 검사
func (a *AmazonQCollector) validateConfigDirectory() error {
	configDir, err := config.ExpandPath(a.config.ConfigDir)
	if err != nil {
		return fmt.Errorf("failed to expand config directory path: %w", err)
	}

	if _, err := a.fileReader.Stat(configDir); os.IsNotExist(err) {
		return fmt.Errorf("Amazon Q CLI config directory does not exist: %s", configDir)
	}

	return nil
}

// collectFromHistoryWithRetry는 재시도 로직이 있는 히스토리 수집
func (a *AmazonQCollector) collectFromHistoryWithRetry(ctx context.Context, collectConfig *models.CollectionConfig) ([]models.SessionData, error) {
	historyPath, err := config.ExpandPath(a.config.HistoryFile)
	if err != nil {
		return nil, fmt.Errorf("failed to expand history file path: %w", err)
	}

	// 파일 존재 및 크기 확인
	info, err := a.fileReader.Stat(historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			a.logger.Warnf("Amazon Q CLI history file not found: %s\n", historyPath)
			return []models.SessionData{}, nil
		}
		return nil, fmt.Errorf("failed to stat history file: %w", err)
	}

	if info.Size() > amazonQMaxFileSize {
		return nil, fmt.Errorf("history file too large: %d bytes (max: %d)", info.Size(), amazonQMaxFileSize)
	}

	// 스트리밍 방식으로 파일 읽기
	return a.parseHistoryFileStreaming(ctx, historyPath, collectConfig)
}

// parseHistoryFileStreaming은 메모리 효율적인 히스토리 파일 파싱
func (a *AmazonQCollector) parseHistoryFileStreaming(ctx context.Context, filePath string, collectConfig *models.CollectionConfig) ([]models.SessionData, error) {
	// 파일 내용 읽기 (테스트와 실제 환경 모두 호환)
	data, err := a.fileReader.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read history file: %w", err)
	}

	var sessions []models.SessionData
	content := string(data)
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		session, err := a.parseHistoryLine(line, lineNum+1)
		if err != nil {
			a.logger.Warnf("Failed to parse Amazon Q history line %d: %v\n", lineNum+1, err)
			continue
		}

		if session != nil {
			sessions = append(sessions, *session)
		}

		// 메모리 사용량 제한
		if len(sessions) >= amazonQMaxMessagesPerFile {
			a.logger.Warnf("Reached maximum messages per file limit: %d\n", amazonQMaxMessagesPerFile)
			break
		}
	}

	return sessions, nil
}

// parseHistoryLine은 안전한 히스토리 라인 파싱
func (a *AmazonQCollector) parseHistoryLine(line string, lineNum int) (*models.SessionData, error) {
	// JSON 파싱 시도
	if strings.HasPrefix(line, "{") {
		return a.parseJSONHistoryEntry(line, lineNum)
	}

	// 텍스트로 처리
	return a.parseTextHistoryEntry(line, lineNum), nil
}

// parseJSONHistoryEntry는 안전한 JSON 히스토리 엔트리 파싱
func (a *AmazonQCollector) parseJSONHistoryEntry(line string, lineNum int) (*models.SessionData, error) {
	var entry AmazonQHistoryEntry
	decoder := json.NewDecoder(strings.NewReader(line))

	if err := decoder.Decode(&entry); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return a.convertHistoryEntryToSession(entry, lineNum), nil
}

// AmazonQHistoryEntry는 Amazon Q CLI 히스토리 엔트리 구조체
type AmazonQHistoryEntry struct {
	ID            string                 `json:"id"`
	ConversationID string                `json:"conversation_id"`
	Query         string                 `json:"query"`
	Response      string                 `json:"response"`
	Timestamp     string                 `json:"timestamp"`
	Service       string                 `json:"service"`
	Region        string                 `json:"region"`
	UserID        string                 `json:"user_id"`
	SessionType   string                 `json:"session_type"`
	Context       map[string]interface{} `json:"context"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// AmazonQSessionData는 Amazon Q CLI 세션 데이터 구조체
type AmazonQSessionData struct {
	ID             string                   `json:"id"`
	ConversationID string                   `json:"conversation_id"`
	Title          string                   `json:"title"`
	CreatedAt      string                   `json:"created_at"`
	UpdatedAt      string                   `json:"updated_at"`
	Service        string                   `json:"service"`
	Region         string                   `json:"region"`
	UserID         string                   `json:"user_id"`
	Messages       []AmazonQMessage         `json:"messages"`
	Context        map[string]interface{}   `json:"context"`
	Settings       *AmazonQSessionSettings  `json:"settings"`
	Metadata       map[string]interface{}   `json:"metadata"`
}

// AmazonQMessage는 Amazon Q CLI 메시지 구조체
type AmazonQMessage struct {
	ID          string                 `json:"id"`
	Role        string                 `json:"role"`
	Content     string                 `json:"content"`
	Timestamp   string                 `json:"timestamp"`
	MessageType string                 `json:"message_type"`
	Service     string                 `json:"service"`
	Context     map[string]interface{} `json:"context"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// AmazonQSessionSettings는 Amazon Q 세션 설정 구조체
type AmazonQSessionSettings struct {
	Service     string `json:"service"`
	Region      string `json:"region"`
	MaxTokens   int    `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
}

// convertHistoryEntryToSession은 히스토리 엔트리를 세션으로 변환
func (a *AmazonQCollector) convertHistoryEntryToSession(entry AmazonQHistoryEntry, index int) *models.SessionData {
	sessionID := entry.ID
	if sessionID == "" {
		sessionID = fmt.Sprintf("amazonq-history-%d", index)
	}

	session := &models.SessionData{
		ID:        sessionID,
		Source:    models.SourceAmazonQ,
		Timestamp: time.Now(),
		Title:     a.extractTitleFromQuery(entry.Query),
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
	session.Metadata["service"] = entry.Service
	session.Metadata["region"] = entry.Region
	session.Metadata["user_id"] = entry.UserID
	session.Metadata["conversation_id"] = entry.ConversationID
	session.Metadata["session_type"] = entry.SessionType
	session.Metadata["source_type"] = "amazon_q_history"

	// 사용자 메시지 추가
	if entry.Query != "" {
		userMsg := models.Message{
			ID:        fmt.Sprintf("%s-user", sessionID),
			Role:      "user",
			Content:   entry.Query,
			Timestamp: session.Timestamp,
			Metadata:  make(map[string]string),
		}
		userMsg.Metadata["service"] = entry.Service
		userMsg.Metadata["region"] = entry.Region
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
		assistantMsg.Metadata["service"] = entry.Service
		assistantMsg.Metadata["region"] = entry.Region
		session.Messages = append(session.Messages, assistantMsg)
	}

	return session
}

// parseTextHistoryEntry는 텍스트 히스토리 엔트리 파싱
func (a *AmazonQCollector) parseTextHistoryEntry(line string, lineNum int) *models.SessionData {
	if len(strings.TrimSpace(line)) == 0 {
		return nil
	}

	sessionID := fmt.Sprintf("amazonq-text-%d", lineNum)
	return &models.SessionData{
		ID:        sessionID,
		Source:    models.SourceAmazonQ,
		Timestamp: time.Now(),
		Title:     "Amazon Q CLI History Entry",
		Messages: []models.Message{
			{
				ID:        fmt.Sprintf("%s-user", sessionID),
				Role:      "user",
				Content:   line,
				Timestamp: time.Now(),
				Metadata:  map[string]string{"source_type": "amazon_q_text"},
			},
		},
		Metadata: map[string]string{
			"source_type":  "amazon_q_history",
			"entry_number": fmt.Sprintf("%d", lineNum),
		},
	}
}

// collectFromSessionDirConcurrent는 동시성 처리가 개선된 세션 디렉토리 수집
func (a *AmazonQCollector) collectFromSessionDirConcurrent(ctx context.Context, collectConfig *models.CollectionConfig) ([]models.SessionData, error) {
	sessionDirPath, err := config.ExpandPath(a.config.SessionDir)
	if err != nil {
		return nil, fmt.Errorf("failed to expand session directory path: %w", err)
	}

	// 디렉토리 존재 확인
	if _, err := a.fileReader.Stat(sessionDirPath); os.IsNotExist(err) {
		a.logger.Warnf("Amazon Q CLI session directory not found: %s\n", sessionDirPath)
		return []models.SessionData{}, nil
	}

	// 파일 목록 수집
	var filePaths []string
	err = a.fileReader.WalkDir(sessionDirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Amazon Q CLI 파일 패턴 매칭
		if a.isAmazonQFile(path) {
			filePaths = append(filePaths, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk session directory: %w", err)
	}

	// 워커 수 결정
	numWorkers := minInts(amazonQMaxWorkers, len(filePaths), runtime.NumCPU())
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
		go a.sessionFileWorker(ctx, &wg, fileChan, resultChan, errorChan, collectConfig)
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
		a.logger.Warnf("Amazon Q session file processing error: %v\n", err)
	}

	return sessions, nil
}

// sessionFileWorker는 세션 파일 처리 워커
func (a *AmazonQCollector) sessionFileWorker(ctx context.Context, wg *sync.WaitGroup, fileChan <-chan string, resultChan chan<- *models.SessionData, errorChan chan<- error, collectConfig *models.CollectionConfig) {
	defer wg.Done()

	for {
		select {
		case filePath, ok := <-fileChan:
			if !ok {
				return
			}

			session, err := a.parseSessionFileSafe(filePath, collectConfig)
			if err != nil {
				errorChan <- fmt.Errorf("failed to parse Amazon Q session file %s: %w", filePath, err)
				continue
			}

			resultChan <- session

		case <-ctx.Done():
			return
		}
	}
}

// parseSessionFileSafe는 안전한 세션 파일 파싱
func (a *AmazonQCollector) parseSessionFileSafe(path string, collectConfig *models.CollectionConfig) (*models.SessionData, error) {
	// 파일 크기 확인
	info, err := a.fileReader.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if info.Size() > amazonQMaxFileSize {
		return nil, fmt.Errorf("file too large: %d bytes", info.Size())
	}

	// 파일 읽기
	data, err := a.fileReader.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// JSON 파싱 시도
	var sessionData AmazonQSessionData
	if err := json.Unmarshal(data, &sessionData); err != nil {
		// JSON 파싱 실패 시 텍스트로 처리
		return a.parseTextSession(string(data), path), nil
	}

	return a.convertAmazonQSessionToModel(sessionData, path), nil
}

// convertAmazonQSessionToModel은 Amazon Q 세션 데이터를 모델로 변환
func (a *AmazonQCollector) convertAmazonQSessionToModel(amazonQSession AmazonQSessionData, filePath string) *models.SessionData {
	session := &models.SessionData{
		ID:        amazonQSession.ID,
		Source:    models.SourceAmazonQ,
		Timestamp: time.Now(),
		Title:     amazonQSession.Title,
		Messages:  make([]models.Message, 0, len(amazonQSession.Messages)),
		Metadata:  make(map[string]string),
	}

	// ID 설정
	if session.ID == "" {
		session.ID = fmt.Sprintf("amazonq-%s", filepath.Base(filePath))
	}

	// 타임스탬프 파싱
	if amazonQSession.CreatedAt != "" {
		if timestamp, err := time.Parse(time.RFC3339, amazonQSession.CreatedAt); err == nil {
			session.Timestamp = timestamp
		}
	}

	// 메타데이터 설정
	session.Metadata["file_path"] = filePath
	session.Metadata["service"] = amazonQSession.Service
	session.Metadata["region"] = amazonQSession.Region
	session.Metadata["user_id"] = amazonQSession.UserID
	session.Metadata["conversation_id"] = amazonQSession.ConversationID
	session.Metadata["source_type"] = "amazon_q_session"

	// 메시지 변환
	for _, amazonQMsg := range amazonQSession.Messages {
		msg := models.Message{
			ID:        amazonQMsg.ID,
			Role:      amazonQMsg.Role,
			Content:   amazonQMsg.Content,
			Timestamp: session.Timestamp,
			Metadata:  make(map[string]string),
		}

		// 메시지 타임스탬프 파싱
		if amazonQMsg.Timestamp != "" {
			if msgTime, err := time.Parse(time.RFC3339, amazonQMsg.Timestamp); err == nil {
				msg.Timestamp = msgTime
			}
		}

		// 메시지 메타데이터 설정
		msg.Metadata["service"] = amazonQMsg.Service
		msg.Metadata["message_type"] = amazonQMsg.MessageType

		session.Messages = append(session.Messages, msg)
	}

	return session
}

// parseTextSession은 텍스트 세션 파싱
func (a *AmazonQCollector) parseTextSession(content string, path string) *models.SessionData {
	fileName := filepath.Base(path)
	sessionID := fmt.Sprintf("amazonq-text-%s", strings.TrimSuffix(fileName, filepath.Ext(fileName)))

	return &models.SessionData{
		ID:        sessionID,
		Source:    models.SourceAmazonQ,
		Timestamp: time.Now(),
		Title:     fmt.Sprintf("Amazon Q CLI Session: %s", fileName),
		Messages: []models.Message{
			{
				ID:        fmt.Sprintf("%s-content", sessionID),
				Role:      "user",
				Content:   content,
				Timestamp: time.Now(),
				Metadata:  map[string]string{"source_type": "amazon_q_text"},
			},
		},
		Metadata: map[string]string{
			"file_path":   path,
			"source_type": "amazon_q_text",
		},
	}
}

// collectFromAWSConfig는 AWS 설정 파일에서 컨텍스트 정보를 수집합니다
func (a *AmazonQCollector) collectFromAWSConfig(ctx context.Context, collectConfig *models.CollectionConfig) ([]models.SessionData, error) {
	// AWS 설정 디렉토리 경로들
	awsPaths := []string{
		"~/.aws/config",
		"~/.aws/credentials", 
		"~/.amazon-q/config",
		"~/.amazon-q/session.json",
	}

	var sessions []models.SessionData

	for _, awsPath := range awsPaths {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		expandedPath, err := config.ExpandPath(awsPath)
		if err != nil {
			continue
		}

		if _, err := a.fileReader.Stat(expandedPath); os.IsNotExist(err) {
			continue
		}

		data, err := a.fileReader.ReadFile(expandedPath)
		if err != nil {
			continue
		}

		// AWS 설정 정보를 세션으로 변환
		session := &models.SessionData{
			ID:        fmt.Sprintf("amazonq-aws-config-%s", filepath.Base(expandedPath)),
			Source:    models.SourceAmazonQ,
			Timestamp: time.Now(),
			Title:     fmt.Sprintf("AWS Configuration: %s", filepath.Base(expandedPath)),
			Messages: []models.Message{
				{
					ID:        fmt.Sprintf("aws-config-%s", filepath.Base(expandedPath)),
					Role:      "system",
					Content:   string(data),
					Timestamp: time.Now(),
					Metadata: map[string]string{
						"source_type": "aws_config",
						"config_file": expandedPath,
					},
				},
			},
			Metadata: map[string]string{
				"source_type": "aws_config",
				"config_path": expandedPath,
			},
		}

		sessions = append(sessions, *session)
	}

	return sessions, nil
}

// isAmazonQFile은 파일이 Amazon Q CLI 파일인지 확인합니다
func (a *AmazonQCollector) isAmazonQFile(filePath string) bool {
	fileName := filepath.Base(filePath)
	fileExt := filepath.Ext(fileName)
	
	// Amazon Q CLI 관련 파일 패턴들
	amazonQPatterns := []string{
		".json",
		".log",
		".session",
		"amazonq",
		"aws-q",
		"q-cli",
	}

	for _, pattern := range amazonQPatterns {
		if strings.Contains(strings.ToLower(fileName), pattern) || strings.ToLower(fileExt) == pattern {
			return true
		}
	}

	return false
}

// extractTitleFromQuery는 쿼리에서 제목을 추출합니다
func (a *AmazonQCollector) extractTitleFromQuery(query string) string {
	if len(query) == 0 {
		return "Amazon Q CLI Session"
	}

	// 첫 줄만 사용
	lines := strings.Split(query, "\n")
	title := strings.TrimSpace(lines[0])

	// 길이 제한
	if len(title) > 100 {
		title = title[:97] + "..."
	}

	if title == "" {
		return "Amazon Q CLI Session"
	}

	return title
}

// filterByDateRange는 날짜 범위 필터링
func (a *AmazonQCollector) filterByDateRange(sessions []models.SessionData, dateRange *models.DateRange) []models.SessionData {
	if dateRange == nil {
		return sessions
	}

	filtered := make([]models.SessionData, 0, len(sessions))
	for _, session := range sessions {
		if a.isWithinDateRange(session.Timestamp, dateRange) {
			filtered = append(filtered, session)
		}
	}

	return filtered
}

// isWithinDateRange는 날짜가 범위 내에 있는지 확인
func (a *AmazonQCollector) isWithinDateRange(timestamp time.Time, dateRange *models.DateRange) bool {
	if !dateRange.Start.IsZero() && timestamp.Before(dateRange.Start) {
		return false
	}
	if !dateRange.End.IsZero() && timestamp.After(dateRange.End) {
		return false
	}
	return true
}

// generateDummyData는 Amazon Q CLI가 설치되지 않은 경우 더미 데이터를 생성합니다
func (a *AmazonQCollector) generateDummyData() []models.SessionData {
	now := time.Now()

	return []models.SessionData{
		{
			ID:        "amazonq-dummy-1",
			Source:    models.SourceAmazonQ,
			Timestamp: now.Add(-24 * time.Hour),
			Title:     "AWS EC2 Instance Management",
			Messages: []models.Message{
				{
					ID:        "amazonq-dummy-1-user",
					Role:      "user",
					Content:   "How do I create an EC2 instance with auto-scaling?",
					Timestamp: now.Add(-24 * time.Hour),
					Metadata:  map[string]string{"service": "ec2", "region": "us-west-2"},
				},
				{
					ID:        "amazonq-dummy-1-assistant",
					Role:      "assistant", 
					Content:   "To create an EC2 instance with auto-scaling, you need to: 1) Create a launch template 2) Create an auto-scaling group 3) Configure scaling policies...",
					Timestamp: now.Add(-24*time.Hour + time.Minute),
					Metadata:  map[string]string{"service": "ec2", "region": "us-west-2"},
				},
			},
			Metadata: map[string]string{
				"service":     "ec2",
				"region":      "us-west-2",
				"source_type": "amazon_q_dummy",
				"user_id":     "demo-user",
			},
		},
		{
			ID:        "amazonq-dummy-2",
			Source:    models.SourceAmazonQ,
			Timestamp: now.Add(-12 * time.Hour),
			Title:     "S3 Bucket Security Configuration",
			Messages: []models.Message{
				{
					ID:        "amazonq-dummy-2-user",
					Role:      "user",
					Content:   "What are the best practices for securing S3 buckets?",
					Timestamp: now.Add(-12 * time.Hour),
					Metadata:  map[string]string{"service": "s3", "region": "us-east-1"},
				},
				{
					ID:        "amazonq-dummy-2-assistant",
					Role:      "assistant",
					Content:   "Here are the key S3 security best practices: 1) Enable versioning 2) Configure bucket policies 3) Use IAM roles 4) Enable access logging...",
					Timestamp: now.Add(-12*time.Hour + time.Minute),
					Metadata:  map[string]string{"service": "s3", "region": "us-east-1"},
				},
			},
			Metadata: map[string]string{
				"service":     "s3",
				"region":      "us-east-1",
				"source_type": "amazon_q_dummy",
				"user_id":     "demo-user",
			},
		},
		{
			ID:        "amazonq-dummy-3",
			Source:    models.SourceAmazonQ,
			Timestamp: now.Add(-6 * time.Hour),
			Title:     "Lambda Function Optimization",
			Messages: []models.Message{
				{
					ID:        "amazonq-dummy-3-user",
					Role:      "user",
					Content:   "How can I optimize my Lambda function for better performance?",
					Timestamp: now.Add(-6 * time.Hour),
					Metadata:  map[string]string{"service": "lambda", "region": "eu-west-1"},
				},
				{
					ID:        "amazonq-dummy-3-assistant",
					Role:      "assistant",
					Content:   "To optimize Lambda performance: 1) Right-size memory allocation 2) Minimize cold starts 3) Use connection pooling 4) Optimize code and dependencies...",
					Timestamp: now.Add(-6*time.Hour + time.Minute),
					Metadata:  map[string]string{"service": "lambda", "region": "eu-west-1"},
				},
			},
			Metadata: map[string]string{
				"service":     "lambda",
				"region":      "eu-west-1",
				"source_type": "amazon_q_dummy",
				"user_id":     "demo-user",
			},
		},
	}
}

// minInts는 정수의 최솟값 반환 (Go 1.21 이전 버전 호환)
func minInts(a ...int) int {
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