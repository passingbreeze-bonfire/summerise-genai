package exporter

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ssamai/internal/interfaces"
	"ssamai/internal/processor"
	"ssamai/pkg/models"
)

// MarkdownExporter는 마크다운 내보내기를 담당합니다
type MarkdownExporter struct {
	config *models.ExportConfig
}

// MarkdownExporter가 모든 관련 인터페이스들을 구현하는지 컴파일 타임에 확인 (ISP 적용)
var _ interfaces.DataExporter = (*MarkdownExporter)(nil)
var _ interfaces.ExporterInfo = (*MarkdownExporter)(nil)
var _ interfaces.ExporterValidator = (*MarkdownExporter)(nil)
var _ interfaces.FullDataExporter = (*MarkdownExporter)(nil)

// NewMarkdownExporter는 새로운 마크다운 내보내기 도구를 생성합니다
func NewMarkdownExporter(config *models.ExportConfig) *MarkdownExporter {
	return &MarkdownExporter{
		config: config,
	}
}

// Export는 처리된 데이터를 마크다운 파일로 내보냅니다 (인터페이스 호환)
func (e *MarkdownExporter) Export(ctx context.Context, data interface{}) error {
	// context 취소 확인
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 타입 캐스팅
	processedData, ok := data.(processor.ProcessedData)
	if !ok {
		return fmt.Errorf("잘못된 데이터 타입입니다. processor.ProcessedData가 필요합니다")
	}

	// 출력 디렉토리 생성
	outputDir := filepath.Dir(e.config.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("출력 디렉토리 생성 실패: %w", err)
	}

	// context 취소 확인
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 템플릿 선택 및 내용 생성
	content, err := e.generateMarkdownContent(&processedData)
	if err != nil {
		return fmt.Errorf("마크다운 내용 생성 실패: %w", err)
	}

	// 파일 쓰기
	if err := os.WriteFile(e.config.OutputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("파일 쓰기 실패: %w", err)
	}

	return nil
}

// ExportToWriter는 처리된 데이터를 Writer에 출력합니다
func (e *MarkdownExporter) ExportToWriter(ctx context.Context, data interface{}, writer io.Writer) error {
	// context 취소 확인
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 타입 캐스팅
	processedData, ok := data.(processor.ProcessedData)
	if !ok {
		return fmt.Errorf("잘못된 데이터 타입입니다. processor.ProcessedData가 필요합니다")
	}

	// 템플릿 선택 및 내용 생성
	content, err := e.generateMarkdownContent(&processedData)
	if err != nil {
		return fmt.Errorf("마크다운 내용 생성 실패: %w", err)
	}

	// Writer에 출력
	if _, err := writer.Write([]byte(content)); err != nil {
		return fmt.Errorf("Writer 출력 실패: %w", err)
	}

	return nil
}

// GetFormat은 내보내기 형식을 반환합니다
func (e *MarkdownExporter) GetFormat() string {
	return "markdown"
}

// Validate는 내보내기 설정이 유효한지 검증합니다
func (e *MarkdownExporter) Validate() error {
	if e.config == nil {
		return fmt.Errorf("내보내기 설정이 nil입니다")
	}
	
	if e.config.OutputPath == "" {
		return fmt.Errorf("출력 경로가 지정되지 않았습니다")
	}

	// 출력 디렉토리가 존재하는지 확인 (없으면 생성 가능한지 확인)
	outputDir := filepath.Dir(e.config.OutputPath)
	if outputDir != "" && outputDir != "." {
		if info, err := os.Stat(outputDir); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("출력 디렉토리 확인 실패: %w", err)
			}
		} else if !info.IsDir() {
			return fmt.Errorf("출력 경로의 부모가 디렉토리가 아닙니다: %s", outputDir)
		}
	}

	return nil
}

// GetSupportedTemplates는 지원하는 템플릿들을 반환합니다
func (e *MarkdownExporter) GetSupportedTemplates() []string {
	return []string{"default", "detailed", "summary", "compact"}
}

