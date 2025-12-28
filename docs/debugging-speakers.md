# Debugging Speakers & Segments

This guide provides quick commands and tips for debugging the speaker identification and segment persistence system.

## Database Inspection (SQLite)

Check the distribution of segments across speakers:
```bash
sqlite3 data/scriberr.db "SELECT speaker_id, COUNT(*) as segment_count FROM speaker_segments GROUP BY speaker_id ORDER BY segment_count DESC;"
```

Verify that segments are correctly linked to transcription jobs:
```bash
sqlite3 data/scriberr.db "SELECT speaker_id, transcription_job_id, start, end FROM speaker_segments LIMIT 20;"
```

Check if a specific speaker has any "Human Name" mapping (SpeakerMappings table):
```bash
sqlite3 data/scriberr.db "SELECT * FROM speaker_mappings WHERE original_speaker = 'Speaker-1aba688e';"
```

## API Verification

Verify the segments endpoint for a specific speaker:
```bash
curl -s -H "X-API-Key: $SCRIBERR_API_KEY" http://localhost:8080/api/v1/speakers/Speaker-1aba688e/segments | jq .
```

Check the global speaker list:
```bash
curl -s -L -H "X-API-Key: $SCRIBERR_API_KEY" http://localhost:8080/api/v1/speakers/ | jq .
```

## Backfilling Data
If you need to re-run the backfill (e.g., after deleting the database or adding new legacy transcripts), use the provided tool:
```bash
go run cmd/backfill_segments/main.go
```

## Common Issues

### "Unknown" Speaker ID
If segments show up as `Unknown`, it means the transcript JSON for that job did not have a `speaker` field assigned to the segments. This usually happens if diarization was disabled for that job.

### Duplicate Segments
The backfill script currently doesn't check for existing segments before inserting. If you run it multiple times, you will get duplicates in the `speaker_segments` table. To clear them:
```bash
sqlite3 data/scriberr.db "DELETE FROM speaker_segments;"
```
