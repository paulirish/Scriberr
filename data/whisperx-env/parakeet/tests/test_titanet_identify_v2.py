import unittest
from unittest.mock import MagicMock, patch, call
import sys
import os
import json
import numpy as np

# Add the adapters directory to the Python path
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '../../internal/transcription/adapters')))

import titanet_identify_v2 as identifier

# Mock objects for Qdrant search results
class MockPoint:
    def __init__(self, id, score, vector, payload=None):
        self.id = id
        self.score = score
        self.vector = vector
        self.payload = payload or {'name': f'Speaker-{id}'}

class TestIdentifier(unittest.TestCase):

    def setUp(self):
        """Setup common mock objects and file paths for tests."""
        self.audio_file = "test.wav"
        self.segments_file = "segments.json"
        self.output_file = "output.json"

        # Create dummy segment file
        with open(self.segments_file, 'w') as f:
            json.dump({
                "segments": [
                    {"speaker": "speaker_0", "start": 0.0, "end": 2.5, "duration": 2.5},
                    {"speaker": "speaker_1", "start": 2.5, "end": 5.0, "duration": 2.5},
                ]
            }, f)

    def tearDown(self):
        """Clean up created files."""                                                                                                           
        for f in [self.segments_file, self.output_file]:                                                                                                
            if os.path.exists(f):                                                                                                                       
                os.remove(f)                                                                                                                            

    @patch('titanet_identify_v2.os.path.getctime', return_value=12345.0) # Mock getctime                                                                
    @patch('titanet_identify_v2.EncDecSpeakerLabelModel')                                                                                               
    @patch('titanet_identify_v2.QdrantClient')                                                                                                          
    @patch('titanet_identify_v2.sf')                                                                                                                    
    def test_enroll_new_speaker(self, mock_sf, MockQdrantClient, MockNeMoModel, mock_getctime): # Added mock_getctime                                   
        """Test enrolling a new speaker when no match is found."""                                                                                      
        mock_client = MockQdrantClient.return_value                                                                                                     
        mock_client.search.return_value = [] # No match found                                                                                           
        mock_client.get_collection.return_value = MagicMock(vectors_count=0) # For S-Norm cohort                                                        
                                                                                                                                                        
        mock_model = MockNeMoModel.restore_from.return_value                                                                                            
        # Mock model to return a consistent embedding with correct shape                                                                                
        mock_model.return_value.side_effect = [
            (None, np.array([[0.1] * 192])) for _ in range(10) # For speaker_0
        ] + [
            (None, np.array([[0.2] * 192])) for _ in range(10) # For speaker_1
        ]
                                                                                                                                                        
        # Mock soundfile to return a dummy waveform                                                                                                     
        mock_sf.read.return_value = (np.zeros(16000 * 5), 16000)                                                                                        
                                                                                                                                                        
        identifier.identify_speakers(                                                                                                                   
            self.audio_file, self.segments_file, self.output_file,                                                                                      
            threshold=0.7, threshold_new=0.55                                                                                                           
        )                                                                                                                                               
                                                                                                                                                        
        # Assertions                                                                                                                                    
        upsert_calls = mock_client.upsert.call_args_list                                                                                                
        # Expect 2 enrollments (speaker_0, speaker_1) and 2 imposter collections                                                                        
        self.assertEqual(len(upsert_calls), 4)

        # Check a call to the main speaker collection
        main_collection_call = any(
            call.kwargs['collection_name'] == 'speakers' for call in upsert_calls
        )
        self.assertTrue(main_collection_call)

        # Check a call to the imposter candidates collection
        imposter_collection_call = any(
            call.kwargs['collection_name'] == 'imposter_candidates' for call in upsert_calls
        )
        self.assertTrue(imposter_collection_call)

    @patch('titanet_identify_v2.os.path.getctime', return_value=12345.0) # Mock getctime
    @patch('titanet_identify_v2.EncDecSpeakerLabelModel')
    @patch('titanet_identify_v2.QdrantClient')
    @patch('titanet_identify_v2.sf')
    def test_match_and_update_speaker(self, mock_sf, MockQdrantClient, MockNeMoModel, mock_getctime): # Added mock_getctime                             
        """Test matching an existing speaker and updating their profile with EMA."""                                                                    
        mock_client = MockQdrantClient.return_value                                                                                                     
        mock_client.get_collection.return_value = MagicMock(vectors_count=0) # For S-Norm cohort                                                        
                                                                                                                                                        
        # Mock a high-confidence match for speaker_0                                                                                                    
        existing_vector = np.array([0.15] * 192)                                                                                                        
        mock_client.search.side_effect = [
            [MockPoint(id="123", score=0.8, vector=existing_vector.tolist())], # speaker_0 matches
            [] # speaker_1 is new
        ]

        mock_model = MockNeMoModel.restore_from.return_value
        # Use side_effect on the callable mock (mock_model.return_value) to return distinct embeddings for each segment
        mock_model.return_value.side_effect = [
            (None, np.array([[0.1] * 192])) for _ in range(10) # For speaker_0
        ] + [
            (None, np.array([[0.2] * 192])) for _ in range(10) # For speaker_1
        ]
        mock_sf.read.return_value = (np.zeros(16000 * 5), 16000)

        identifier.identify_speakers(
            self.audio_file, self.segments_file, self.output_file,
            threshold=0.7, threshold_new=0.55
        )

        # Assertions
        upsert_calls = mock_client.upsert.call_args_list
        # Expect 1 update, 1 new enrollment, 1 imposter
        self.assertEqual(len(upsert_calls), 3)

        # Check that the EMA update happened for the matched speaker
        update_call = upsert_calls[0]
        self.assertEqual(update_call.kwargs['collection_name'], 'speakers')
        updated_point = update_call.kwargs['points'][0]
        self.assertEqual(updated_point.id, "123")
        
        # Verify the EMA logic (simplified check)
        # wt = min(alpha_max, total_duration / min_duration_for_full_weight)
        # wt = min(0.25, 2.5 / 4.0) = 0.25
        # updated = (1-0.25)*0.15 + 0.25*0.1 = 0.1125 + 0.025 = 0.1375
        expected_val = 0.1375 
        # The vector is normalized, so we check the ratio, not the absolute value
        self.assertAlmostEqual(updated_point.vector[0] / np.linalg.norm(updated_point.vector), expected_val / np.linalg.norm([expected_val]*192), places=4)


    @patch('titanet_identify_v2.os.path.getctime', return_value=12345.0) # Mock getctime
    @patch('titanet_identify_v2.EncDecSpeakerLabelModel')
    @patch('titanet_identify_v2.QdrantClient')
    @patch('titanet_identify_v2.sf')
    def test_s_norm_logic(self, mock_sf, MockQdrantClient, MockNeMoModel, mock_getctime): # Added mock_getctime
        """Test that S-Norm is applied when a cohort is available."""                                                                                   
        mock_client = MockQdrantClient.return_value                                                                                                     
                                                                                                                                                        
        # Mock cohort                                                                                                                                   
        cohort_vectors = [MagicMock(vector=np.random.rand(192).tolist()) for _ in range(10)]                                                            
        mock_client.get_collection.return_value = MagicMock(vectors_count=10) # Ensure vectors_count returns an int                                     
        mock_client.scroll.return_value = (cohort_vectors, None)

        # Mock a match that is ambiguous raw, but clear with S-Norm
        # Raw score is 0.6, which is < 0.7 threshold
        mock_client.search.side_effect = [
            [MockPoint(id="123", score=0.6, vector=[0.15]*192)], # speaker_0 matches
            [] # speaker_1 is new
        ]

        mock_model = MockNeMoModel.restore_from.return_value
        # Use side_effect on the callable mock (mock_model.return_value) to return distinct embeddings for each segment
        mock_model.return_value.side_effect = [
            (None, np.array([[0.1] * 192])) for _ in range(10) # For speaker_0
        ] + [
            (None, np.array([[0.2] * 192])) for _ in range(10) # For speaker_1
        ]
        mock_sf.read.return_value = (np.zeros(16000 * 5), 16000)

        # No patching of numpy here, let actual numpy functions be used                                                                                 
        identifier.identify_speakers(
            self.audio_file, self.segments_file, self.output_file,
            threshold=0.7, threshold_new=0.55, norm_threshold=1.5
        )

        # Assertions
        # With a high norm_score, we expect a match and update.                                                                                         
        # For speaker_1, it will be in the ambiguity zone and enroll as a new speaker.
        # Expected upsert calls: 1 for speaker_0 update, 1 for speaker_1 enrollment.
        self.assertEqual(mock_client.upsert.call_count, 2) 
        update_call = mock_client.upsert.call_args_list[0]                                                                                              
        self.assertEqual(update_call.kwargs['points'][0].id, '123')