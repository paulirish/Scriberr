import os
import subprocess
import json
import unittest

class TestSortformerDiarizeScript(unittest.TestCase):
    """
    Tests the sortformer_diarize.py script by running it with a test file
    and validating the generated JSON output.
    """
    def setUp(self):
        """Set up test environment."""
        self.output_file = "boobs.diar.json"
        # Ensure the output file doesn't exist before running the test
        if os.path.exists(self.output_file):
            os.remove(self.output_file)

    def tearDown(self):
        """Clean up after test."""
        # Clean up the created file
        if os.path.exists(self.output_file):
            os.remove(self.output_file)

    def test_script_produces_valid_json(self):
        """
        Run the script and validate its JSON output.
        """
        input_file = "data/whisperx-env/parakeet/test_files/mono-boobs.wav"
        command = [
            "uv", "run", "--native-tls", "--project", "data/whisperx-env/parakeet",
            "python", "data/whisperx-env/parakeet/sortformer_diarize.py",
            input_file, self.output_file
        ]

        # Run the script
        result = subprocess.run(command, capture_output=True, text=True)

        # Assert that the script ran successfully
        self.assertEqual(result.returncode, 0, f"Script failed with exit code {result.returncode}\nStderr: {result.stderr}")

        # Assert that the output file was created
        self.assertTrue(os.path.exists(self.output_file), f"Output file '{self.output_file}' was not created.")

        # Read and parse the JSON output
        with open(self.output_file, 'r') as f:
            try:
                diarization_result = json.load(f)
            except json.JSONDecodeError:
                self.fail(f"Output file '{self.output_file}' is not a valid JSON file.")

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