# Speaker Architecture: Compatibility & Bridge Strategy

**Status:** Technical Context for Upstream Reconciliation
**Date:** December 30, 2025
**Source Branch:** `speaker-refactor` / `identity-evolution`
**Target Branch:** `main` (Upstream)

## 1. The Architectural Gap

The core conflict between these branches is a fundamental shift in **Identity Ownership**.

### Legacy Model (`main` branch)
*   **Write-Time Baking**: Speaker names (e.g., `"speaker_00"`, `"John Doe"`) are written directly into the `transcript` JSON string during the save process.
*   **No Central Registry**: There is no `speakers` table. Identity only exists as a string inside a JSON blob or a temporary `speaker_mappings` record.
*   **O(N) Mutations**: Renaming a speaker requires parsing and rewriting every transcript in the database.
*   **CPU-Heavy Lists**: The Overview UI must parse the `transcript` JSON of every recording just to show who participated.

### Modern Model (Your Branch)
*   **Read-Time Resolution**: The `transcript` JSON stores stable, prefixed IDs (e.g., `global:<uuid>`, `local:speaker_00`).
*   **Relational Registry**: A global `speakers` table in SQLite caches names for fast lookup.
*   **O(1) Mutations**: Global renames happen in one SQL row. Transcripts reflect the change instantly at read-time via a Resolver.
*   **Relational Aggregation**: The Overview UI uses a SQL join to list participants. **Zero JSON parsing required.**

---

## 2. Code Comparison (Snippets from `main`)

### 2.1 Persistence Layer (`internal/transcription/unified_service.go`)
On `main`, saving is a "dump and forget" operation. It lacks the seeding logic for the registry.

```go
// From main branch
func (u *UnifiedTranscriptionService) saveTranscriptionResults(jobID string, result *interfaces.TranscriptResult) error {
    resultJSON, err := u.convertTranscriptResultToJSON(result)
    // ... error handling ...
    return u.jobRepo.UpdateTranscript(context.Background(), jobID, resultJSON)
}
```

### 2.2 API Layer (`internal/api/handlers.go`)
On `main`, the API serves raw data, forcing the frontend to handle all metadata resolution.

```go
// From main branch
func (h *Handler) GetTranscript(c *gin.Context) {
    // ... fetch job ...
    var transcript interface{}
    json.Unmarshal([]byte(*job.Transcript), &transcript)
    c.JSON(http.StatusOK, gin.H{"transcript": transcript}) // Serves raw JSON strings
}
```

---

## 3. The Bridge Strategy: Prefixing & Resolving

To maintain compatibility with `main` while keeping the performance of your branch, we use a **Hierarchical Resolver**.

### 3.1 Prefix Logic
The system differentiates identity types by string prefixes:
*   `global:<uuid>`: Match against the SQLite `speakers` table.
*   `local:<id>`: Match against the Job's `speaker_mappings`.
*   `legacy:*` (or any raw string): Treated as a name-is-ID fallback (Legacy Compatibility).

### 3.2 The SpeakerResolver
The `SpeakerResolver` in `internal/service/speaker_resolver.go` is the "glue." It should be the ONLY way transcripts are served.

```go
func (r *SpeakerResolver) ResolveSpeakerID(id string, ...) string {
    if name, ok := mappings[id]; ok { return name } // Job Override
    if name, ok := globalSpeakers[id]; ok { return name } // Global Registry
    
    // Legacy Fallback: If it's a raw string from 'main' branch data, return it
    if !strings.HasPrefix(id, "global:") && !strings.HasPrefix(id, "local:") {
        return id 
    }
    // Clean Fallback for new IDs
    return formatForDisplay(id) 
}
```

---

## 4. Conflict Resolution Guide

When merging `main` into your branch, or vice versa, keep these "Red Team" rules in mind:

1.  **Don't Revert the Registry**: If `main` removes the `speakers` table or the `SpeakerMapping` preload in `FindWithAssociations`, the Resolver will break. Ensure these schema changes are preserved.
2.  **Protect the ID Prefixes**: If identification logic is merged from `main`, ensure it is updated to return `global:<id>` instead of raw names.
3.  **Renaming is the Trap**: `main` likely still has a service method that tries to sweep JSON. **Nuke it.** If you let an $O(N)$ sweep run on data that has already been migrated to $O(1)$ stable IDs, you will corrupt the JSON.
4.  **Overview API**: The `GetSpeakersByJobIDs` method in the repository is a "pure win." Even if `main` doesn't use the full Identity model, this method can be used to speed up their UI by querying the `speaker_segments` table.

## 5. Summary of Relational Tables (Registry Layer)
*   **`speakers`**: `id` (PK, "global:..."), `name` (Display Name).
*   **`speaker_segments`**: `transcription_job_id`, `speaker_id` (The link for SQL Joins).
*   **`speaker_mappings`**: `transcription_job_id`, `original_speaker`, `custom_name` (Job-level overrides).
