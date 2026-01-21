# NVIDIA Parakeet Transcription Approaches

This document outlines the differences between the standard and buffered transcription approaches used with the NVIDIA Parakeet model in Scriberr.

## 1. `parakeet_transcribe.py` (Standard Approach)
*   **Processing:** Transcribes the **entire audio file in one go**.
*   **Memory Management:** Relies on the model's internal capability to handle the audio. If the file is very long, it may run out of GPU memory (OOM).
*   **Long-form Optimization:** Attempts to handle long-form audio by adjusting the model's attention context (`change_attention_model`) using `context_left` and `context_right` parameters (default 256).
*   **Use Case:** Best for shorter clips or when you have sufficient GPU memory to process the full file at once.

## 2. `parakeet_transcribe_buffered.py` (Buffered Approach)
*   **Processing:** Manually **splits the audio into fixed-size chunks** (defaulting to 300 seconds / 5 minutes) before transcription.
*   **Memory Management:** Designed specifically to **avoid GPU memory issues**. By processing smaller segments, it keeps the memory footprint low and predictable regardless of the total audio length.
*   **Implementation Details:**
    *   Uses `librosa` to load and slice the audio.
    *   Saves chunks to temporary files (`/tmp/chunk_i.wav`) for processing.
    *   **Timestamp Reconstitution:** It manually offsets the timestamps of words and segments by the start time of each chunk so the final output matches the original file's timeline.
*   **Use Case:** Necessary for very long recordings (e.g., hour-long meetings) that would otherwise crash the GPU.

## Summary Table

| Feature | `parakeet_transcribe` | `parakeet_transcribe_buffered` |
| :--- | :--- | :--- |
| **Strategy** | Single pass | Chunked/Iterative |
| **GPU Memory** | Higher (proportional to length) | Low and constant |
| **Complexity** | Low | Higher (handles splitting & merging) |
| **Long-form Handling** | Attention context adjustment | Physical audio splitting |
| **Dependencies** | NeMo | NeMo + `librosa` + `soundfile` |