func (e *MarkdownExporter) generateMarkdownContent(data *processor.ProcessedData) (string, error) {
	var content strings.Builder

	// 헤더 생성
	e.writeHeader(&content, data)

	// 목차 생성
	if e.config.GenerateTOC {
		e.writeTableOfContents(&content, data.TableOfContents)
	}

	// 개요 섹션
	e.writeOverview(&content, data)

	// 통계 섹션
	e.writeStatistics(&content, data.Statistics)

	// 소스별 세션 내용
	e.writeSourceSections(&content, data)

	// 푸터 생성
	if e.config.IncludeMetadata {
		e.writeFooter(&content, data)
	}

	return content.String(), nil
}

func (e *MarkdownExporter) writeHeader(content *strings.Builder, data *processor.ProcessedData) {
	content.WriteString("# AI CLI 도구 활동 요약\n\n")
	
	if e.config.IncludeTimestamps {
		content.WriteString(fmt.Sprintf("**생성 시간**: %s\n\n", 
			data.ProcessedAt.Format("2006-01-02 15:04:05")))
	}

	if len(data.Sessions) > 0 && data.Statistics.DateRange != nil {
		content.WriteString(fmt.Sprintf("**활동 기간**: %s ~ %s\n\n",
			data.Statistics.DateRange.Start.Format("2006-01-02"),
			data.Statistics.DateRange.End.Format("2006-01-02")))
	}
}

func (e *MarkdownExporter) writeTableOfContents(content *strings.Builder, toc []processor.TOCEntry) {
	content.WriteString("## 목차\n\n")
	
	for _, entry := range toc {
		e.writeTOCEntry(content, entry, 0)
	}
	content.WriteString("\n")
}

func (e *MarkdownExporter) writeTOCEntry(content *strings.Builder, entry processor.TOCEntry, indent int) {
	// 들여쓰기 생성
	for i := 0; i < indent; i++ {
		content.WriteString("  ")
	}
	
	content.WriteString(fmt.Sprintf("- [%s](#%s)\n", entry.Title, entry.Anchor))
	
	// 하위 항목들 처리
	for _, child := range entry.Children {
		e.writeTOCEntry(content, child, indent+1)
	}
}

func (e *MarkdownExporter) writeOverview(content *strings.Builder, data *processor.ProcessedData) {
	content.WriteString("## 개요 {#overview}\n\n")
	
	if len(data.Sessions) == 0 {
		content.WriteString("수집된 세션이 없습니다.\n\n")
		return
	}

	content.WriteString(fmt.Sprintf("총 **%d개**의 AI 도구 세션이 수집되었습니다.\n\n", 
		data.Statistics.TotalSessions))

	// 소스별 요약
	content.WriteString("### 소스별 활동 현황\n\n")
	content.WriteString("| AI 도구 | 세션 수 | 메시지 수 |\n")
	content.WriteString("|---------|---------|----------|\n")
	
	for source, sessions := range data.SourceGroups {
		if len(sessions) == 0 {
			continue
		}
		
		messageCount := 0
		for _, session := range sessions {
			messageCount += len(session.Messages)
		}
		
		sourceName := e.getSourceDisplayName(source)
		content.WriteString(fmt.Sprintf("| %s | %d | %d |\n", 
			sourceName, len(sessions), messageCount))
	}
	content.WriteString("\n")
}

func (e *MarkdownExporter) writeStatistics(content *strings.Builder, stats processor.Statistics) {
	content.WriteString("## 통계 {#statistics}\n\n")
	
	content.WriteString("### 전체 활동 통계\n\n")
	content.WriteString(fmt.Sprintf("- **총 세션 수**: %d개\n", stats.TotalSessions))
	content.WriteString(fmt.Sprintf("- **총 메시지 수**: %d개\n", stats.TotalMessages))
	
	if stats.TotalCommands > 0 {
		content.WriteString(fmt.Sprintf("- **총 실행 명령어 수**: %d개\n", stats.TotalCommands))
	}
	
	if stats.TotalFiles > 0 {
		content.WriteString(fmt.Sprintf("- **총 참조 파일 수**: %d개\n", stats.TotalFiles))
	}
	
	if stats.MostActiveSource != "" {
		sourceName := e.getSourceDisplayName(stats.MostActiveSource)
		content.WriteString(fmt.Sprintf("- **가장 활발한 도구**: %s\n", sourceName))
	}
	
	if stats.AverageSessionTime > 0 {
		content.WriteString(fmt.Sprintf("- **평균 세션 지속 시간**: %v\n", 
			stats.AverageSessionTime.Round(time.Second)))
	}
	
	content.WriteString("\n")
}

