# Contributing to 1stcrack

Thank you for your interest in contributing to **1stcrack**. We welcome contributions that improve reliability, performance, documentation, and operational clarity.

---

## Contribution Process

1. **Fork the Repository**: Create a personal fork on GitHub.
2. **Branch Strategy**: Create a dedicated feature or bugfix branch:
   - `feat/<feature-name>`: New functionality or enhancements.
   - `fix/<issue-description>`: Bug fixes and error handling corrections.
   - `perf/<optimization-target>`: Performance and memory optimizations.
   - `docs/<doc-name>`: Documentation updates.
3. **Develop & Test**: Implement changes following Go idioms and project architecture. Write unit and integration tests covering new logic.
4. **Run Validation Sequence**: Ensure all linters, formatting checks, and tests pass with zero warnings.
5. **Submit Pull Request**: Open a pull request against the `main` branch with a clear description of changes and test evidence.

---

## Code Quality Standards

### 1. Go Documentation Standard
Every exported and unexported symbol (types, functions, methods, constants, variables) must be documented with a doc comment in English starting with the symbol name:

```go
// WeightMg represents a coffee weight in milligrams.
type WeightMg int64
```

### 2. Linting & Formatting
Ensure code adheres to `.golangci.yml` lint rules:

```bash
# Format code
go fmt ./...

# Run vet static analysis
go vet ./...
```

### 3. Layer Boundary Integrity
- Never import `internal/repository` or `internal/database` directly into `internal/domain` or `internal/cli`.
- Keep the `domain` package purely in-memory with zero external dependencies.
- Ensure all business validation resides in the `service` layer.

---

## Code of Conduct

All contributors are expected to adhere to the [Code of Conduct](CODE_OF_CONDUCT.md). Please report unacceptable behavior to **dzulkiflianwar2@gmail.com**.
