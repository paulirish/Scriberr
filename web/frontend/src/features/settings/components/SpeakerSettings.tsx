import { useState, useEffect, useRef, useCallback } from "react";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { speakersApi, type Speaker, type SpeakerSegment } from "@/lib/speakersApi";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Play, Trash2, Edit2, Check, X, Volume2 } from "lucide-react";
import { toast } from "sonner";
import { getSpeakerColorStyles, speakerColorClass } from "@/lib/speakerColors";
import { cn } from "@/lib/utils";

export function SpeakerSettings() {
  const { getAuthHeaders } = useAuth();
  const [speakers, setSpeakers] = useState<Speaker[]>([]);
  const [loading, setLoading] = useState(true);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editName, setEditName] = useState("");

  // Single Audio Instance for voice samples
  const audioRef = useRef<HTMLAudioElement>(new Audio());
  const [activeSegmentId, setActiveSegmentId] = useState<number | null>(null);

  const stopAudio = useCallback(() => {
    const audio = audioRef.current;
    audio.pause();
    audio.ontimeupdate = null;
    setActiveSegmentId(null);
  }, []);

  const playSegment = useCallback((url: string, start: number, end: number, segmentId: number) => {
    const audio = audioRef.current;

    // Stop current if any
    audio.pause();
    audio.ontimeupdate = null;

    if (!audio.src.includes(url)) {
      audio.src = url;
    }

    audio.currentTime = start;
    audio.ontimeupdate = () => {
      if (audio.currentTime >= end) {
        audio.pause();
        audio.ontimeupdate = null;
        setActiveSegmentId(null);
      }
    };

    audio.play().catch(console.error);
    setActiveSegmentId(segmentId);
  }, []);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      const audio = audioRef.current;
      audio.pause();
      audio.src = "";
    };
  }, []);

  const fetchSpeakers = async () => {
    try {
      const data = await speakersApi.list(getAuthHeaders);
      setSpeakers(data);
    } catch (error) {
      console.error(error);
      toast.error("Failed to fetch speakers");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchSpeakers();
  }, []);

  const handleRename = async (id: string) => {
    try {
      await speakersApi.rename(id, editName, getAuthHeaders);
      toast.success("Speaker renamed");
      setEditingId(null);
      fetchSpeakers();
    } catch (error) {
      toast.error("Failed to rename speaker");
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("Are you sure you want to delete this speaker profile? This will not delete transcripts but will remove the voice identity.")) return;
    try {
      await speakersApi.delete(id, getAuthHeaders);
      toast.success("Speaker deleted");
      fetchSpeakers();
    } catch (error) {
      toast.error("Failed to delete speaker");
    }
  };

  if (loading) return <div className="p-8 text-center">Loading speakers...</div>;

  return (
    <div className="space-y-6">
      <div className="bg-[var(--bg-card)] border border-[var(--border-subtle)] rounded-[var(--radius-card)] p-4 sm:p-6 shadow-sm">
        <div className="mb-6">
          <h3 className="text-lg font-medium text-[var(--text-primary)]">
            Identified Speakers
          </h3>
          <p className="text-sm text-[var(--text-secondary)] mt-1">
            Manage persistent speaker identities and listen to their voice samples.
          </p>
        </div>

        <div className="space-y-4">

          {speakers
            // .filter(s => s.id === "72e4deac-037e-4f61-b301-efb685d53f89" || s.id === "2d18b992-c7ec-436f-85d7-aa82b1e1267f")
            .map((speaker) => (
              <SpeakerRow
                key={speaker.id}
                speaker={speaker}
                onDelete={() => handleDelete(speaker.id)}
                isEditing={editingId === speaker.id}
                onEditStart={() => {
                  setEditingId(speaker.id);
                  setEditName(speaker.name);
                }}
                onEditCancel={() => setEditingId(null)}
                onEditSave={() => handleRename(speaker.id)}
                editName={editName}
                onEditNameChange={setEditName}
                activeSegmentId={activeSegmentId}
                onPlaySegment={playSegment}
                onStopAudio={stopAudio}
              />
            ))}
          {speakers.length === 0 && (
            <div className="text-center py-8 text-[var(--text-tertiary)]">
              No speakers identified yet. Process some audio with diarization enabled to see them here.
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function SpeakerRow({
  speaker,
  onDelete,
  isEditing,
  onEditStart,
  onEditCancel,
  onEditSave,
  editName,
  onEditNameChange,
  activeSegmentId,
  onPlaySegment,
  onStopAudio
}: {
  speaker: Speaker;
  onDelete: () => void;
  isEditing: boolean;
  onEditStart: () => void;
  onEditCancel: () => void;
  onEditSave: () => void;
  editName: string;
  onEditNameChange: (val: string) => void;
  activeSegmentId: number | null;
  onPlaySegment: (url: string, start: number, end: number, id: number) => void;
  onStopAudio: () => void;
}) {
  const { getAuthHeaders } = useAuth();
  const [segments, setSegments] = useState<SpeakerSegment[]>([]);
  const [loadingSegments, setLoadingSegments] = useState(false);

  useEffect(() => {
    const fetchSegments = async () => {
      setLoadingSegments(true);
      try {
        const data = await speakersApi.getSegments(speaker.id, getAuthHeaders);
        setSegments(data);
      } catch (error) {
        console.error(error);
      } finally {
        setLoadingSegments(false);
      }
    };
    fetchSegments();
  }, [speaker.id]);

  return (
    <div className="p-4 border border-[var(--border-subtle)] rounded-lg bg-[var(--bg-main)]/30 transition-all hover:bg-[var(--bg-main)]/50">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-3 flex-1">
          {isEditing ? (
            <div className="flex items-center gap-2 flex-1 max-w-md">
              <Input
                value={editName}
                onChange={(e) => onEditNameChange(e.target.value)}
                className="h-8"
                autoFocus
                onKeyDown={(e) => e.key === 'Enter' && onEditSave()}
              />
              <Button size="icon" variant="ghost" className="h-8 w-8 text-green-500" onClick={onEditSave}>
                <Check className="h-4 w-4" />
              </Button>
              <Button size="icon" variant="ghost" className="h-8 w-8 text-red-500" onClick={onEditCancel}>
                <X className="h-4 w-4" />
              </Button>
            </div>
          ) : (
            <>
              <h4 
                style={getSpeakerColorStyles(speaker.name)}
                className={cn(
                  "font-medium px-2 py-0.5 rounded-full text-sm border",
                  speakerColorClass
                )}
              >
                {speaker.name}
              </h4>
              <button
                onClick={onEditStart}
                className="p-1 text-[var(--text-tertiary)] hover:text-[var(--text-primary)] transition-colors"
              >
                <Edit2 className="h-3 w-3" />
              </button>
            </>
          )}
        </div>
        <div className="flex items-center gap-2">
          <span className="text-xs text-[var(--text-tertiary)] hidden sm:inline">
            ID: {speaker.id.slice(0, 8)}...
          </span>
          <Button size="icon" variant="ghost" className="h-8 w-8 text-[var(--text-tertiary)] hover:text-red-500" onClick={onDelete}>
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      </div>

      <div className="flex flex-wrap gap-2">
        {loadingSegments ? (
          <div className="text-xs text-[var(--text-tertiary)] animate-pulse">Loading samples...</div>
        ) : (
          segments.map((seg) => (
            <AudioChip
              key={seg.id}
              segment={seg}
              speakerName={speaker.name}
              isPlaying={activeSegmentId === seg.id}
              onPlay={() => onPlaySegment(`/api/v1/transcription/${seg.transcription_job_id}/audio`, seg.start, seg.end, seg.id)}
              onStop={onStopAudio}
            />
          ))
        )}
      </div>
    </div>
  );
}

function AudioChip({
  segment,
  speakerName,
  isPlaying,
  onPlay,
  onStop
}: {
  segment: SpeakerSegment;
  speakerName: string;
  isPlaying: boolean;
  onPlay: () => void;
  onStop: () => void;
}) {
  const duration = (segment.end - segment.start).toFixed(1);

  return (
    <div
      onMouseEnter={onPlay}
      onMouseLeave={onStop}
      onClick={onPlay}
      style={getSpeakerColorStyles(speakerName)}
      className={cn(
        "group flex items-center gap-2 px-3 py-1.5 rounded-full text-xs font-medium cursor-pointer transition-all",
        isPlaying
          ? "bg-[var(--brand-gradient)] text-black shadow-md border-transparent"
          : cn("bg-[var(--bg-main)] border hover:border-[var(--brand-primary)]", speakerColorClass)
      )}
      title={segment.text}
    >
      {isPlaying ? <Volume2 className="h-3 w-3 animate-pulse" /> : <Play className="h-3 w-3 opacity-50 group-hover:opacity-100" />}
      <span>{duration}s</span>
      <span className="max-w-[100px] truncate opacity-70 font-normal hidden md:inline">
        {segment.text}
      </span>
    </div>
  );
}