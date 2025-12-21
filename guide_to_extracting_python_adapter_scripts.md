# Guide to Extracting Python Adapter Scripts

This guide explains how to extract inline Python scripts and configuration files from Go adapters into separate source files, using the `embed` package to bundle them.

## 1. Create a Scripts Directory

Create a directory to hold the scripts corresponding to the adapter.
Standard location: `internal/transcription/adapters/scripts/<adapter_name>`

```bash
mkdir -p internal/transcription/adapters/scripts/<adapter_name>
```

## 2. Extract Files

Copy the inline content from the Go file into separate files in the new directory.

*   `transcribe.py`: The main transcription script.
*   `pyproject.toml`: The dependency configuration (if using `uv` or similar).
*   Any other scripts (e.g., `transcribe_buffered.py`).

## 3. Update the Go Adapter

### Import `embed`

Add the `embed` package to the imports.

```go
import (
    "embed"
    // ... other imports
)
```

### Embed the Scripts

Add the `//go:embed` directive and a variable to hold the file system. This should be at the package level.

```go
//go:embed scripts/<adapter_name>/*
var <adapterName>Scripts embed.FS
```

**Note:** The path in `//go:embed` is relative to the Go file.

### Read Embedded Files

Replace the inline strings with calls to `ReadFile`.

**Before:**
```go
scriptContent := `... inline python code ...`
scriptPath := filepath.Join(p.envPath, "transcribe.py")
if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil { ... }
```

**After:**
```go
scriptContent, err := <adapterName>Scripts.ReadFile("scripts/<adapter_name>/transcribe.py")
if err != nil {
    return fmt.Errorf("failed to read embedded transcribe.py: %w", err)
}

scriptPath := filepath.Join(p.envPath, "transcribe.py")
if err := os.WriteFile(scriptPath, scriptContent, 0755); err != nil { ... }
```

Repeat this for all extracted files (`pyproject.toml`, etc.).

## 4. Update .gitignore

Ensure that Python artifacts are ignored.

```gitignore
# Python
__pycache__/
*.py[cod]
*$py.class
```

## 5. Verify

Run the application or tests to ensure the adapter still functions correctly. The files should be written to the runtime environment path (e.g., `data/<adapter>-env`) just as they were before.
