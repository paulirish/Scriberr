# Speaker Architecture & Identity Resolution

This document defines the architecture for speaker identification, naming, and persistence in Scriberr following the December 2025 refactor.

## 1. Core Philosophy: Identity Resolution at Read-Time
Scriberr has moved from a **Write-Time Baking** model (where names were frozen in JSON) to a **Read-Time Resolution** model. 

*   **Stable IDs**: The `transcript` JSON is an immutable record of *who* spoke and *when*. It stores stable, prefixed IDs (e.g., `global:<uuid>` or `local:speaker_00`).
*   **Dynamic Names**: Display names are resolved by the API at the moment of the request. This allows names to change globally without ever modifying historical JSON blobs.

---

## 2. The Life of a Speaker

The speaker pipeline follows four distinct stages from audio processing to UI display:

### Stage 1: Diarization (The Turn)
When a recording is processed, the diarization engine (Pyannote or Sortformer) identifies speaker turns. 
*   **Action**: Assigns temporary labels like `speaker_0`, `speaker_1`.
*   **Result**: A `DiarizationResult` containing segments marked with these local labels.

### Stage 2: Identification (The Identity)
If TitaNet is enabled, the system attempts to match the local voice prints against the global database.
*   **Comparison**: The TitaNet engine extracts embeddings for each local speaker and queries the **Qdrant Vector DB**.
*   **Match Found**: The local label is replaced with a stable ID: `global:<qdrant-uuid>`.
*   **No Match (Enrollment)**: A new identity is created in Qdrant. A new UUID is generated, and the speaker is marked as `global:<new-uuid>`.
*   **Fallback**: If identification is disabled or fails, the speaker remains `local:speaker_0`.

### Stage 3: Persistence (The Record)
The `UnifiedTranscriptionService` saves the results to SQLite:
*   **Transcript JSON**: All `speaker` fields are updated to use the prefixed stable IDs.
*   **Global Registry**: Any new `global:` identities are mirrored from Qdrant into the SQLite `speakers` table (the local name cache).
*   **Participant Index**: Unique IDs are saved to the `speaker_segments` table, creating a relational index for the Overview UI.

### Stage 4: Resolution (The Display)
When the UI requests a transcript or a list of recordings, the `SpeakerResolver` performs a hierarchical lookup:
1.  **Job Override**: Checks `speaker_mappings` (e.g., "Speaker 0" -> "Dad" for *this* job).
2.  **Global Registry**: Checks the `speakers` table (e.g., `<uuid>` -> "John Doe").
3.  **Clean Fallback**: Formats the ID for display (e.g., `local:speaker_0` -> "Speaker 0").

---

## 3. Renaming Mechanics

The new architecture enables high-performance renaming without risks of data corruption.

### Local Rename (Job-Specific)
*   **UI Action**: User renames a speaker on the transcript page.
*   **Mechanism**: A record is created/updated in the `speaker_mappings` table.
*   **Effect**: Only affects that specific Job ID. The underlying JSON remains untouched.

### Global Rename (System-Wide)
*   **UI Action**: User renames a speaker in the global "Speakers" settings.
*   **Mechanism**: **O(1) Operation**.
    1.  Update the name in the **Qdrant Vector DB** (for future matches).
    2.  Update the name in the SQLite **`speakers` table**.
*   **Effect**: Every transcript containing that `global:<uuid>` instantly displays the new name upon the next refresh. **Zero JSON sweeps are performed.**

---

## 4. Performance & Scale

| Feature | Legacy Architecture | New Architecture |
| :--- | :--- | :--- |
| **Renaming 1 Speaker** | $O(N)$ JSON Parsers (Slow/Unsafe) | **$O(1)$ SQL Update (Instant)** |
| **Overview UI Names** | $O(N)$ JSON Parsers (CPU Heavy) | **Relational Join (Zero JSON Parsing)** |
| **Data Integrity** | Prone to partial updates/corruption | High (Immutable JSON IDs) |
| **Identity Source** | Fragmented (JSON, Qdrant, Maps) | Centralized (SQLite Registry) |

## 5. Summary of Data Sources

*   **Qdrant**: Stores high-dimensional voice embeddings (Centroids) for vector matching.
*   **SQLite `speakers`**: The global cache for IDs to Names.
*   **SQLite `speaker_mappings`**: User-defined overrides for specific recordings.
*   **SQLite `speaker_segments`**: Relational index linking Job IDs to Speaker IDs.
