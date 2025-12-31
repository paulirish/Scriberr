
/**
 * JobStatus represents the status of a transcription job
 */
export const JobStatus = {
  Uploaded: "uploaded",
  Pending: "pending",
  Processing: "processing",
  Completed: "completed",
  Failed: "failed",
} as const;

export type JobStatus = typeof JobStatus[keyof typeof JobStatus];

/**
 * WhisperXParams contains parameters for WhisperX transcription
 */
export interface WhisperXParams {
  model_family?: string;
  model?: string;
  model_cache_only?: boolean;
  model_dir?: string;
  device?: string;
  device_index?: number;
  batch_size?: number;
  compute_type?: string;
  threads?: number;
  output_format?: string;
  verbose?: boolean;
  task?: string;
  language?: string;
  align_model?: string;
  interpolate_method?: string;
  no_align?: boolean;
  return_char_alignments?: boolean;
  vad_method?: string;
  vad_onset?: number;
  vad_offset?: number;
  chunk_size?: number;
  diarize?: boolean;
  min_speakers?: number | undefined;
  max_speakers?: number | undefined;
  diarize_model?: string;
  speaker_embeddings?: boolean;
  temperature?: number;
  best_of?: number;
  beam_size?: number;
  patience?: number;
  length_penalty?: number;
  suppress_tokens?: string;
  suppress_numerals?: boolean;
  initial_prompt?: string;
  condition_on_previous_text?: boolean;
  fp16?: boolean;
  temperature_increment_on_fallback?: number;
  compression_ratio_threshold?: number;
  logprob_threshold?: number;
  no_speech_threshold?: number;
  max_line_width?: number;
  max_line_count?: number;
  highlight_words?: boolean;
  segment_resolution?: string;
  hf_token?: string;
  print_progress?: boolean;
  attention_context_left?: number;
  attention_context_right?: number;
  is_multi_track_enabled?: boolean;
  callback_url?: string;
  api_key?: string;
}

/**
 * MultiTrackFile represents an individual audio track in a multi-track recording
 */
export interface MultiTrackFile {
  id: number;
  transcription_job_id: string;
  file_name: string;
  file_path: string;
  track_index: number;
  offset: number;
  gain: number;
  pan: number;
  mute: boolean;
  created_at: string;
  updated_at: string;
}

/**
 * TranscriptionJob represents a transcription job record
 * Matches internal/models/transcription.go
 */
export interface TranscriptionJob {
  id: string;
  title?: string;
  status: JobStatus;
  audio_path: string;
  transcript?: string;
  diarization: boolean;
  summary?: string;
  error_message?: string;
  is_multi_track: boolean;
  aup_file_path?: string;
  multi_track_folder?: string;
  merged_audio_path?: string;
  merge_status: string;
  merge_error?: string;
  individual_transcripts?: string; // JSON-serialized map[string]*string
  created_at: string;
  updated_at: string;
  deleted_at?: string;

  // WhisperX parameters
  parameters: WhisperXParams;

  // Relationships
  multi_track_files?: MultiTrackFile[];

  // Computed fields (Frontend only)
  speakers?: number; // Count of unique speakers
  duration?: number; // Duration in seconds
}

/**
 * AudioFile is an alias for TranscriptionJob to maintain UI nomenclature
 */
export type AudioFile = TranscriptionJob;

/**
 * MultiTrackTiming represents timing data for individual track processing
 */
export interface MultiTrackTiming {
  track_name: string;
  start_time: string;
  end_time: string;
  duration: number; // Duration in milliseconds
}

/**
 * ExecutionData represents execution metadata for completed transcription jobs
 * Aligned with the enhanced JSON response from the job execution handler
 */
export interface ExecutionData {
  id: string;
  transcription_job_id: string;
  started_at: string;
  completed_at?: string;
  processing_duration?: number;
  actual_parameters?: WhisperXParams;
  status: JobStatus;
  error_message?: string;
  created_at: string;
  updated_at: string;

  // Enhanced fields from GetJobExecutionData handler
  is_multi_track: boolean;
  multi_track_timings?: MultiTrackTiming[];
  merge_start_time?: string;
  merge_end_time?: string;
  merge_duration?: number;
  multi_track_files?: MultiTrackFile[];

  // Graceful empty response fields
  available?: boolean;
  message?: string;
}

/**
 * SpeakerMapping represents custom speaker names for a transcription job
 */
export interface SpeakerMapping {
  id: number;
  transcription_job_id: string;
  original_speaker: string;
  custom_name: string;
  created_at: string;
  updated_at: string;
}

/**
 * TranscriptionProfile represents a saved transcription configuration profile
 */
export interface TranscriptionProfile {
  id: string;
  name: string;
  description?: string;
  is_default: boolean;
  parameters: WhisperXParams;
  created_at: string;
  updated_at: string;
}

/**
 * WordSegment represents a single word in the transcript
 */
export interface WordSegment {
    start: number;
    end: number;
    word: string;
    score: number;
    speaker?: string;
}

/**
 * TranscriptSegment represents a segment of the transcript
 */
export interface TranscriptSegment {
    start: number;
    end: number;
    text: string;
    speaker?: string;
}

/**
 * Transcript represents the parsed transcript structure
 */
export interface Transcript {
    text: string;
    segments?: TranscriptSegment[];
    word_segments?: WordSegment[];
}
