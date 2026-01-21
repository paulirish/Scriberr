# Unified Type Sharing: Go Backend to TypeScript Frontend

This document outlines the effort to align frontend TypeScript types with the Go backend "source of truth" and establishes best practices for maintaining this synchronization.

## Context
The Scriberr codebase had fragmented type definitions in the frontend (e.g., `useAudioFiles.ts`, `useAudioDetail.ts`) that often diverged from the backend models (`internal/models/transcription.go`) and the API documentation (`api-docs/swagger.json`). The goal is to centralize these definitions and ensure they accurately reflect the data structures returned by the Gin API handlers.

## What Has Been Done
1.  **Centralized Type Repository**: Created `web/frontend/src/types/transcription.ts` to house core models.
    *   **TranscriptionJob**: The primary model, matching the backend Go struct exactly.
    *   **AudioFile**: An alias for `TranscriptionJob` to maintain UI nomenclature while remaining backed by the canonical type.
    *   **WhisperXParams**: Fully mapped from the backend, including all transcription and diarization flags.
    *   **ExecutionData**: Aligned with the enhanced JSON response from the job execution handler.
2.  **Automated Generation Path**: Verified that `swagger-typescript-api` can generate types from the existing Swagger 2.0 spec.
    *   Generated `web/frontend/src/types/api-generated.ts` using:
      ```bash
      npx swagger-typescript-api generate -p api-docs/swagger.json -o web/frontend/src/types -n api-generated.ts --no-client
      ```
    *   *Note: `openapi-typescript` was attempted but failed as it requires OpenAPI 3.0.*
3.  **Refactoring**: 
    *   Updated `useAudioFiles.ts` and `useAudioDetail.ts` hooks to import from the centralized `transcription.ts`.
    *   Fixed component regressions in `AudioFilesTable.tsx`, `AudioFilesWeekCalendar.tsx`, and `AudioFilesMonthCalendar.tsx` caused by flattening the paginated data structure.
    *   Unified `TranscriptionProfile` and updated usages in `QuickTranscriptionDialog.tsx`.

## Best Practices
1.  **Single Source of Truth**: Always treat `internal/models/*.go` as the source of truth for data structures.
2.  **Centralized Imports**: Never define API-bound interfaces locally within a hook or component. Always export/import from `web/frontend/src/types/`.
3.  **Computed Fields**: Interfaces in `transcription.ts` include optional fields like `speakers` or `duration` which are calculated on the frontend (e.g., parsed from titles or JSON transcripts). Keep these clearly documented as UI-only.
4.  **Verification**: Always run `npm run type-check` (which executes `tsc -b`) after modifying types to catch cascading breaks in the UI.

## Roadmap for Future Agents
- [ ] **Complete Hook Migration**: Review remaining hooks in `web/frontend/src/features/` and ensure they use the centralized types.
- [ ] **Automated Build Integration**: Add a script to `web/frontend/package.json` that regenerates `api-generated.ts` whenever the backend Swagger spec is updated.
- [ ] **OpenAPI 3.0 Migration**: Long-term, update `swag` annotations to support OpenAPI 3.0 to unlock more modern TS generation tooling.
- [ ] **Cleanup**: Once all migrations are complete, use `api-generated.ts` as the base for `transcription.ts` to minimize manual maintenance.
