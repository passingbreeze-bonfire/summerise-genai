.PHONY: help test test-verbose test-coverage test-race build lint clean fmt vet run install dev-deps benchmark

# 기본 변수
BINARY_NAME=summerise-genai
MAIN_FILE=main.go
PKG_LIST=$$(go list ./... | grep -v /vendor/)

# 색상 출력을 위한 변수
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[0;33m
NC=\033[0m # No Color

## help: 사용 가능한 명령어들을 표시합니다
help:
	@echo "사용 가능한 명령어들:"
	@echo "  ${GREEN}test${NC}           - 모든 테스트 실행"
	@echo "  ${GREEN}test-verbose${NC}   - 상세한 테스트 출력"
	@echo "  ${GREEN}test-coverage${NC}  - 커버리지 리포트 생성"
	@echo "  ${GREEN}test-race${NC}      - Race condition 검사"
	@echo "  ${GREEN}build${NC}          - 바이너리 빌드"
	@echo "  ${GREEN}lint${NC}           - golangci-lint 실행"
	@echo "  ${GREEN}fmt${NC}            - 코드 포매팅 (goimports)"
	@echo "  ${GREEN}vet${NC}            - go vet 실행"
	@echo "  ${GREEN}run${NC}            - 애플리케이션 실행"
	@echo "  ${GREEN}install${NC}        - 바이너리를 GOPATH/bin에 설치"
	@echo "  ${GREEN}clean${NC}          - 생성된 파일들 정리"
	@echo "  ${GREEN}dev-deps${NC}       - 개발 의존성 설치"
	@echo "  ${GREEN}benchmark${NC}      - 벤치마크 테스트 실행"

## test: 모든 테스트 실행
test:
	@echo "${YELLOW}모든 테스트 실행 중...${NC}"
	@go test ./...

## test-verbose: 상세한 테스트 출력
test-verbose:
	@echo "${YELLOW}상세한 테스트 실행 중...${NC}"
	@go test -v ./...

## test-coverage: 테스트 커버리지 리포트 생성
test-coverage:
	@echo "${YELLOW}커버리지 리포트 생성 중...${NC}"
	@mkdir -p coverage
	@go test -coverprofile=coverage/coverage.out ./...
	@go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@go tool cover -func=coverage/coverage.out
	@echo "${GREEN}커버리지 리포트가 coverage/coverage.html에 생성되었습니다${NC}"

## test-race: Race condition 검사
test-race:
	@echo "${YELLOW}Race condition 검사 중...${NC}"
	@go test -race -short ./...

## build: 바이너리 빌드
build:
	@echo "${YELLOW}바이너리 빌드 중...${NC}"
	@go build -o $(BINARY_NAME) $(MAIN_FILE)
	@echo "${GREEN}바이너리가 $(BINARY_NAME)으로 생성되었습니다${NC}"

## lint: golangci-lint 실행 (설치되어 있는 경우)
lint:
	@echo "${YELLOW}Linting 실행 중...${NC}"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "${RED}golangci-lint가 설치되어 있지 않습니다. 'go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest'로 설치하세요${NC}"; \
		echo "${YELLOW}go vet으로 기본 검사를 실행합니다...${NC}"; \
		go vet ./...; \
	fi

## fmt: 코드 포매팅 (goimports 사용, 없으면 gofmt)
fmt:
	@echo "${YELLOW}코드 포매팅 중...${NC}"
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
		echo "${GREEN}goimports로 포매팅 완료${NC}"; \
	else \
		gofmt -w .; \
		echo "${GREEN}gofmt로 포매팅 완료${NC}"; \
		echo "${YELLOW}더 나은 포매팅을 위해 goimports를 설치하세요: go install golang.org/x/tools/cmd/goimports@latest${NC}"; \
	fi

## vet: go vet 실행
vet:
	@echo "${YELLOW}go vet 실행 중...${NC}"
	@go vet ./...

## run: 애플리케이션 실행
run: build
	@echo "${YELLOW}애플리케이션 실행 중...${NC}"
	@./$(BINARY_NAME) --help

## install: 바이너리를 GOPATH/bin에 설치
install:
	@echo "${YELLOW}바이너리 설치 중...${NC}"
	@go install $(MAIN_FILE)
	@echo "${GREEN}바이너리가 설치되었습니다${NC}"

## clean: 생성된 파일들 정리
clean:
	@echo "${YELLOW}정리 중...${NC}"
	@rm -f $(BINARY_NAME)
	@rm -rf coverage/
	@go clean
	@echo "${GREEN}정리 완료${NC}"

## dev-deps: 개발에 필요한 도구들 설치
dev-deps:
	@echo "${YELLOW}개발 의존성 설치 중...${NC}"
	@echo "goimports 설치..."
	@go install golang.org/x/tools/cmd/goimports@latest
	@echo "golangci-lint 설치..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "${GREEN}개발 의존성 설치 완료${NC}"

## benchmark: 벤치마크 테스트 실행
benchmark:
	@echo "${YELLOW}벤치마크 테스트 실행 중...${NC}"
	@go test -bench=. -benchmem ./...

## ci: CI에서 사용할 명령어들 (포매팅, 테스트, 빌드)
ci: fmt vet test build
	@echo "${GREEN}CI 검사 완료${NC}"

## quality: 코드 품질 검사 (포매팅, 린팅, 테스트, 커버리지)
quality: fmt lint test-coverage
	@echo "${GREEN}코드 품질 검사 완료${NC}"

## all: 모든 검사와 빌드 실행
all: clean fmt vet lint test build
	@echo "${GREEN}모든 작업 완료${NC}"

# 기본 타겟
.DEFAULT_GOAL := help