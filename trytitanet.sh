set -euxo pipefail

uv run --native-tls --project data/whisperx-env/parakeet/ python data/whisperx-env/parakeet/titanet_identify.py data/whisperx-env/parakeet/test_files/mono-boobs.wav data/whisperx-env/parakeet/test_files/boobs.input_segments.json boobs.output_segments.json --qdrant localhost --collection speakers --threshold 0.5


