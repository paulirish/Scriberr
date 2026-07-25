import { useEffect, useRef, useState, useCallback, type RefObject } from "react";
import { type AudioFile } from "@/features/transcription/hooks/useAudioFiles";
import "@/components/calendar/audio-files-week-calendar";
import weekHtml from "@/components/calendar/audio-files-week-calendar.html?raw";

interface AudioFilesWeekCalendarProps {
  data: AudioFile[];
  onFileClick: (fileId: string) => void;
  onFileHoverStart?: (fileId: string) => void;
  onFileHoverEnd?: () => void;
}

interface CalendarElement extends HTMLElement {
  data: AudioFile[];
  baseDate: Date;
}

export const AudioFilesWeekCalendar = ({ 
  data, 
  onFileClick,
  onFileHoverStart,
  onFileHoverEnd
}: AudioFilesWeekCalendarProps) => {
  const [baseDate, setBaseDate] = useState(new Date());
  const calendarRef = useRef<CalendarElement>(null);

  const handleFileClick = useCallback((e: Event) => {
    const customEvent = e as CustomEvent<{ fileId: string }>;
    onFileClick(customEvent.detail.fileId);
  }, [onFileClick]);

  const handleFileHoverStart = useCallback((e: Event) => {
    const customEvent = e as CustomEvent<{ fileId: string }>;
    onFileHoverStart?.(customEvent.detail.fileId);
  }, [onFileHoverStart]);

  const handleFileHoverEnd = useCallback(() => {
    onFileHoverEnd?.();
  }, [onFileHoverEnd]);

  const handlePrevWeeks = useCallback(() => {
    setBaseDate(prev => {
      const newDate = new Date(prev);
      newDate.setDate(newDate.getDate() - 28);
      return newDate;
    });
  }, []);

  const handleNextWeeks = useCallback(() => {
    setBaseDate(prev => {
      const newDate = new Date(prev);
      newDate.setDate(newDate.getDate() + 28);
      return newDate;
    });
  }, []);

  useEffect(() => {
    const calendar = calendarRef.current;
    if (!calendar) return;

    calendar.data = data;
    calendar.baseDate = baseDate;

    calendar.addEventListener('file-click', handleFileClick);
    calendar.addEventListener('file-hover-start', handleFileHoverStart);
    calendar.addEventListener('file-hover-end', handleFileHoverEnd);
    calendar.addEventListener('prev-weeks', handlePrevWeeks);
    calendar.addEventListener('next-weeks', handleNextWeeks);

    return () => {
      calendar.removeEventListener('file-click', handleFileClick);
      calendar.removeEventListener('file-hover-start', handleFileHoverStart);
      calendar.removeEventListener('file-hover-end', handleFileHoverEnd);
      calendar.removeEventListener('prev-weeks', handlePrevWeeks);
      calendar.removeEventListener('next-weeks', handleNextWeeks);
    };
  }, [data, baseDate, handleFileClick, handleFileHoverStart, handleFileHoverEnd, handlePrevWeeks, handleNextWeeks]);

  return (
    // @ts-expect-error - Custom element
    <audio-files-week-calendar
      ref={calendarRef as unknown as RefObject<HTMLElement>}
      dangerouslySetInnerHTML={{ __html: weekHtml }}
    />
  );
};
