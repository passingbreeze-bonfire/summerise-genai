# Suggested Commands for Summerise GenAI Development

## Build and Development Commands

### Core Development
- `go build -o summerise-genai` - Build the main application
- `go test ./...` - Run all tests
- `goimports -w .` - Format Go code and organize imports
- `golangci-lint run` - Run Go linter (if available)

### Application Usage
- `./summerise-genai --help` - Show help and available commands
- `./summerise-genai config --show` - Display current configuration
- `./summerise-genai config --init` - Initialize default configuration
- `./summerise-genai config --validate` - Validate configuration file
- `./summerise-genai collect --all --verbose` - Collect data from all AI tools
- `./summerise-genai export --output ./summary.md` - Export collected data to markdown

### System Utilities (macOS/Darwin)
- `ls -la` - List files with details
- `find . -name "*.go"` - Find Go source files
- `grep -r "pattern" .` - Search for patterns in files
- `git status` - Check git repository status
- `git add .` - Stage all changes
- `git commit -m "message"` - Commit changes

### Development Workflow
1. Make code changes
2. Run `goimports -w .` to format code
3. Run `go test ./...` to ensure tests pass
4. Run `go build -o summerise-genai` to verify build
5. Test CLI functionality with `./summerise-genai --help`
6. Commit changes with appropriate git commands

### Debugging and Analysis
- `go run main.go --help` - Run application directly
- `go mod tidy` - Clean up module dependencies
- `go mod download` - Download dependencies
- `./summerise-genai --verbose` - Run with verbose output for debugging