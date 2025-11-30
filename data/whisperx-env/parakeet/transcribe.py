#!/usr/bin/env python3
"""
NVIDIA Parakeet transcription script with timestamp support.
"""

import argparse
import json
import sys
import os
from pathlib import Path
import nemo.collections.asr as nemo_asr


def transcribe_audio(
    audio_path: str,
    timestamps: bool = True,
    output_file: str = None,
    context_left: int = 256,
    context_right: int = 256,
    include_confidence: bool = True,
):
    """
    Transcribe audio using NVIDIA Parakeet model.
    """
    # Get the directory where this script is located
    script_dir = os.path.dirname(os.path.abspath(__file__))
    model_path = os.path.join(script_dir, "parakeet-tdt-0.6b-v3.nemo")
    
    print(f"Script directory: {script_dir}")
    print(f"Looking for model at: {model_path}")
    
    if not os.path.exists(model_path):
        print(f"Error during transcription: Can't find {model_path}")
        # List files in the directory to help debug
        try:
            files = os.listdir(script_dir)
            print(f"Files in {script_dir}: {files}")
        except Exception as e:
            print(f"Could not list directory: {e}")
        sys.exit(1)
    
    print(f"Loading NVIDIA Parakeet model from: {model_path}")
    asr_model = nemo_asr.models.ASRModel.restore_from(model_path)
    
    # Configure for long-form audio if context sizes are not default
    if context_left != 256 or context_right != 256:
        print(f"Configuring attention context: left={context_left}, right={context_right}")
        try:
            asr_model.change_attention_model(
                self_attention_model="rel_pos_local_attn",
                att_context_size=[context_left, context_right]
            )
            print("Long-form audio mode enabled")
        except Exception as e:
            print(f"Warning: Failed to configure attention model: {e}")
            print("Continuing with default attention settings")
    
    print(f"Transcribing: {audio_path}")
    
    if timestamps:
        output = asr_model.transcribe([audio_path], timestamps=True)
        
        # Extract text and timestamps
        result_data = output[0]
        text = result_data.text
        word_timestamps = result_data.timestamp.get("word", [])
        segment_timestamps = result_data.timestamp.get("segment", [])
        
        print(f"Transcription: {text}")
        
        # Prepare output data
        output_data = {
            "transcription": text,
            "language": "en",
            "word_timestamps": word_timestamps,
            "segment_timestamps": segment_timestamps,
            "audio_file": audio_path,
            "model": "parakeet-tdt-0.6b-v3",
            "context": {
                "left": context_left,
                "right": context_right
            }
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
        # Simple transcription without timestamps
        output = asr_model.transcribe([audio_path])
        text = output[0].text
        
        output_data = {
            "transcription": text,
            "language": "en", 
            "audio_file": audio_path,
            "model": "parakeet-tdt-0.6b-v3"
        }
        
        if output_file:
            with open(output_file, 'w', encoding='utf-8') as f:
                json.dump(output_data, f, indent=2, ensure_ascii=False)
            print(f"Results saved to: {output_file}")
        else:
            print(json.dumps(output_data, indent=2, ensure_ascii=False))


def main():
    parser = argparse.ArgumentParser(
        description="Transcribe audio using NVIDIA Parakeet model"
    )
    parser.add_argument("audio_file", help="Path to audio file")
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
        "--context-left", type=int, default=256,
        help="Left attention context size (default: 256)"
    )
    parser.add_argument(
        "--context-right", type=int, default=256,
        help="Right attention context size (default: 256)"
    )
    parser.add_argument(
        "--include-confidence", action="store_true", default=True,
        help="Include confidence scores"
    )
    parser.add_argument(
        "--no-confidence", dest="include_confidence", action="store_false",
        help="Exclude confidence scores"
    )
    
    args = parser.parse_args()
    
    # Validate input file
    if not os.path.exists(args.audio_file):
        print(f"Error: Audio file not found: {args.audio_file}")
        sys.exit(1)
    
    try:
        transcribe_audio(
            audio_path=args.audio_file,
            timestamps=args.timestamps,
            output_file=args.output,
            context_left=args.context_left,
            context_right=args.context_right,
            include_confidence=args.include_confidence,
        )
    except Exception as e:
        print(f"Error during transcription: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
