# Architecture and Design Patterns

## Overall Architecture
Summerise GenAI follows a layered architecture with clear separation of concerns:

1. **Presentation Layer** (`cmd/`): CLI interface using Cobra framework
2. **Business Logic Layer** (`internal/`): Core application logic
3. **Data Layer** (`pkg/models/`): Data structures and models
4. **Configuration Layer** (`configs/`): YAML-based configuration management

## Key Design Patterns

### 1. Command Pattern (Cobra CLI)
- Each CLI command is implemented as a separate file in `cmd/`
- Commands are self-contained with their own flags and validation
- Shared functionality is abstracted to the root command

### 2. Strategy Pattern (Data Collection)
- Different collectors implement the same interface
- Pluggable collection strategies for different AI tools
- Fallback mechanism when real collection fails

### 3. Pipeline Pattern (Data Processing)
- Data flows through collect → process → export stages
- Each stage is responsible for one transformation
- Error handling at each stage with graceful degradation

### 4. Configuration Pattern
- YAML-based external configuration
- Validation and default value setting
- Path expansion for user directory references (~/.claude)

## MCP Agent Integration
- Model Context Protocol agents provide extensibility
- Configured through YAML files
- Support for file system management, markdown processing, etc.
- Future-ready for additional agent types

## Error Handling Philosophy
- Fail fast on configuration errors
- Graceful degradation for data collection failures
- Detailed error messages for user debugging
- Fallback to dummy data when appropriate for demo purposes

## Extensibility Points
1. **New Collectors**: Add to `internal/collector/` with interface implementation
2. **Export Formats**: Extend `internal/exporter/` for new output types
3. **MCP Agents**: Configure new agents in `configs/agents.yaml`
4. **Data Models**: Extend `pkg/models/` for new data types

## Code Organization Principles
- Internal packages are truly internal (not for external use)
- Public packages in `pkg/` could theoretically be imported by other projects
- Clear dependency direction: cmd → internal → pkg
- No circular dependencies between packages