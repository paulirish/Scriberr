import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export const parseTitleForDate = (title: string): { date: Date; duration: number } | null => {
  const regex = /^(\d{4})_(\d{2})_(\d{2})_[a-zA-Z]{3}_(AM|PM)_(\d{2})_(\d{2})_(\d{2})__(\d+)min$/;
  const match = title.match(regex);

  if (!match) return null;

  const [, year, month, day, meridiem, hour, minute, second, duration] = match;

  let hour24 = parseInt(hour, 10);
  if (meridiem === "PM" && hour24 < 12) {
    hour24 += 12;
  }
  if (meridiem === "AM" && hour24 === 12) {
    hour24 = 0;
  }

  return {
    date: new Date(parseInt(year, 10), parseInt(month, 10) - 1, parseInt(day, 10), hour24, parseInt(minute, 10), parseInt(second, 10)),
    duration: parseInt(duration, 10),
  };
};

export const formatAudioFileTitle = (title: string): string => {
  const parsed = parseTitleForDate(title);
  if (!parsed) {
    return title; // fallback to original title
  }

  const { date, duration } = parsed;
  const hours = date.getHours();
  const minutes = date.getMinutes();
  const ampm = hours >= 12 ? 'pm' : 'am';
  const formattedHours = hours % 12 === 0 ? 12 : hours % 12;
  const formattedMinutes = minutes < 10 ? `0${minutes}` : minutes;

  return `${duration}m | ${formattedHours}:${formattedMinutes}${ampm}`;
};

interface SpeakerMapping {
  original_speaker: string;
  custom_name: string;
}

interface SpeakerInfo {
  id: string;
  name: string;
}

export const getSpeakersFromAudioFile = (file: { 
  transcript?: string; 
  speaker_mappings?: SpeakerMapping[];
  speakers?: SpeakerInfo[];
}): SpeakerInfo[] => {
  // Priority 1: Use pre-resolved speakers from the API (The "Modern" way)
  if (file.speakers && file.speakers.length > 0) {
    return file.speakers;
  }

  // Priority 2: Fallback to manual resolution (The "Legacy" way)
  if (!file.transcript) {
    return file.speaker_mappings?.map(m => ({
      id: m.original_speaker,
      name: m.custom_name || m.original_speaker
    })) || [];
  }

  try {
    const transcript = JSON.parse(file.transcript);
    const speakerIds = new Set<string>();
    
    transcript.segments?.forEach((seg: any) => {
      if (seg.speaker) {
        speakerIds.add(seg.speaker);
      }
    });

    if (speakerIds.size === 0) {
      return file.speaker_mappings?.map(m => m.custom_name || m.original_speaker) || [];
    }

    const mappingMap = new Map<string, string>();
    file.speaker_mappings?.forEach(m => {
      mappingMap.set(m.original_speaker, m.custom_name);
    });

    return Array.from(speakerIds).map(id => ({
      id: id,
      name: mappingMap.get(id) || id
    })).sort((a, b) => a.name.localeCompare(b.name));
  } catch (e) {
    return file.speaker_mappings?.map(m => ({
      id: m.original_speaker,
      name: m.custom_name || m.original_speaker
    })) || [];
  }
};