func (e *MarkdownExporter) writeSourceSections(content *strings.Builder, data *processor.ProcessedData) {
	// 소스별로 정렬된 순서로 처리
	sources := []models.CollectionSource{
		models.SourceClaudeCode,
		models.SourceGeminiCLI,
		models.SourceAmazonQ,
	}

	for _, source := range sources {
		sessions, exists := data.SourceGroups[source]
		if !exists || len(sessions) == 0 {
			continue
		}

		sourceName := e.getSourceDisplayName(source)
		anchor := e.generateAnchor(sourceName)
		
		content.WriteString(fmt.Sprintf("## %s {#%s}\n\n", sourceName, anchor))
		content.WriteString(fmt.Sprintf("총 %d개의 세션이 수집되었습니다.\n\n", len(sessions)))

		// 각 세션 내용
		for _, session := range sessions {
			e.writeSession(content, session, source)
		}
	}
}

func (e *MarkdownExporter) writeSession(content *strings.Builder, session models.SessionData, source models.CollectionSource) {
	// 세션 제목
	title := session.Title
	if title == "" {
		title = fmt.Sprintf("세션 %s", session.ID)
	}
	
	sourceName := e.getSourceDisplayName(source)
	anchor := e.generateAnchor(fmt.Sprintf("%s-%s", sourceName, session.ID))
	
	content.WriteString(fmt.Sprintf("### %s {#%s}\n\n", title, anchor))

	// 세션 메타데이터
	if e.config.IncludeMetadata {
		content.WriteString(fmt.Sprintf("**세션 ID**: `%s`\n", session.ID))
		
		if e.config.IncludeTimestamps {
			content.WriteString(fmt.Sprintf("**시간**: %s\n", 
				session.Timestamp.Format("2006-01-02 15:04:05")))
		}
		
		if len(session.Metadata) > 0 {
			content.WriteString("**메타데이터**:\n")
			for key, value := range session.Metadata {
				content.WriteString(fmt.Sprintf("- %s: %s\n", key, value))
			}
		}
		content.WriteString("\n")
	}

	// 메시지들
	if len(session.Messages) > 0 {
		content.WriteString("#### 대화 내용\n\n")
		for i, message := range session.Messages {
			e.writeMessage(content, message, i+1)
		}
	}

	// 명령어들
	if len(session.Commands) > 0 && e.config.IncludeMetadata {
		content.WriteString("#### 실행된 명령어\n\n")
		for i, cmd := range session.Commands {
			e.writeCommand(content, cmd, i+1)
		}
	}

	// 파일 참조
	if len(session.Files) > 0 && e.config.IncludeMetadata {
		content.WriteString("#### 참조된 파일\n\n")
		for _, file := range session.Files {
			content.WriteString(fmt.Sprintf("- **%s** (`%s`)\n", file.Name, file.Path))
			if file.Size > 0 {
				content.WriteString(fmt.Sprintf("  - 크기: %d bytes\n", file.Size))
			}
			if e.config.IncludeTimestamps {
				content.WriteString(fmt.Sprintf("  - 수정시간: %s\n", 
					file.ModTime.Format("2006-01-02 15:04:05")))
			}
		}
		content.WriteString("\n")
	}

	content.WriteString("---\n\n")
}

