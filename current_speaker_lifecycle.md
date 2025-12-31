# Current Speaker Lifecycle in Scriberr

This document outlines the current architecture for speaker identification, naming, and persistence as of December 30, 2025. It identifies the "two sources of truth" problem and the lifecycle of speaker data from transcription to UI display.

## Data Sources

Currently, speaker information lives in three distinct places:

1.  **Transcript JSON (`transcription_jobs.transcript`)**:
    *   A JSON-serialized string stored in the database.
    *   Contains a `segments` array. Each segment has a `speaker` field.
    *   This field is used as both a **unique identifier** and a **display name**.
2.  **Speaker Mappings Table (`speaker_mappings`)**:
    *   A database table that maps an `original_speaker` ID to a `custom_name` for a **specific job**.
    *   Used as a local override layer in the UI.
3.  **Global Identity Store (Qdrant Vector DB)**:
    *   Stores speaker embeddings (centroids) and their global names.
    *   Managed by TitaNet during the identification phase.

---

## Lifecycle of a Speaker

### 1. Diarization (The Birth)
When a recording is processed, the diarization engine (Pyannote or Sortformer) identifies speaker turns. It assigns local labels like `speaker_00`, `speaker_01`.

### 2. Identification (The Global Sync)
If TitaNet is enabled:
*   Local segments are compared against the Qdrant vector database.
*   **If matched**: The local label (e.g., `speaker_00`) is replaced with the **Global Name** (e.g., "John Doe") or **Global ID** (e.g., `Speaker-abc12345`) inside the `DiarizationResult`.
*   **If no match**: A new identity is created in Qdrant, and the local label is replaced with a new unique ID.

### 3. Merging and Persistence
The `UnifiedTranscriptionService` merges the diarization results back into the transcript segments.
*   The final `Transcript` JSON is saved to the database.
*   **Crucial**: At this point, the `segments[].speaker` field in the JSON contains the "best known" name or ID.

### 4. Local Renaming (Job-level)
In the UI, a user can rename a speaker for just that recording.
*   This creates/updates a record in the `speaker_mappings` table.
*   It **does not** modify the transcript JSON.

### 5. Global Renaming (The Heavy Lift)
When a user renames a speaker globally (via the Speakers settings):
1.  The name is updated in the Qdrant vector database.
2.  The `SpeakerService` performs a **retroactive update**:
    *   It iterates through **every single transcription job** in the SQLite database.
    *   For each job, it parses the `Transcript` JSON string.
    *   It searches all `segments` for the `oldName` and replaces it with the `newName`.
    *   It re-serializes and saves the JSON back to the database.

---

## The "Two Sources of Truth" Problem

There is no single "Speaker ID" that remains constant and separate from the "Display Name".

1.  **The JSON is "Mutable"**: The `transcript` JSON blob, which should ideally be a stable record of the transcription, is treated as a mutable store for display names. This makes global renames extremely expensive ($O(N)$ jobs) and prone to data corruption if the JSON parsing fails.
2.  **Naming Conflicts**: If the JSON contains "John Doe" but the `speaker_mappings` table says "John D.", the UI has to decide which one to show. 
3.  **API Redundancy**:
    *   `GET /api/v1/transcription/:id/transcript` returns names inside the JSON.
    *   `GET /api/v1/transcription/:id/speakers` returns a separate list of mappings.
    *   The frontend must manually merge these two sources to show the "correct" name.

## Future Refactor Goals

*   **Stable IDs**: The transcript JSON should only ever store a stable `speaker_id` (e.g., a UUID or a consistent "local_0" / "global_uuid" format).
*   **Separation of Concerns**: The transcript JSON is for *content and timing*. The `speaker_mappings` (or a new `speakers` table) is for *identity and metadata*.
*   **Identity Layer**: A central `speakers` table in SQLite should act as the source of truth for display names, linked to Qdrant for vector matching.
*   **Dynamic Resolution**: The UI and Export functions should resolve `speaker_id` to `display_name` at runtime, rather than relying on string replacement inside JSON blobs.
