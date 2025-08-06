package processor

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"summerise-genai/pkg/models"
)

// Processor는 데이터 처리를 담당합니다
type Processor struct {
	config *models.ExportConfig
}

// NewProcessor는 새로운 데이터 처리기를 생성합니다
func NewProcessor(config *models.ExportConfig) *Processor {
	return &Processor{
		config: config,
	}
}

// Process는 세션 데이터를 처리하여 구조화된 형태로 변환합니다 (인터페이스 호환)
func (p *Processor) Process(ctx context.Context, sessions []models.SessionData) (interface{}, error) {
	// context 취소 확인
	select {
	case <-ctx.Done():
		return ProcessedData{}, ctx.Err()
	default:
	}

	if len(sessions) == 0 {
		return ProcessedData{}, nil
	}

	// 세션을 타임스탬프 기준으로 정렬
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Timestamp.After(sessions[j].Timestamp)
	})

	// context 취소 확인
	select {
	case <-ctx.Done():
		return ProcessedData{}, ctx.Err()
	default:
	}

	// 소스별로 그룹화
	sourceGroups := make(map[models.CollectionSource][]models.SessionData)
	for _, session := range sessions {
		sourceGroups[session.Source] = append(sourceGroups[session.Source], session)
	}

	// 통계 생성
	stats := p.generateStatistics(sessions, sourceGroups)

	// TOC 생성
	toc := p.generateTableOfContents(sourceGroups)

	return ProcessedData{
		Sessions:        sessions,
		SourceGroups:    sourceGroups,
		Statistics:      stats,
		TableOfContents: toc,
		ProcessedAt:     time.Now(),
	}, nil
}

// Validate는 처리기 설정이 유효한지 검증합니다
func (p *Processor) Validate() error {
	if p.config == nil {
		return fmt.Errorf("처리기 설정이 nil입니다")
	}
	return nil
}

// GetSupportedOutputFormats는 지원하는 출력 형식들을 반환합니다
func (p *Processor) GetSupportedOutputFormats() []string {
	return []string{"structured", "grouped", "statistical"}
}

// ProcessedData는 처리된 데이터를 나타냅니다
type ProcessedData struct {
	Sessions        []models.SessionData                                   `json:"sessions"`
	SourceGroups    map[models.CollectionSource][]models.SessionData       `json:"source_groups"`
	Statistics      Statistics                                             `json:"statistics"`
	TableOfContents []TOCEntry                                             `json:"table_of_contents"`
	ProcessedAt     time.Time                                              `json:"processed_at"`
}

// Statistics는 통계 정보를 나타냅니다
type Statistics struct {
	TotalSessions      int                                    `json:"total_sessions"`
	TotalMessages      int                                    `json:"total_messages"`
	TotalCommands      int                                    `json:"total_commands"`
	TotalFiles         int                                    `json:"total_files"`
	SourceCounts       map[models.CollectionSource]int       `json:"source_counts"`
	DateRange          *models.DateRange                      `json:"date_range,omitempty"`
	MostActiveSource   models.CollectionSource                `json:"most_active_source"`
	AverageSessionTime time.Duration                          `json:"average_session_time"`
}

// TOCEntry는 목차 항목을 나타냅니다
type TOCEntry struct {
	Title    string      `json:"title"`
	Level    int         `json:"level"`
	Anchor   string      `json:"anchor"`
	Children []TOCEntry  `json:"children,omitempty"`
}

func (p *Processor) generateStatistics(sessions []models.SessionData, sourceGroups map[models.CollectionSource][]models.SessionData) Statistics {
	stats := Statistics{
		TotalSessions: len(sessions),
		SourceCounts:  make(map[models.CollectionSource]int),
	}

	var totalMessages, totalCommands, totalFiles int
	var oldestTime, newestTime time.Time
	var sessionDurations []time.Duration

	// 초기값 설정
	if len(sessions) > 0 {
		oldestTime = sessions[0].Timestamp
		newestTime = sessions[0].Timestamp
	}

	// 통계 계산
	for source, sourceSessions := range sourceGroups {
		stats.SourceCounts[source] = len(sourceSessions)
		
		for _, session := range sourceSessions {
			// 메시지, 명령어, 파일 수 계산
			totalMessages += len(session.Messages)
			totalCommands += len(session.Commands)
			totalFiles += len(session.Files)
			
			// 날짜 범위 계산
			if session.Timestamp.Before(oldestTime) {
				oldestTime = session.Timestamp
			}
			if session.Timestamp.After(newestTime) {
				newestTime = session.Timestamp
			}
			
			// 세션 지속 시간 계산 (메시지 간 시간차 기반)
			if len(session.Messages) > 1 {
				first := session.Messages[0].Timestamp
				last := session.Messages[len(session.Messages)-1].Timestamp
				sessionDurations = append(sessionDurations, last.Sub(first))
			}
		}
	}

	stats.TotalMessages = totalMessages
	stats.TotalCommands = totalCommands
	stats.TotalFiles = totalFiles

	// 날짜 범위 설정
	if len(sessions) > 0 {
		stats.DateRange = &models.DateRange{
			Start: oldestTime,
			End:   newestTime,
		}
	}

	// 가장 활발한 소스 찾기
	maxCount := 0
	for source, count := range stats.SourceCounts {
		if count > maxCount {
			maxCount = count
			stats.MostActiveSource = source
		}
	}

	// 평균 세션 시간 계산
	if len(sessionDurations) > 0 {
		var total time.Duration
		for _, duration := range sessionDurations {
			total += duration
		}
		stats.AverageSessionTime = total / time.Duration(len(sessionDurations))
	}

	return stats
}