func (e *MarkdownExporter) writeMessage(content *strings.Builder, message models.Message, index int) {
	roleIcon := ""
	switch message.Role {
	case "user":
		roleIcon = "👤"
	case "assistant":
		roleIcon = "🤖"
	case "system":
		roleIcon = "⚙️"
	}

	content.WriteString(fmt.Sprintf("**%s %s** (%d)\n\n", roleIcon, 
		strings.Title(message.Role), index))

	if e.config.IncludeTimestamps {
		content.WriteString(fmt.Sprintf("*%s*\n\n", 
			message.Timestamp.Format("15:04:05")))
	}

	// 메시지 내용 처리
	messageContent := message.Content
	if e.config.FormatCodeBlocks {
		messageContent = e.formatCodeInContent(messageContent)
	}

	content.WriteString(messageContent)
	content.WriteString("\n\n")
}

func (e *MarkdownExporter) writeCommand(content *strings.Builder, cmd models.Command, index int) {
	content.WriteString(fmt.Sprintf("**명령어 %d**\n\n", index))
	
	// 명령어 라인
	cmdLine := cmd.Command
	if len(cmd.Args) > 0 {
		cmdLine += " " + strings.Join(cmd.Args, " ")
	}
	
	content.WriteString(fmt.Sprintf("```bash\n%s\n```\n\n", cmdLine))
	
	// 실행 정보
	if e.config.IncludeTimestamps {
		content.WriteString(fmt.Sprintf("- **실행시간**: %s\n", 
			cmd.Timestamp.Format("2006-01-02 15:04:05")))
	}
	content.WriteString(fmt.Sprintf("- **종료코드**: %d\n", cmd.ExitCode))
	if cmd.Duration > 0 {
		content.WriteString(fmt.Sprintf("- **소요시간**: %v\n", cmd.Duration))
	}

	// 출력 결과
	if cmd.Output != "" {
		content.WriteString("\n**출력**:\n")
		content.WriteString(fmt.Sprintf("```\n%s\n```\n", cmd.Output))
	}

	// 에러 메시지
	if cmd.Error != "" {
		content.WriteString("\n**에러**:\n")
		content.WriteString(fmt.Sprintf("```\n%s\n```\n", cmd.Error))
	}

	content.WriteString("\n")
}

func (e *MarkdownExporter) writeFooter(content *strings.Builder, data *processor.ProcessedData) {
	content.WriteString("---\n\n")
	content.WriteString("## 메타데이터\n\n")
	content.WriteString(fmt.Sprintf("- **문서 생성 도구**: summerise-genai\n"))
	content.WriteString(fmt.Sprintf("- **생성 시간**: %s\n", 
		data.ProcessedAt.Format("2006-01-02 15:04:05")))
	content.WriteString(fmt.Sprintf("- **템플릿**: %s\n", e.config.Template))
	
	if len(e.config.CustomFields) > 0 {
		content.WriteString("- **사용자 정의 필드**:\n")
		for key, value := range e.config.CustomFields {
			content.WriteString(fmt.Sprintf("  - %s: %s\n", key, value))
		}
	}
	
	content.WriteString("\n")
}

func (e *MarkdownExporter) formatCodeInContent(content string) string {
	// 간단한 코드 블록 감지 및 개선
	lines := strings.Split(content, "\n")
	var formatted strings.Builder
	inCodeBlock := false
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
		}
		
		formatted.WriteString(line)
		formatted.WriteString("\n")
	}
	
	return strings.TrimSuffix(formatted.String(), "\n")
}

func (e *MarkdownExporter) getSourceDisplayName(source models.CollectionSource) string {
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

func (e *MarkdownExporter) generateAnchor(text string) string {
	anchor := strings.ToLower(text)
	anchor = strings.ReplaceAll(anchor, " ", "-")
	anchor = strings.ReplaceAll(anchor, "_", "-")
	
	var result strings.Builder
	for _, r := range anchor {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	
	anchor = result.String()
	for strings.Contains(anchor, "--") {
		anchor = strings.ReplaceAll(anchor, "--", "-")
	}
	
	return strings.Trim(anchor, "-")
}