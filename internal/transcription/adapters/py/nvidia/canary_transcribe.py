#!/usr/bin/env python3
"""
NVIDIA Canary multilingual transcription and translation script.
"""

import argparse
import json
import sys
import os
from pathlib import Path
import torch
import torch.profiler
import nemo.collections.asr as nemo_asr


def transcribe_audio(
    audio_path: str,
    source_lang: str = "en",
    target_lang: str = "en",
    task: str = "transcribe",
    timestamps: bool = True,
    output_file: str = None,
    include_confidence: bool = True,
    preserve_formatting: bool = True,
    profile: bool = False,
    profile_output: str = None,
):
    """
    Transcribe or translate audio using NVIDIA Canary model.
    """
    # Determine model path
    model_filename = "canary-1b-v2.nemo"
    model_path = None

    # Locate project root: derived from VIRTUAL_ENV, which is set by `uv run` to path/.venv
    virtual_env = os.environ.get("VIRTUAL_ENV")
    if not virtual_env:
        print("Error: VIRTUAL_ENV environment variable not set. Script must be run with 'uv run'.")
        sys.exit(1)

    project_root = os.path.dirname(virtual_env)
    model_path = os.path.join(project_root, model_filename)

    if not os.path.exists(model_path):
        print(f"Error during transcription: Can't find {model_filename} in project root: {project_root}")
        sys.exit(1)

    # Device selection
    device_type = "cpu"
    if torch.cuda.is_available():
        device_type = "cuda"
    elif torch.backends.mps.is_available():
        device_type = "mps"
        # Not all operations are implemented on MPS yet, enable fallback to CPU
        os.environ["PYTORCH_ENABLE_MPS_FALLBACK"] = "1"

    device = torch.device(device_type)
    print(f"Using device: {device_type.upper()}")

    print(f"Loading NVIDIA Canary model from: {model_path}")

    # map_location handles loading tensors to the correct device initially
    asr_model = nemo_asr.models.ASRModel.restore_from(model_path, map_location=device)

    # Explicitly move the model to the device to ensure all submodules are correct
    asr_model = asr_model.to(device)

    # Optimize for inference
    asr_model.eval()

    print(f"Processing: {audio_path}")
    print(f"Task: {task}")
    print(f"Source language: {source_lang}")
    print(f"Target language: {target_lang}")

    # Set up profiling context
    if profile:
        activities = [torch.profiler.ProfilerActivity.CPU]
        if torch.cuda.is_available():
            activities.append(torch.profiler.ProfilerActivity.CUDA)

        profiler = torch.profiler.profile(
            activities=activities,
            record_shapes=True,
            with_stack=True,
            on_trace_ready=None  # We will manually export
        )
    else:
        # Dummy context manager if profiling is disabled
        from contextlib import nullcontext
        profiler = nullcontext()

    with profiler as prof:
        if timestamps:
            if task == "translate" and source_lang != target_lang:
                # Translation with timestamps
                output = asr_model.transcribe(
                    [audio_path],
                    source_lang=source_lang,
                    target_lang=target_lang,
                    timestamps=True
                )
            else:
                # Transcription with timestamps
                output = asr_model.transcribe(
                    [audio_path],
                    source_lang=source_lang,
                    target_lang=target_lang,
                    timestamps=True
                )

            # Extract text and timestamps
            result_data = output[0]
            text = result_data.text
            word_timestamps = result_data.timestamp.get("word", [])
            segment_timestamps = result_data.timestamp.get("segment", [])

            print(f"Result: {text}")

            # Prepare output data
            output_data = {
                "transcription": text,
                "source_language": source_lang,
                "target_language": target_lang,
                "task": task,
                "word_timestamps": word_timestamps,
                "segment_timestamps": segment_timestamps,
                "audio_file": audio_path,
                "model": "canary-1b-v2"
            }

            if include_confidence:
                # Add confidence scores if available
                if hasattr(result_data, 'confidence') and result_data.confidence:
                    output_data["confidence"] = result_data.confidence

            # Save to file
            if output_file:
                with open(output_file, 'w', encoding='utf-8') as f:
                    json.dump(output_data, f, indent=2, ensure_ascii=False)
                print(f"Results saved to: {output_file}")
            else:
                print(json.dumps(output_data, indent=2, ensure_ascii=False))

        else:
            # Simple transcription/translation without timestamps
            if task == "translate" and source_lang != target_lang:
                output = asr_model.transcribe(
                    [audio_path],
                    source_lang=source_lang,
                    target_lang=target_lang
                )
            else:
                output = asr_model.transcribe(
                    [audio_path],
                    source_lang=source_lang,
                    target_lang=target_lang
                )

            text = output[0].text

            output_data = {
                "transcription": text,
                "source_language": source_lang,
                "target_language": target_lang,
                "task": task,
                "audio_file": audio_path,
                "model": "canary-1b-v2"
            }

            if output_file:
                with open(output_file, 'w', encoding='utf-8') as f:
                    json.dump(output_data, f, indent=2, ensure_ascii=False)
                print(f"Results saved to: {output_file}")
            else:
                print(json.dumps(output_data, indent=2, ensure_ascii=False))

    if profile:
        trace_path = "trace.json"
        if profile_output:
            if os.path.isdir(profile_output):
                trace_path = os.path.join(profile_output, "trace.json")
            else:
                trace_path = profile_output

        prof.export_chrome_trace(trace_path)
        print(f"Profiling trace saved to: {trace_path}")


