import unittest
import sys
import os
import shutil
import json
from pathlib import Path
import sortformer_diarize

class TestSortformerIntegration(unittest.TestCase):

    @classmethod
    def setUpClass(cls):
        # Symlink model from /tmp to current directory for the script to find it
        # The script looks for model in the same directory as itself
        script_dir = os.path.dirname(os.path.abspath(sortformer_diarize.__file__))
        cls.model_path = os.path.join(script_dir, "diar_streaming_sortformer_4spk-v2.nemo")
        cls.tmp_model_path = "/tmp/diar_streaming_sortformer_4spk-v2.nemo"

        if not os.path.exists(cls.model_path) and os.path.exists(cls.tmp_model_path):
            os.symlink(cls.tmp_model_path, cls.model_path)

        cls.test_files = [
            "test_files/an255-fash-b.wav",
            "test_files/cen7-fash-b.wav"
        ]

    @classmethod
    def tearDownClass(cls):
        # Remove symlink
        if os.path.islink(cls.model_path):
            os.remove(cls.model_path)

    def test_run_diarization(self):
        # Skip if model not found
        if not os.path.exists(self.model_path):
            self.skipTest("Model file not found")

        for audio_file in self.test_files:
            if not os.path.exists(audio_file):
                print(f"Skipping {audio_file}, not found")
                continue

            output_file = f"{audio_file}.json"

            print(f"Running diarization on {audio_file}")
            try:
                # We call the script's main function via command line arguments simulation
                # or directly call diarize_audio
                sortformer_diarize.diarize_audio(
                    audio_path=audio_file,
                    output_file=output_file,
                    batch_size=1,
                    device="cpu",
                    output_format="json"
                )

                # Verify output exists and is valid JSON
                self.assertTrue(os.path.exists(output_file))
                with open(output_file, 'r') as f:
                    data = json.load(f)
                    self.assertIn("segments", data)
                    print(f"Found {len(data['segments'])} segments in {audio_file}")

            except Exception as e:
                self.fail(f"Diarization failed: {e}")

if __name__ == "__main__":
    unittest.main()
