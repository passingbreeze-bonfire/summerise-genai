# Summerise GenAI Project Overview

## Purpose
Summerise GenAI is a Go-based CLI tool that collects data from multiple AI CLI tools (Claude Code, Gemini CLI, Amazon Q CLI) and converts them into structured markdown documents. It serves as an automation tool for summarizing AI tool activities and generating comprehensive reports.

## Technology Stack
- **Language**: Go 1.24.5
- **CLI Framework**: Cobra CLI
- **Configuration**: YAML-based with gopkg.in/yaml.v3
- **Architecture**: Modular design with clear separation of concerns
- **MCP Integration**: Model Context Protocol agent support for extensibility

## Project Structure
```
summerise-genai/
├── main.go                 # Application entry point
├── cmd/                    # CLI commands (Cobra framework)
│   ├── root.go            # Main CLI configuration
│   ├── collect.go         # Data collection command
│   ├── export.go          # Markdown export command
│   └── config.go          # Configuration management
├── internal/              # Internal packages
│   ├── collector/         # Data collection implementations
│   ├── config/           # Configuration management
│   ├── processor/        # Data processing pipeline
│   └── exporter/         # Markdown generation
├── pkg/                   # Public packages
│   ├── models/           # Data models and types
│   └── agents/           # MCP agents
├── configs/              # Configuration files
└── go.mod               # Go module dependencies
```

## Key Components
1. **Data Models**: Well-defined structs with JSON/YAML tags for serialization
2. **Collectors**: Pluggable data collection system for different AI tools
3. **Processor**: Data transformation and markdown generation pipeline
4. **Configuration**: YAML-based configuration with validation
5. **CLI Interface**: User-friendly command-line interface with help and validation