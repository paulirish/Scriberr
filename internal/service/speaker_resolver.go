package service

import (
	"context"
	"strings"

	"scriberr/internal/repository"
	"scriberr/internal/transcription/interfaces"
)

// SpeakerResolver handles resolving speaker IDs to display names at read-time
type SpeakerResolver struct {
	mappingRepo repository.SpeakerMappingRepository
	speakerRepo repository.SpeakerRepository
}

// NewSpeakerResolver creates a new speaker resolver
func NewSpeakerResolver(mappingRepo repository.SpeakerMappingRepository, speakerRepo repository.SpeakerRepository) *SpeakerResolver {
	return &SpeakerResolver{
		mappingRepo: mappingRepo,
		speakerRepo: speakerRepo,
	}
}

// ResolveTranscript resolves speaker IDs to names within a transcript result
func (r *SpeakerResolver) ResolveTranscript(ctx context.Context, jobID string, transcript *interfaces.TranscriptResult) {
	// 1. Get Job Mappings (Overrides)
	mappings, _ := r.mappingRepo.ListByJob(ctx, jobID)
	mappingMap := make(map[string]string)
	for _, m := range mappings {
		mappingMap[m.OriginalSpeaker] = m.CustomName
	}

	// 2. Get Global Speakers (Cache)
	// For performance, we only fetch if we see global: IDs, but for now let's just fetch all
	// or fetch specifically what we need. Let's fetch all speakers for simplicity in this prototype.
	speakers, _ := r.speakerRepo.FindAll(ctx)
	speakerMap := make(map[string]string)
	for _, s := range speakers {
		speakerMap[s.ID] = s.Name
	}

	// 3. Resolve each segment
	for i := range transcript.Segments {
		segment := &transcript.Segments[i]
		if segment.Speaker != nil {
			originalID := *segment.Speaker
			resolvedName := r.ResolveSpeakerID(originalID, mappingMap, speakerMap)
			segment.Speaker = &resolvedName
			segment.SpeakerID = &originalID
		}
	}

	// 4. Resolve words
	for i := range transcript.WordSegments {
		word := &transcript.WordSegments[i]
		if word.Speaker != nil {
			originalID := *word.Speaker
			resolvedName := r.ResolveSpeakerID(originalID, mappingMap, speakerMap)
			word.Speaker = &resolvedName
			word.SpeakerID = &originalID
		}
	}
}

// ResolveSpeakerID performs the hierarchical lookup for a single speaker ID
func (r *SpeakerResolver) ResolveSpeakerID(id string, mappings map[string]string, globalSpeakers map[string]string) string {
	// Priority 1: Job-specific mapping (Original speaker ID -> Custom Name)
	if name, ok := mappings[id]; ok {
		return name
	}

	// Priority 2: Global identity registry
	if name, ok := globalSpeakers[id]; ok {
		return name
	}

	// Priority 3: Fallback (Clean up the ID for display if possible)
	if strings.HasPrefix(id, "global:") {
		// If we have a global ID but no name in our cache, we might have a sync issue,
		// or it's a new speaker that hasn't been named yet.
		return "Unknown Speaker"
	}

	if strings.HasPrefix(id, "local:") {
		// Transform local:speaker_00 -> Speaker 00
		localID := strings.TrimPrefix(id, "local:")
		return strings.Title(strings.ReplaceAll(localID, "_", " "))
	}

	return id
}
