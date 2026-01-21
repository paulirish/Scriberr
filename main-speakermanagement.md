# Speaker Management in Scriberr

This document details the current implementation of speaker management, including data storage, diarization adapter behavior, and user-initiated updates. This analysis serves as preparation for upgrading to a global identity system.

## 1. Database Schema and Storage

Speaker names are stored in the `speaker_mappings` table, which maps engine-generated labels to user-friendly names on a per-job basis.

### `speaker_mappings` Table
- **`id`**: Primary identifier (auto-incrementing integer).
- **`transcription_job_id`**: Links the mapping to a specific transcription job.
- **`original_speaker`**: The raw label provided by the diarization engine (e.g., `"SPEAKER_00"`) or the cleaned filename for multi-track jobs.
- **`custom_name`**: The display name assigned by the user (e.g., `"John Doe"`).
- **`created_at` / `updated_at`**: Timestamps for record management.

**Constraint**: There is a unique index on `(transcription_job_id, original_speaker)`. This ensures that for any given job, an engine label can only have one custom name.

## 2. Speaker Label Sources

The `original_speaker` value depends on the transcription method:

### A. Diarization Adapters (PyAnnote, Sortformer, WhisperX)
These adapters use machine learning models to identify different speakers in a single audio track.
- **PyAnnote / Sortformer**: Run Python scripts that produce RTTM or JSON output. They typically generate labels like `SPEAKER_00`, `SPEAKER_01`, etc.
- **WhisperX**: Produces a JSON result where each segment has a `speaker` field containing similar engine-generated labels.

### B. Multi-track Transcription
In multi-track mode, Scriberr treats each audio file as a distinct speaker.
- The `MultiTrackTranscriber` cleans up the filename to create the initial speaker name.
- **Example**: `recording_paul_irish.wav` becomes `Recording Paul Irish`.
- These cleaned names are stored as both `original_speaker` and `custom_name` in the `speaker_mappings` table immediately after transcription.

## 3. Speaker Rename Workflow

When a user updates speaker names in the UI, the following process occurs:

### Frontend (`SpeakerRenameDialog.tsx`)
1. **Fetch**: The dialog calls `GET /api/v1/transcription/:id/speakers` to get existing mappings.
2. **Merge**: It also scans the current transcript segments for any speaker labels that don't yet have an entry in the database (ensuring all detected speakers are listed).
3. **Edit**: The user provides new names for any of the detected speakers.
4. **Save**: The frontend sends a `POST` request to `/api/v1/transcription/:id/speakers` containing the complete list of mappings for that job.

### Backend (`speaker_mapping_repository.go`)
The repository uses a **"Delete and Replace"** strategy within a database transaction:
1. All existing mappings for the `transcription_job_id` are deleted.
2. The new set of mappings provided by the user is inserted.

## 4. Usage in Transcript Display

The system applies speaker names dynamically during rendering:
- The frontend fetches the mappings and creates a lookup object.
- **Logic**: `displayName = speakerMappings[segment.speaker] || segment.speaker;`
- This ensures that if no mapping exists, the raw engine label is shown.
- This same logic is used when exporting transcripts to SRT, TXT, or JSON formats.

### Relationship to the `transcript` JSON
The `transcription_jobs` table contains a `transcript` column storing a JSON blob. This blob includes an array of `segments`, each having a `speaker` field.
- **Source of Truth**: The `speaker` field in the JSON always stores the **original** engine label (e.g., `"SPEAKER_00"`).
- **Immutability**: The `transcript` blob is **never updated** after the initial transcription is completed. Even if a user renames a speaker, the JSON remains exactly as it was when first generated.
- **Read-Time Resolution vs. Write-Time Baking**:
    - Scriberr uses **Read-Time Resolution**: Custom names are fetched from the `speaker_mappings` table and merged with the transcript JSON only when the user views or exports it.
    - This is the opposite of **Write-Time Baking**, where names would be permanently written into the JSON. By avoiding baking, Scriberr remains high-performance (renaming is a simple SQL update, not a full JSON rewrite) and maintains a stable technical link to the diarization engine's output.