def main():
    parser = argparse.ArgumentParser(
        description="Transcribe or translate audio using NVIDIA Canary model"
    )
    parser.add_argument("audio_file", help="Path to audio file")
    parser.add_argument(
        "--source-lang", default="en",
        choices=["en", "de", "es", "fr", "hi", "it", "ja", "ko", "pl", "pt", "ru", "zh"],
        help="Source language (default: en)"
    )
    parser.add_argument(
        "--target-lang", default="en",
        choices=["en", "de", "es", "fr", "hi", "it", "ja", "ko", "pl", "pt", "ru", "zh"],
        help="Target language (default: en)"
    )
    parser.add_argument(
        "--task", choices=["transcribe", "translate"], default="transcribe",
        help="Task to perform (default: transcribe)"
    )
    parser.add_argument(
        "--timestamps", action="store_true", default=True,
        help="Include word and segment level timestamps"
    )
    parser.add_argument(
        "--no-timestamps", dest="timestamps", action="store_false",
        help="Disable timestamps"
    )
    parser.add_argument(
        "--output", "-o", help="Output file path"
    )
    parser.add_argument(
        "--include-confidence", action="store_true", default=True,
        help="Include confidence scores"
    )
    parser.add_argument(
        "--no-confidence", dest="include_confidence", action="store_false",
        help="Exclude confidence scores"
    )
    parser.add_argument(
        "--preserve-formatting", action="store_true", default=True,
        help="Preserve punctuation and capitalization"
    )
    parser.add_argument(
        "--profile", action="store_true",
        help="Enable PyTorch profiling (saves Chrome trace)"
    )
    parser.add_argument(
        "--profile-output", default=None,
        help="Output path for profiling trace (default: trace.json)"
    )

    args = parser.parse_args()

    # Validate input file
    if not os.path.exists(args.audio_file):
        print(f"Error: Audio file not found: {args.audio_file}")
        sys.exit(1)

    try:
        transcribe_audio(
            audio_path=args.audio_file,
            source_lang=args.source_lang,
            target_lang=args.target_lang,
            task=args.task,
            timestamps=args.timestamps,
            output_file=args.output,
            include_confidence=args.include_confidence,
            preserve_formatting=args.preserve_formatting,
            profile=args.profile,
            profile_output=args.profile_output,
        )
    except Exception as e:
        print(f"Error during transcription: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
