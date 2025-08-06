# Code Style and Conventions

## Go Language Conventions
This project follows standard Go conventions and best practices:

### Naming Conventions
- **Package names**: lowercase, single word when possible
- **Exported functions/types**: PascalCase (e.g., `SessionData`, `CollectionConfig`)
- **Unexported functions/variables**: camelCase (e.g., `cfgFile`, `outputPath`)
- **Constants**: PascalCase for exported, camelCase for unexported
- **Interfaces**: Often end with "er" suffix (though not strictly followed)

### Structure Tags
All structs use consistent JSON and YAML tags:
```go
type SessionData struct {
    ID          string            `json:"id" yaml:"id"`
    Source      CollectionSource  `json:"source" yaml:"source"`
    Timestamp   time.Time         `json:"timestamp" yaml:"timestamp"`
    // ...with omitempty for optional fields
    Title       string            `json:"title,omitempty" yaml:"title,omitempty"`
}
```

### Error Handling
- Always handle errors explicitly
- Use fmt.Errorf for error wrapping with %w verb
- Provide contextual error messages
- Fallback gracefully when possible (e.g., dummy data when real collection fails)

### Package Organization
- `cmd/`: CLI command implementations
- `internal/`: Private packages not intended for external use
- `pkg/`: Public packages that could be imported by other projects
- Clear separation of concerns between packages

### Documentation
- Package-level documentation for all packages
- Exported functions and types have godoc comments
- Korean comments are used for user-facing messages and CLI help text
- English comments for internal code documentation

### CLI Design Patterns
- Uses Cobra CLI framework consistently
- Mutual exclusive flags where appropriate
- Verbose mode for debugging
- Graceful error handling with helpful messages
- Configuration validation before execution