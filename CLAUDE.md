# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

Summerise GenAI is a Go-based CLI tool that collects data from multiple AI CLI tools (Claude Code, Gemini CLI, Amazon Q CLI) and converts them into structured markdown documents. Built with Cobra CLI framework and designed for extensibility through MCP (Model Context Protocol) agents.

## Core Architecture

### Project Structure
```
summerise-genai/
├── cmd/                    # CLI commands (Cobra framework)
├── internal/               # Internal packages
│   ├── collector/         # Data collection implementations
│   ├── config/           # Configuration management
│   ├── processor/        # Data processing pipeline
│   └── exporter/         # Markdown generation
├── pkg/                   # Public packages
│   ├── models/           # Data models and types
│   └── agents/           # MCP agents
└── configs/              # Configuration files
```

## Development Commands

### Build and Test
- **Build**: `go build -o summerise-genai`
- **Test**: `go test ./...`
- **Format**: `goimports -w .`
- **Lint**: `golangci-lint run` (if available)

### Application Commands
- **Help**: `./summerise-genai --help`
- **Config**: `./summerise-genai config --show`
- **Collect**: `./summerise-genai collect --all --verbose`
- **Export**: `./summerise-genai export --output ./summary.md`

## Key Components

### 1. Data Models (`pkg/models/types.go`)
- `SessionData`: Core session data structure
- `Message`: Individual conversation messages
- `Command`: Executed command information
- `CollectionConfig`: Data collection configuration
- `ExportConfig`: Markdown export settings

### 2. Collectors (`internal/collector/`)
- `ClaudeCodeCollector`: Implemented for Claude Code data collection
- Gemini CLI and Amazon Q collectors: Planned for future implementation
- Supports JSON parsing, file system traversal, and pattern matching

### 3. Configuration System (`internal/config/`)
- YAML-based configuration with validation
- MCP agent management
- Path expansion and environment handling
- Default values and error handling

### 4. Processing Pipeline (`internal/processor/`)
- Data transformation and normalization
- Statistics generation
- Table of contents creation
- Code formatting and sanitization

### 5. Export System (`internal/exporter/`)
- Markdown template processing
- Multi-format output support
- Metadata and timestamp handling
- Custom field support

## MCP Agent Integration

The application supports Model Context Protocol agents for extensibility:
- File system management
- Markdown processing
- Multi-CLI integration
- Collaboration workflows

Configure agents in `configs/agents.yaml` with proper command and argument settings.

## Development Guidelines

### Adding New Collectors
1. Create new collector in `internal/collector/`
2. Implement the collector interface with `Collect()` method
3. Add configuration to `CLIToolConfig` structure
4. Update `collectFromSource()` in `cmd/collect.go`
5. Add tests and documentation

### Extending Export Formats
1. Add new template options to `ExportConfig`
2. Implement template logic in `internal/exporter/`
3. Update CLI flags and help text
4. Add examples to documentation

### Configuration Changes
1. Update structs in `internal/config/config.go`
2. Add validation logic in `Validate()` method
3. Set appropriate defaults in `SetDefaults()`
4. Update YAML schema documentation

## Collaboration with Gemini CLI

This project emphasizes collaboration with Gemini CLI for code quality improvement:

### Code Review Process
```bash
# Review implementation
gemini -p "다음 Go 코드를 검토해주세요: [code]"

# Architecture validation
gemini -p "시스템 아키텍처를 검토하고 개선사항을 제안해주세요"

# Performance optimization
gemini -p "성능 최적화 관점에서 이 코드를 검토해주세요"
```

### Best Practices
- Always review new features with Gemini CLI before committing
- Use structured prompts for consistent review quality
- Document review feedback and implementation decisions
- Maintain collaboration logs for future reference

## Testing Strategy

- Unit tests for core logic in collectors and processors
- Integration tests for CLI command functionality
- Configuration validation tests
- Mock implementations for external dependencies
- Test data fixtures for various AI tool formats

## Security Considerations

- Path traversal protection in file system access
- Input validation for configuration files
- Safe handling of sensitive data in logs
- Secure temporary file management
- Access control for MCP agents

## Performance Notes

- Streaming processing for large data sets
- Concurrent collection from multiple sources
- Memory-efficient data structures
- Configurable batch sizes and timeouts
- Resource cleanup and error recovery

## Future Extensions

- Real-time data collection monitoring
- Web-based dashboard interface
- Plugin architecture for custom collectors
- Advanced filtering and search capabilities
- Multi-language template support