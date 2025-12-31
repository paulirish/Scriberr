# Speaker Architecture Refactor: From String-Matching to Identity Resolution

**Status:** Proposed (Red Team Review)
**Author:** Principal Software Engineer
**Date:** December 30, 2025

## 1. The Problem (The "Ticking Time Bomb")
The current system treats a serialized JSON blob as a database. This has led to:
*   **Referential Integrity Loss:** Display names are baked into transcripts. Renaming a speaker requires an $O(N)$ sweep of every job in the system.
*   **Performance Bottlenecks:** Getting a list of speakers for an "Overview UI" requires parsing massive JSON blobs for every single recording.
*   **Data Corruption Risk:** Manual string-replacement in JSON is prone to escaping errors and partial updates.
*   **No Source of Truth:** Identity is fragmented across Qdrant, SQLite mappings, and the JSON itself.

---

## 2. The Architectural Shift: Read-Time Resolution
We are moving from a **Write-Time Baking** model (where names are frozen in JSON) to a **Read-Time Resolution** model (where IDs are resolved to names at the API layer).

### 2.1 The New Identity Layer (SQLite)
A new `speakers` table will act as the local source of truth and a high-performance cache for Qdrant metadata.

```go
type Speaker struct {
    ID        string    `gorm:"primaryKey"` // Format: "global:<uuid>"
    Name      string    `gorm:"not null"`   // The Global Display Name
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### 2.2 Stable Identifiers in JSON
The `transcript` JSON will no longer store names. It will store stable, immutable IDs.
*   `global:<uuid>`: Linked to a persistent identity in Qdrant/SQLite.
*   `local:<diarization_id>`: (e.g., `local:speaker_00`) For unidentified speakers.

### 2.3 The Resolution Hierarchy
When the API serves a transcript, it resolves IDs using this priority:
1.  **Job Override:** `speaker_mappings` table (User renamed "Speaker 1" to "Dad" for *this* recording).
2.  **Global Identity:** `speakers` table (Global name "John Doe").
3.  **Fallback:** The raw ID (e.g., "Speaker 00").

---

## 3. Implementation Plan

### Phase 1: Data Migration (The "Final Sweep")
We must reconcile existing data without breaking the system.
1.  Create the `speakers` table.
2.  Run a migration script that:
    *   Parses every `transcript` JSON.
    *   Extracts unique names.
    *   Generates/Assigns UUIDs and seeds the `speakers` table.
    *   Replaces names in JSON with `global:<uuid>` or `local:<id>`.
    *   **This is the last time the system ever performs a retroactive sweep.**

### Phase 2: Identification Engine Update
Update `titanet_identify.py` and the `TitanetAdapter`:
*   **Identification:** Returns the UUID from Qdrant, not the name.
*   **Persistence:** The adapter ensures that when a new identity is enrolled in Qdrant, it is mirrored into the SQLite `speakers` table.

### Phase 3: API & Service Refactor
1.  **SpeakerService:** Remove the $O(N)$ loop in `RenameSpeaker`. Global renaming now becomes a single SQL `UPDATE` on the `speakers` table.
2.  **Read-time Resolver:** Create a Go internal service that takes a `JobID` and a `Transcript` and returns a JSON-compatible object with names resolved.
3.  **Overview API:** Create a lightweight `GET /api/v1/transcriptions/speakers` that queries the `SpeakerSegment` table joined with the `speakers` table. **Zero JSON parsing required.**

---

## 4. Benefits

### Performance
*   **Global Renames:** Decreased from minutes (or timing out) to **milliseconds**.
*   **Overview UI:** Speaker lists for 100+ recordings can be fetched in a single SQL query instead of parsing 100+ blobs.

### Reliability
*   **Consistency:** Changing a name in the settings page instantly reflects in every transcript, past and future.
*   **Safety:** The `transcript` JSON becomes a "read-only" record of timing and content, reducing the surface area for corruption.

### Maintainability
*   Eliminates the "Two Sources of Truth" problem.
*   The frontend no longer has to "merge" sources; the backend provides a single, resolved view.