## 5. Life Cycle of a Speaker Identity

The following diagram and description trace how a speaker's identity is handled from the moment of detection until it is displayed to the user.

### ASCII Life Cycle Diagram

```text
+-----------------------+
|  Diarization Engine   | (PyAnnote, Sortformer, WhisperX)
| (Python Environment)  |
+-----------+-----------+
            |
            | 1. Detects voice activity and clusters segments.
            |    Assigns internal labels (e.g., "SPEAKER_00").
            v
+-----------+-----------+
|  Backend Adapter      | (Go Adapter: whisperx_adapter.go, etc.)
|  (Parsing Result)     |
+-----------+-----------+
            |
            | 2. Parses RTTM/JSON. Maps segments to TranscriptResult.
            |    Speaker field = "SPEAKER_00".
            v
+-----------+-----------+
|  Unified Service      | (unified_service.go)
|  (Storage)            |
+-----------+-----------+
            |
            | 3. Marshals TranscriptResult to JSON.
            |    Saves to transcription_jobs.transcript column.
            v
+-----------+-----------+
|  Frontend UI          | (TranscriptView.tsx)
|  (Resolution)         |
+-----------+-----------+
            |
            | 4. Fetches speaker_mappings table.
            |    Fetches transcript JSON.
            |    Applies mapping: "SPEAKER_00" -> "Alice".
            v
+-----------+-----------+
|  User Interaction     | (SpeakerRenameDialog.tsx)
|  (Update)             |
+-----------+-----------+
            |
            | 5. User renames "Alice" to "Bob".
            |    POST /api/v1/transcription/:id/speakers
            |    Backend DELETES old mappings and INSERTS new.
            |    *Note: The transcript JSON remains UNCHANGED.*
            v
+-----------+-----------+
|  Final Display        |
|  "Bob: Hello world"   |
+-----------------------+
```

### Identity Handling throughout the Life Cycle

1.  **Generation**: The identity starts as a purely mathematical cluster in a Diarization engine. It has no name, only a label used to group similar voice segments.
2.  **Persistence**: This label is treated as an immutable reference within the scope of that specific job's `transcript` JSON. It acts as a "foreign key" of sorts, but pointing to the `speaker_mappings` table instead of a global speaker table.
3.  **Mapping**: The link between the technical label and the human name is volatile and stored in `speaker_mappings`. This allows for high performance (no JSON parsing/editing on rename) and flexibility.
4.  **Display**: The identity is only "materialized" with a human name at the very last moment. This decoupling is what ensures that even if a job is re-processed or mappings are lost, the structural integrity of the transcript (who spoke when) is preserved via the original engine labels.

## 6. Considerations for Global Identities Upgrade

### Current Limitations
- **Per-Job Scope**: Mappings are isolated to individual jobs. Renaming "SPEAKER_00" to "Alice" in one recording does not help the system recognize "Alice" in a future recording, even if she is "SPEAKER_00" there as well.
- **Label Inconsistency**: Engine-generated labels like "SPEAKER_00" are arbitrary and can change between jobs or even if the same job is re-processed with different parameters.

### Recommendations for Upgrading
- **Global Identity Table**: Introduce a `speakers` or `identities` table that is independent of transcription jobs.
- **Linking Mechanism**: Add a `speaker_id` (nullable foreign key to the global table) to the `speaker_mappings` table.
- **Auto-Matching**:
    - **Multi-track**: Can be matched by filename or metadata.
    - **Diarization**: Requires **Speaker Embeddings**. PyAnnote and Sortformer can generate these. Storing a representative embedding for a global identity would allow the system to automatically link new "SPEAKER_XX" labels to known identities by comparing vector similarity.
- **Backwards Compatibility**:
    - The current `custom_name` field should be retained as a "per-job override" or as the primary source of truth for legacy data.
    - A migration could attempt to group `custom_name` values across jobs to seed the global identity table.
