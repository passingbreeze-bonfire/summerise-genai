# Task Completion Checklist

## When Completing Any Development Task

### 1. Code Quality Checks
- [ ] Run `goimports -w .` to format all Go files
- [ ] Run `go test ./...` to ensure all tests pass
- [ ] Run `go build -o summerise-genai` to verify successful build
- [ ] Check for any compiler warnings or errors
- [ ] Run `golangci-lint run` if available for additional quality checks

### 2. Functionality Testing
- [ ] Test CLI commands manually:
  - [ ] `./summerise-genai --help` works correctly
  - [ ] `./summerise-genai config --validate` passes
  - [ ] New functionality works as expected with verbose mode
- [ ] Verify error handling works correctly
- [ ] Test edge cases and invalid inputs

### 3. Configuration and Documentation
- [ ] Update configuration files if needed
- [ ] Update README.md if public interface changed
- [ ] Update CLAUDE.md if development guidance changed
- [ ] Add or update relevant comments in code

### 4. Git Workflow
- [ ] Stage changes: `git add .`
- [ ] Review changes: `git diff --staged`
- [ ] Commit with descriptive message: `git commit -m "descriptive message"`
- [ ] Consider if changes should be pushed to remote

### 5. Gemini CLI Collaboration (if applicable)
- [ ] Review code changes with Gemini CLI for quality improvement
- [ ] Use structured prompts for consistent review quality
- [ ] Document any architectural decisions or trade-offs

### 6. Integration Verification
- [ ] Ensure new features integrate well with existing codebase
- [ ] Verify MCP agent configurations still work
- [ ] Test data collection and export pipeline end-to-end

## Specific to This Project
- Always test CLI functionality after changes
- Verify YAML configuration parsing works correctly
- Ensure backward compatibility with existing config files
- Test with verbose mode to see detailed execution flow