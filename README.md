# SSamAI - Summerise GenAI Job Results

AI CLI 도구들의 작업 내용을 수집하여 구조화된 마크다운 문서로 변환하는 자동화 도구입니다.

## 개요

SSamAI는 Claude Code, Gemini CLI, Amazon Q CLI에서 작업한 세션 데이터, 히스토리, 로그 등을 자동으로 수집하여 하나의 구조화된 마크다운 문서로 정리해주는 도구입니다.

### 주요 기능

- 🤖 **다중 AI CLI 도구 지원**: Claude Code, Gemini CLI, Amazon Q CLI 데이터 수집
- 📄 **마크다운 자동 생성**: 구조화된 마크다운 문서 자동 생성

## 설치

### 요구 사항

- Go 1.21+

### 빌드

```bash
git clone <repository-url>
cd summerise-genai
go build -o summerise-genai
```

## 사용법

### 1. 설정 초기화

```bash
# 기본 설정 파일 생성
./summerise-genai config --init

# 현재 설정 확인
./summerise-genai config --show

# 설정 파일 검증
./summerise-genai config --validate
```

### 2. 데이터 수집

```bash
# 모든 AI 도구에서 데이터 수집
./summerise-genai collect --all --verbose

# 특정 도구만 수집
./summerise-genai collect --sources claude_code,gemini_cli

# 날짜 범위 지정
./summerise-genai collect --all --from 2024-01-01 --to 2024-01-31

# 파일 및 명령어 정보 포함
./summerise-genai collect --all --include-files --include-commands
```

### 3. 마크다운 내보내기

```bash
# 기본 마크다운 생성
./summerise-genai export --output ./summary.md

# 커스텀 옵션으로 생성
./summerise-genai export \
  --output ./detailed-summary.md \
  --template comprehensive \
  --custom project=MyProject \
  --custom version=1.0

# 간단한 형식으로 생성
./summerise-genai export \
  --output ./simple.md \
  --no-toc --no-meta --no-timestamp
```

## 프로젝트 구조

```
summerise-genai/
├── cmd/                    # CLI 명령어
│   ├── root.go            # 메인 CLI 진입점
│   ├── collect.go         # 데이터 수집 명령어
│   ├── export.go          # 마크다운 내보내기 명령어
│   └── config.go          # 설정 관리 명령어
├── internal/              # 내부 패키지
│   ├── collector/         # 데이터 수집기
│   │   └── claude.go      # Claude Code 수집기
│   ├── config/            # 설정 관리
│   ├── processor/         # 데이터 처리
│   └── exporter/          # 마크다운 내보내기
├── pkg/                   # 공개 패키지
│   ├── models/            # 데이터 모델
│   └── agents/            # MCP 에이전트
├── configs/               # 설정 파일
│   ├── agents.yaml        # 에이전트 설정
│   ├── claude-agents.json # Claude 에이전트 설정
│   └── collaboration.json # 협업 설정
└── templates/             # 마크다운 템플릿
```

## 설정

### 기본 설정 파일 (`configs/agents.yaml`)

```yaml
output_settings:
  default_template: "comprehensive"
  include_metadata: true
  include_timestamps: true
  format_code_blocks: true
  generate_toc: true
```

## 예시 출력

생성된 마크다운 문서는 다음과 같은 구조를 갖습니다:

- **목차**: 자동 생성된 TOC
- **개요**: 수집된 데이터 요약
- **통계**: 활동 통계 및 분석
- **소스별 세션**: 각 AI 도구별 세션 내용
  - 대화 내용
  - 실행된 명령어
  - 참조된 파일
  - 메타데이터

## 개발

### 테스트

```bash
go test ./...
```

### Gemini CLI와 협업

이 프로젝트는 개발 과정에서 Gemini CLI와의 협업을 통해 코드 품질을 향상시킵니다:

```bash
# 코드 리뷰 요청
gemini -p "다음 Go 코드를 검토해주세요: [코드 내용]"

# 아키텍처 검토
gemini -p "시스템 아키텍처를 검토하고 개선사항을 제안해주세요"
```

### 기여 방법

1. 이슈 생성
2. 기능 브랜치 생성
3. 구현 및 테스트
4. Gemini CLI로 코드 리뷰
5. Pull Request 생성

## 라이센스

MIT License - LICENSE 파일을 참조하세요.

## 향후 개선 계획

- [ ] Gemini CLI 실제 데이터 수집기 구현
- [ ] Amazon Q CLI 실제 데이터 수집기 구현
- [ ] 템플릿 시스템 확장
- [ ] 웹 기반 대시보드 추가
- [ ] 다국어 지원
- [ ] 플러그인 아키텍처 도입
