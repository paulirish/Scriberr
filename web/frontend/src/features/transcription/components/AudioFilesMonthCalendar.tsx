import { useEffect, useRef, useState, useCallback, type RefObject } from "react";
import { type AudioFile } from "@/features/transcription/hooks/useAudioFiles";
import "@/components/calendar/audio-files-month-calendar";
import monthHtml from "@/components/calendar/audio-files-month-calendar.html?raw";

interface AudioFilesMonthCalendarProps {
  data: AudioFile[];
  onFileClick: (fileId: string) => void;
  onFileHoverStart?: (fileId: string) => void;
  onFileHoverEnd?: () => void;
}

interface CalendarElement extends HTMLElement {
  data: AudioFile[];
  currentDate: Date;
}

export const AudioFilesMonthCalendar = ({ 
  data, 
  onFileClick,
  onFileHoverStart,
  onFileHoverEnd
}: AudioFilesMonthCalendarProps) => {
  const [currentDate, setCurrentDate] = useState(new Date());
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

  const handlePrevMonth = useCallback(() => {
    setCurrentDate(prev => new Date(prev.getFullYear(), prev.getMonth() - 1, 1));
  }, []);

  const handleNextMonth = useCallback(() => {
    setCurrentDate(prev => new Date(prev.getFullYear(), prev.getMonth() + 1, 1));
  }, []);

  useEffect(() => {
    const calendar = calendarRef.current;
    if (!calendar) return;

    calendar.data = data;
    calendar.currentDate = currentDate;

    calendar.addEventListener('file-click', handleFileClick);
    calendar.addEventListener('file-hover-start', handleFileHoverStart);
    calendar.addEventListener('file-hover-end', handleFileHoverEnd);
    calendar.addEventListener('prev-month', handlePrevMonth);
    calendar.addEventListener('next-month', handleNextMonth);

    return () => {
      calendar.removeEventListener('file-click', handleFileClick);
      calendar.removeEventListener('file-hover-start', handleFileHoverStart);
      calendar.removeEventListener('file-hover-end', handleFileHoverEnd);
      calendar.removeEventListener('prev-month', handlePrevMonth);
      calendar.removeEventListener('next-month', handleNextMonth);
    };
  }, [data, currentDate, handleFileClick, handleFileHoverStart, handleFileHoverEnd, handlePrevMonth, handleNextMonth]);

  return (
    // @ts-expect-error - Custom element
    <audio-files-month-calendar
      ref={calendarRef as unknown as RefObject<HTMLElement>}
      dangerouslySetInnerHTML={{ __html: monthHtml }}
    />
  );
};
