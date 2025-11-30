import os
import subprocess
import json
import unittest

class TestParakeetScripts(unittest.TestCase):
    """
    Tests the parakeet diarization and transcription scripts by running them
    with test files and validating the generated JSON output.
    """
    def setUp(self):
        """Set up test environment."""
        self.diarize_output_file = "boobs.diar.json"
        self.transcribe_output_file = "boobs.parakeet.json"

        # Ensure output files don't exist before running the test
        if os.path.exists(self.diarize_output_file):
            os.remove(self.diarize_output_file)
        if os.path.exists(self.transcribe_output_file):
            os.remove(self.transcribe_output_file)

    def tearDown(self):
        """Clean up after test."""
        # Clean up the created files
        if os.path.exists(self.diarize_output_file):
            os.remove(self.diarize_output_file)
        if os.path.exists(self.transcribe_output_file):
            os.remove(self.transcribe_output_file)



    def test_transcribe_script_produces_valid_json(self):
        """
        Run the transcribe.py script and validate its JSON output.
        """
        input_file = "data/whisperx-env/parakeet/test_files/mono-boobs.wav"
        command = [
            "uv", "run", "--native-tls", "--project", "data/whisperx-env/parakeet",
            "python", "data/whisperx-env/parakeet/transcribe.py",
            input_file,
            "--output", self.transcribe_output_file,
            "--timestamps",
            "--context-left", "256",
            "--context-right", "256"
        ]

        # Run the script
        result = subprocess.run(command, capture_output=True, text=True)

        # Assert that the script ran successfully
        self.assertEqual(result.returncode, 0, f"Script failed with exit code {result.returncode}\nStderr: {result.stderr}")

        # Assert that the output file was created
        self.assertTrue(os.path.exists(self.transcribe_output_file), f"Output file '{self.transcribe_output_file}' was not created.")

        # Read and parse the JSON output
        with open(self.transcribe_output_file, 'r') as f:
            try:
                transcription_result = json.load(f)
            except json.JSONDecodeError:
                self.fail(f"Output file '{self.transcribe_output_file}' is not a valid JSON file.")

        # Perform assertions on the JSON content
        self.assertIsInstance(transcription_result, dict, "The top-level JSON element should be a dictionary.")

        self.assertIn("transcription", transcription_result, "The JSON output should contain a 'transcription' key.")
        self.assertIsInstance(transcription_result["transcription"], str, "'transcription' should be a string.")
        self.assertGreater(len(transcription_result["transcription"]), 0, "'transcription' should not be empty.")

        self.assertIn("language", transcription_result, "The JSON output should contain a 'language' key.")
        self.assertEqual(transcription_result["language"], "en", "'language' should be 'en'.")

        self.assertIn("word_timestamps", transcription_result, "The JSON output should contain a 'word_timestamps' key.")
        word_timestamps = transcription_result["word_timestamps"]
        self.assertIsInstance(word_timestamps, list, "'word_timestamps' should be a list.")
        self.assertGreater(len(word_timestamps), 0, "'word_timestamps' should not be empty.")

        if len(word_timestamps) > 0:
            first_word_segment = word_timestamps[0]
            self.assertIn("word", first_word_segment, "Each word segment should have a 'word' key.")
            self.assertIn("start_offset", first_word_segment, "Each word segment should have a 'start_offset' key.")
            self.assertIn("end_offset", first_word_segment, "Each word segment should have a 'end_offset' key.")
            self.assertIn("start", first_word_segment, "Each word segment should have a 'start' key.")
            self.assertIn("end", first_word_segment, "Each word segment should have a 'end' key.")

            self.assertIsInstance(first_word_segment["word"], str, "'word' should be a string.")
            self.assertIsInstance(first_word_segment["start_offset"], int, "'start_offset' should be an integer.")
            self.assertIsInstance(first_word_segment["end_offset"], int, "'end_offset' should be an integer.")
            self.assertIsInstance(first_word_segment["start"], (int, float), "'start' should be a number.")
            self.assertIsInstance(first_word_segment["end"], (int, float), "'end' should be a number.")

        self.assertIn("segment_timestamps", transcription_result, "The JSON output should contain a 'segment_timestamps' key.")
        segment_timestamps = transcription_result["segment_timestamps"]
        self.assertIsInstance(segment_timestamps, list, "'segment_timestamps' should be a list.")
        self.assertGreater(len(segment_timestamps), 0, "'segment_timestamps' should not be empty.")

        if len(segment_timestamps) > 0:
            first_segment_ts = segment_timestamps[0]
            self.assertIn("segment", first_segment_ts, "Each segment timestamp should have a 'segment' key.")
            self.assertIn("start_offset", first_segment_ts, "Each segment timestamp should have a 'start_offset' key.")
            self.assertIn("end_offset", first_segment_ts, "Each segment timestamp should have a 'end_offset' key.")
            self.assertIn("start", first_segment_ts, "Each segment timestamp should have a 'start' key.")
            self.assertIn("end", first_segment_ts, "Each segment timestamp should have a 'end' key.")

            self.assertIsInstance(first_segment_ts["segment"], str, "'segment' should be a string.")
            self.assertIsInstance(first_segment_ts["start_offset"], int, "'start_offset' should be an integer.")
            self.assertIsInstance(first_segment_ts["end_offset"], int, "'end_offset' should be an integer.")
            self.assertIsInstance(first_segment_ts["start"], (int, float), "'start' should be a number.")
            self.assertIsInstance(first_segment_ts["end"], (int, float), "'end' should be a number.")

        self.assertIn("audio_file", transcription_result, "The JSON output should contain an 'audio_file' key.")
        self.assertIsInstance(transcription_result["audio_file"], str, "'audio_file' should be a string.")
        self.assertEqual(transcription_result["audio_file"], input_file, "'audio_file' should match the input file.")

        self.assertIn("model", transcription_result, "The JSON output should contain a 'model' key.")
        self.assertIsInstance(transcription_result["model"], str, "'model' should be a string.")
        self.assertGreater(len(transcription_result["model"]), 0, "'model' should not be empty.")

        self.assertIn("context", transcription_result, "The JSON output should contain a 'context' key.")
        self.assertIsInstance(transcription_result["context"], dict, "'context' should be a dictionary.")
        self.assertIn("left", transcription_result["context"], "'context' should contain a 'left' key.")
        self.assertEqual(transcription_result["context"]["left"], 256, "'context.left' should be 256.")
        self.assertIn("right", transcription_result["context"], "'context' should contain a 'right' key.")
        self.assertEqual(transcription_result["context"]["right"], 256, "'context.right' should be 256.")



    def test_sortformer_diarize_script_produces_valid_json(self):
        """
        Run the sortformer_diarize.py script and validate its JSON output.
        """
        input_file = "data/whisperx-env/parakeet/test_files/mono-boobs.wav"
        command = [
            "uv", "run", "--native-tls", "--project", "data/whisperx-env/parakeet",
            "python", "data/whisperx-env/parakeet/sortformer_diarize.py",
            input_file, self.diarize_output_file
        ]

        # Run the script
        result = subprocess.run(command, capture_output=True, text=True)

        # Assert that the script ran successfully
        self.assertEqual(result.returncode, 0, f"Script failed with exit code {result.returncode}\nStderr: {result.stderr}")

        # Assert that the output file was created
        self.assertTrue(os.path.exists(self.diarize_output_file), f"Output file '{self.diarize_output_file}' was not created.")

        # Read and parse the JSON output
        with open(self.diarize_output_file, 'r') as f:
            try:
                diarization_result = json.load(f)
            except json.JSONDecodeError:
                self.fail(f"Output file '{self.diarize_output_file}' is not a valid JSON file.")

        # Perform assertions on the JSON content
        self.assertIsInstance(diarization_result, dict, "The top-level JSON element should be a dictionary.")
        self.assertIn("segments", diarization_result, "The JSON output should contain a 'segments' key.")

        segments = diarization_result["segments"]
        self.assertIsInstance(segments, list, "The 'segments' value should be a list.")
        self.assertGreater(len(segments), 0, "The diarization result should not be empty.")

        # Check the structure of the first element
        if len(segments) > 0:
            first_segment = segments[0]
            self.assertIn("start", first_segment, "Each segment should have a 'start' key.")
            self.assertIn("end", first_segment, "Each segment should have a 'end' key.")
            self.assertIn("speaker", first_segment, "Each segment should have a 'speaker' key.")

            self.assertIsInstance(first_segment["start"], (int, float), "'start' should be a number.")
            self.assertIsInstance(first_segment["end"], (int, float), "'end' should be a number.")
            self.assertIsInstance(first_segment["speaker"], str, "'speaker' should be a string.")

        # Assertions for speaker_count, total_segments, and total_duration
        self.assertIn("speaker_count", diarization_result, "The JSON output should contain a 'speaker_count' key.")
        self.assertEqual(diarization_result["speaker_count"], 2, "'speaker_count' should be 2.")

        self.assertIn("total_segments", diarization_result, "The JSON output should contain a 'total_segments' key.")
        self.assertGreater(diarization_result["total_segments"], 7, "'total_segments' should be greater than 7.")

        self.assertIn("total_duration", diarization_result, "The JSON output should contain a 'total_duration' key.")
        self.assertGreater(diarization_result["total_duration"], 22, "'total_duration' should be greater than 22.")

if __name__ == '__main__':
    unittest.main()