func (p *Processor) generateTableOfContents(sourceGroups map[models.CollectionSource][]models.SessionData) []TOCEntry {
	var toc []TOCEntry

	// 개요 섹션
	toc = append(toc, TOCEntry{
		Title:  "개요",
		Level:  1,
		Anchor: "overview",
	})

	// 통계 섹션
	toc = append(toc, TOCEntry{
		Title:  "통계",
		Level:  1,
		Anchor: "statistics",
	})

	// 소스별 섹션
	sources := make([]models.CollectionSource, 0, len(sourceGroups))
	for source := range sourceGroups {
		sources = append(sources, source)
	}
	
	// 소스 정렬
	sort.Slice(sources, func(i, j int) bool {
		return string(sources[i]) < string(sources[j])
	})

	for _, source := range sources {
		sessions := sourceGroups[source]
		if len(sessions) == 0 {
			continue
		}

		sourceTitle := p.getSourceDisplayName(source)
		sourceAnchor := p.generateAnchor(sourceTitle)
		
		sourceEntry := TOCEntry{
			Title:    fmt.Sprintf("%s (%d개 세션)", sourceTitle, len(sessions)),
			Level:    1,
			Anchor:   sourceAnchor,
			Children: make([]TOCEntry, 0),
		}

		// 각 세션을 하위 항목으로 추가
		for _, session := range sessions {
			sessionTitle := session.Title
			if sessionTitle == "" {
				sessionTitle = fmt.Sprintf("세션 %s", session.ID)
			}
			
			sessionEntry := TOCEntry{
				Title:  sessionTitle,
				Level:  2,
				Anchor: p.generateAnchor(fmt.Sprintf("%s-%s", sourceAnchor, session.ID)),
			}
			sourceEntry.Children = append(sourceEntry.Children, sessionEntry)
		}

		toc = append(toc, sourceEntry)
	}

	return toc
}

func (p *Processor) getSourceDisplayName(source models.CollectionSource) string {
	switch source {
	case models.SourceClaudeCode:
		return "Claude Code"
	case models.SourceGeminiCLI:
		return "Gemini CLI"
	case models.SourceAmazonQ:
		return "Amazon Q"
	default:
		return string(source)
	}
}

func (p *Processor) generateAnchor(text string) string {
	// 소문자 변환 및 공백을 하이픈으로 변경
	anchor := strings.ToLower(text)
	anchor = strings.ReplaceAll(anchor, " ", "-")
	anchor = strings.ReplaceAll(anchor, "_", "-")
	
	// 특수 문자 제거
	var result strings.Builder
	for _, r := range anchor {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	
	// 연속된 하이픈 제거
	anchor = result.String()
	for strings.Contains(anchor, "--") {
		anchor = strings.ReplaceAll(anchor, "--", "-")
	}
	
	// 시작과 끝의 하이픈 제거
	anchor = strings.Trim(anchor, "-")
	
	return anchor
}

// FormatCodeContent는 코드 내용을 마크다운 형식으로 포맷팅합니다
func (p *Processor) FormatCodeContent(content string) string {
	if !p.config.FormatCodeBlocks {
		return content
	}

	// 간단한 코드 블록 감지 및 포맷팅
	lines := strings.Split(content, "\n")
	var formatted strings.Builder
	inCodeBlock := false
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// 코드 블록 시작/종료 감지
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			formatted.WriteString(line)
			formatted.WriteString("\n")
			continue
		}
		
		// 코드 블록 내부이거나 들여쓰기된 코드로 보이는 경우
		if inCodeBlock || (strings.HasPrefix(line, "    ") && trimmed != "") {
			formatted.WriteString(line)
		} else {
			// 일반 텍스트는 그대로
			formatted.WriteString(line)
		}
		formatted.WriteString("\n")
	}
	
	return strings.TrimSuffix(formatted.String(), "\n")
}

// SanitizeContent는 마크다운에서 문제가 될 수 있는 문자를 이스케이프합니다
func (p *Processor) SanitizeContent(content string) string {
	// 마크다운 특수 문자 이스케이프
	content = strings.ReplaceAll(content, "\\", "\\\\")
	content = strings.ReplaceAll(content, "`", "\\`")
	content = strings.ReplaceAll(content, "*", "\\*")
	content = strings.ReplaceAll(content, "_", "\\_")
	content = strings.ReplaceAll(content, "[", "\\[")
	content = strings.ReplaceAll(content, "]", "\\]")
	content = strings.ReplaceAll(content, "(", "\\(")
	content = strings.ReplaceAll(content, ")", "\\)")
	content = strings.ReplaceAll(content, "#", "\\#")
	content = strings.ReplaceAll(content, "+", "\\+")
	content = strings.ReplaceAll(content, "-", "\\-")
	content = strings.ReplaceAll(content, ".", "\\.")
	content = strings.ReplaceAll(content, "!", "\\!")
	
	return content
}