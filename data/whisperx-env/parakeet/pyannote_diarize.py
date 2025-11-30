#!/usr/bin/env python3
"""
PyAnnote speaker diarization script.
Processes audio files to identify and separate different speakers.
"""

import argparse
import json
import sys
import os
from pathlib import Path
from pyannote.audio import Pipeline


def diarize_audio(
    audio_path: str,
    output_file: str,
    hf_token: str,
    model: str = "pyannote/speaker-diarization-3.1",
    min_speakers: int = None,
    max_speakers: int = None,
    output_format: str = "rttm",
    device: str = "cpu"
):
    """
    Perform speaker diarization on audio file using PyAnnote.
    """
    print(f"Loading PyAnnote speaker diarization pipeline: {model}")
    
    try:
        # Initialize the diarization pipeline
        pipeline = Pipeline.from_pretrained(
            model,
            use_auth_token=hf_token
        )
        
        # Move to specified device
        if device == "cuda":
            try:
                import torch
                if torch.cuda.is_available():
                    pipeline = pipeline.to(torch.device("cuda"))
                    print("Using CUDA for diarization")
                else:
                    print("CUDA not available, falling back to CPU")
            except ImportError:
                print("PyTorch not available for CUDA, using CPU")
        
        print("Pipeline loaded successfully")
    except Exception as e:
        print(f"Error loading pipeline: {e}")
        print("Make sure you have a valid Hugging Face token and have accepted the model's license")
        sys.exit(1)

    print(f"Processing audio file: {audio_path}")
    
    try:
        # Run diarization
        diarization_params = {}
        if min_speakers is not None:
            diarization_params["min_speakers"] = min_speakers
        if max_speakers is not None:
            diarization_params["max_speakers"] = max_speakers
            
        if diarization_params:
            print(f"Using speaker constraints: {diarization_params}")
            diarization = pipeline(audio_path, **diarization_params)
        else:
            print("Using automatic speaker detection")
            diarization = pipeline(audio_path)
        
        print(f"Diarization completed. Saving results to: {output_file}")
        
        if output_format == "rttm":
            # Save the diarization output to RTTM format
            with open(output_file, "w") as rttm:
                diarization.write_rttm(rttm)
        else:
            # Save as JSON format
            save_json_format(diarization, output_file, audio_path)
        
        # Print summary
        speakers = set()
        total_speech_time = 0.0
        
        for segment, track, speaker in diarization.itertracks(yield_label=True):
            speakers.add(speaker)
            total_speech_time += segment.duration
        
        print(f"\nDiarization Summary:")
        print(f"  Speakers detected: {len(speakers)}")
        print(f"  Speaker labels: {sorted(speakers)}")
        print(f"  Total speech time: {total_speech_time:.2f} seconds")
        print(f"  Output file saved: {output_file}")
        
    except Exception as e:
        print(f"Error during diarization: {e}")
        sys.exit(1)


def save_json_format(diarization, output_file: str, audio_path: str):
    """Save diarization results in JSON format."""
    segments = []
    speakers = set()
    
    for segment, track, speaker in diarization.itertracks(yield_label=True):
        segments.append({
            "start": segment.start,
            "end": segment.end,
            "speaker": speaker,
            "confidence": 1.0,  # PyAnnote doesn't provide per-segment confidence
            "duration": segment.duration
        })
        speakers.add(speaker)
    
    # Sort segments by start time
    segments.sort(key=lambda x: x["start"])
    
    results = {
        "audio_file": audio_path,
        "model": "pyannote/speaker-diarization-3.1",
        "segments": segments,
        "speakers": sorted(speakers),
        "speaker_count": len(speakers),
        "total_duration": max(seg["end"] for seg in segments) if segments else 0,
        "processing_info": {
            "total_segments": len(segments),
            "total_speech_time": sum(seg["duration"] for seg in segments)
        }
    }
    
    with open(output_file, "w") as f:
        json.dump(results, f, indent=2)


def main():
    parser = argparse.ArgumentParser(
        description="Perform speaker diarization using PyAnnote.audio"
    )
    parser.add_argument(
        "audio_file",
        help="Path to audio file"
    )
    parser.add_argument(
        "--output", "-o",
        required=True,
        help="Output file path"
    )
    parser.add_argument(
        "--hf-token",
        required=True,
        help="Hugging Face access token"
    )
    parser.add_argument(
        "--model",
        default="pyannote/speaker-diarization-3.1",
        help="PyAnnote model to use"
    )
    parser.add_argument(
        "--min-speakers",
        type=int,
        help="Minimum number of speakers"
    )
    parser.add_argument(
        "--max-speakers",
        type=int,
        help="Maximum number of speakers"
    )
    parser.add_argument(
        "--output-format",
        choices=["rttm", "json"],
        default="rttm",
        help="Output format"
    )
    parser.add_argument(
        "--device",
        choices=["cpu", "cuda"],
        default="cpu",
        help="Device to use for computation"
    )

    args = parser.parse_args()

    # Validate input file
    if not os.path.exists(args.audio_file):
        print(f"Error: Audio file not found: {args.audio_file}")
        sys.exit(1)

    # Validate speaker constraints
    if args.min_speakers is not None and args.min_speakers < 1:
        print("Error: min_speakers must be at least 1")
        sys.exit(1)
        
    if args.max_speakers is not None and args.max_speakers < 1:
        print("Error: max_speakers must be at least 1")
        sys.exit(1)
        
    if (args.min_speakers is not None and args.max_speakers is not None and 
        args.min_speakers > args.max_speakers):
        print("Error: min_speakers cannot be greater than max_speakers")
        sys.exit(1)

    # Create output directory if it doesn't exist
    output_path = Path(args.output)
    output_path.parent.mkdir(parents=True, exist_ok=True)

    try:
        diarize_audio(
            audio_path=args.audio_file,
            output_file=args.output,
            hf_token=args.hf_token,
            model=args.model,
            min_speakers=args.min_speakers,
            max_speakers=args.max_speakers,
            output_format=args.output_format,
            device=args.device
        )
    except Exception as e:
        print(f"Error during diarization: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
