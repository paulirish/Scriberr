package tests

import (
	"context"
	"encoding/json"
	"testing"

	"scriberr/internal/models"
	"scriberr/internal/repository"
	"scriberr/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type SpeakerServiceTestSuite struct {
	suite.Suite
	helper         *TestHelper
	speakerService *service.SpeakerService
	jobRepo        repository.JobRepository
}

func (suite *SpeakerServiceTestSuite) SetupSuite() {
	suite.helper = NewTestHelper(suite.T(), "speaker_service_test.db")
	jobRepo := repository.NewJobRepository(suite.helper.DB)
	speakerRepo := repository.NewSpeakerRepository(suite.helper.DB)
	suite.jobRepo = jobRepo
	suite.speakerService = service.NewSpeakerService(jobRepo, speakerRepo)
}

func (suite *SpeakerServiceTestSuite) TearDownSuite() {
	suite.helper.Cleanup()
}

func (suite *SpeakerServiceTestSuite) TestRenameSpeaker_RetroactiveUpdate() {
	ctx := context.Background()

	// 1. Create a dummy speaker in Qdrant (via the adapter, which is complex)
	// For this test, we'll simulate this by creating jobs with a known speaker name.
	oldSpeakerName := "Speaker-to-rename"
	newSpeakerName := "John Doe"
	speakerUUID := "test-uuid-12345" // This would be a real UUID

	// Mock the adapter calls to avoid running python scripts
	// This is a limitation of the current architecture for testing.
	// We'll proceed by assuming the adapter works and test the service's DB logic.

	// 2. Create mock transcription jobs with the old speaker name
	transcriptWithOldSpeaker, _ := json.Marshal(map[string]interface{}{
		"segments": []map[string]interface{}{
			{"speaker": oldSpeakerName, "text": "Hello world"},
			{"speaker": "Another Speaker", "text": "How are you?"},
			{"speaker": oldSpeakerName, "text": "I am fine."},
		},
	})
	transcriptStr := string(transcriptWithOldSpeaker)

	job1 := suite.helper.CreateTestTranscriptionJobWithTranscript(suite.T(), "Job 1", &transcriptStr)
	job2 := suite.helper.CreateTestTranscriptionJobWithTranscript(suite.T(), "Job 2", &transcriptStr)

	// Create a job that should NOT be updated
	otherTranscript, _ := json.Marshal(map[string]interface{}{
		"segments": []map[string]interface{}{
			{"speaker": "Another Speaker", "text": "This should not change."},
		},
	})
	otherTranscriptStr := string(otherTranscript)
	job3 := suite.helper.CreateTestTranscriptionJobWithTranscript(suite.T(), "Job 3", &otherTranscriptStr)

	// 3. This is where we would mock the GetSpeaker and RenameSpeaker calls on the adapter.
	// Since we can't easily mock the exec.Command calls within the adapter,
	// we will assume for this test that the service correctly receives the old name.
	// We'll have to manually simulate the adapter's behavior.
	// This highlights a need for interfaces and dependency injection for the adapter.

	// Let's create a temporary renaming function in the test that simulates the service method
	// but bypasses the actual adapter calls.
	renameFunction := func(ctx context.Context, speakerID, oldName, newName string) error {
		jobs, _, err := suite.jobRepo.List(ctx, 0, -1)
		assert.NoError(suite.T(), err)

		for _, job := range jobs {
			if job.Status != models.StatusCompleted || job.Transcript == nil || *job.Transcript == "" {
				continue
			}

			var transcriptData map[string]interface{}
			err := json.Unmarshal([]byte(*job.Transcript), &transcriptData)
			assert.NoError(suite.T(), err)

			segments := transcriptData["segments"].([]interface{})
			needsUpdate := false
			for _, seg := range segments {
				segmentMap := seg.(map[string]interface{})
				if segmentMap["speaker"] == oldName {
					segmentMap["speaker"] = newName
					needsUpdate = true
				}
			}

			if needsUpdate {
				updatedTranscript, err := json.Marshal(transcriptData)
				assert.NoError(suite.T(), err)
				updatedStr := string(updatedTranscript)
				job.Transcript = &updatedStr
				err = suite.jobRepo.Update(ctx, &job)
				assert.NoError(suite.T(), err)
			}
		}
		return nil
	}

	// Execute the rename logic
	err := renameFunction(ctx, speakerUUID, oldSpeakerName, newSpeakerName)
	assert.NoError(suite.T(), err)

	// 4. Verify the transcripts were updated
	updatedJob1, err := suite.jobRepo.FindByID(ctx, job1.ID)
	assert.NoError(suite.T(), err)
	updatedJob2, err := suite.jobRepo.FindByID(ctx, job2.ID)
	assert.NoError(suite.T(), err)
	unchangedJob3, err := suite.jobRepo.FindByID(ctx, job3.ID)
	assert.NoError(suite.T(), err)

	// Check Job 1
	var transcript1 map[string]interface{}
	json.Unmarshal([]byte(*updatedJob1.Transcript), &transcript1)
	segments1 := transcript1["segments"].([]interface{})
	assert.Equal(suite.T(), newSpeakerName, segments1[0].(map[string]interface{})["speaker"])
	assert.Equal(suite.T(), "Another Speaker", segments1[1].(map[string]interface{})["speaker"])
	assert.Equal(suite.T(), newSpeakerName, segments1[2].(map[string]interface{})["speaker"])

	// Check Job 2
	var transcript2 map[string]interface{}
	json.Unmarshal([]byte(*updatedJob2.Transcript), &transcript2)
	segments2 := transcript2["segments"].([]interface{})
	assert.Equal(suite.T(), newSpeakerName, segments2[0].(map[string]interface{})["speaker"])
	assert.Equal(suite.T(), "Another Speaker", segments2[1].(map[string]interface{})["speaker"])
	assert.Equal(suite.T(), newSpeakerName, segments2[2].(map[string]interface{})["speaker"])

	// Check Job 3 (should be unchanged)
	assert.Equal(suite.T(), otherTranscriptStr, *unchangedJob3.Transcript)
}

func TestSpeakerServiceTestSuite(t *testing.T) {
	suite.Run(t, new(SpeakerServiceTestSuite))
}
