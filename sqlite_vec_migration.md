# Migration Plan: Qdrant to SQLite-Vec

**Status:** Proposed (Red Team Recommendation)
**Author:** Principal Software Engineer
**Date:** December 30, 2025

## 1. Executive Summary
The current architecture uses Qdrant for vector storage and SQLite for metadata, creating a "Two Sources of Truth" sync problem. For Scriberr's scale (thousands of speakers, not millions), Qdrant adds unnecessary Docker complexity and network overhead. 

We will migrate to **`sqlite-vec`**, a high-performance vector search extension for SQLite. This will allow us to store identities and voice embeddings in a single ACID-compliant file.

## 2. Benefits
*   **Zero External Dependencies**: Removes the requirement for a Qdrant Docker container.
*   **Atomic Transactions**: Name changes and vector updates happen in a single SQL `COMMIT`.
*   **Simplified Backups**: The entire identity registry is contained within the standard `scriberr.db` file.
*   **Performance**: Eliminates REST/gRPC overhead between the app and the vector store.

---

## 3. Technical Changes

### 3.1 Database Schema Evolution
We will enhance the existing `speakers` table with a virtual vector table provided by `sqlite-vec`.

```sql
-- The Metadata Table (Existing)
CREATE TABLE speakers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at DATETIME,
    updated_at DATETIME
);

-- The Vector Table (New)
CREATE VIRTUAL TABLE speaker_embeddings USING vec0(
    speaker_id TEXT PRIMARY KEY,
    embedding FLOAT32[192] -- TitaNet Large vector size
);
```

### 3.2 Identification Engine Update
The `titanet_identify.py` script will be refactored to interact with SQLite instead of Qdrant.

*   **Before**: `client = QdrantClient(host="qdrant", ...)`
*   **After**: `conn = sqlite3.connect("data/scriberr.db"); conn.enable_load_extension(True); conn.load_extension("vec0")`

### 3.3 Go Repository Update
The `SpeakerRepository` will handle the vector search directly using SQL.

```go
// Search for closest speaker
query := `
    SELECT speaker_id, distance
    FROM speaker_embeddings
    WHERE embedding MATCH ?
    ORDER BY distance
    LIMIT 1
`
```

### 3.4 Alternative: Standard SQLite + NumPy (Zero-Extension)
For Scriberr's expected scale (<10,000 speakers), we can skip the `sqlite-vec` extension entirely and perform identification in-memory using NumPy.

*   **Storage**: Store centroids as `BLOB` in a standard table.
*   **Search**: Fetch all centroids into Python and use NumPy for matrix multiplication (Cosine Similarity).
*   **Pros**: Zero runtime dependencies (no C extensions to bundle), maximum portability, and trivial implementation.
*   **Cons**: O(N) search time (linear), though at 1k vectors this takes <1ms.

---

## 4. Implementation Phases

### Phase 1: Infrastructure Preparation
1.  Add `sqlite-vec` binary to the Docker image and dev environment.
2.  Update `internal/database/database.go` to load the extension on connection.
3.  Add the virtual table migration.

### Phase 2: Python Script Refactor
1.  Update `titanet_identify_v2.py` and `titanet_manage.py` to use `sqlite3` for vector operations.
2.  Remove all `qdrant-client` dependencies from the Python environment.

### Phase 3: Data Migration (The "Transfer")
Create a one-time migration script that:
1.  Connects to Qdrant.
2.  Scrolls through all points in the `speakers` collection.
3.  Inserts the IDs, Names, and Vectors into the new SQLite tables.
4.  Verifies the counts match.

### Phase 4: Cleanup
1.  Remove Qdrant service from `docker-compose.yml`.
2.  Delete `internal/transcription/adapters/titanet_adapter.go` logic related to Qdrant host management.
3.  Update documentation to reflect the new "Single File" architecture.

---

## 5. Risk Assessment & Mitigations

| Risk | Impact | Mitigation |
| :--- | :--- | :--- |
| **Extension Loading** | High | Ensure `sqlite-vec` shared library is properly bundled in the Docker image. |
| **Performance at Scale** | Low | `sqlite-vec` is highly optimized for k-NN search; benchmarks show it handles 10k+ vectors with sub-millisecond latency. |
| **Data Loss during Transfer** | High | Run the migration script in "verify-only" mode before deleting the Qdrant volume. |

## 6. Verdict
This move completes the "Scriberr Simplification." By consolidating all state into SQLite, we make the app more robust, easier to deploy, and mathematically simpler to maintain.
