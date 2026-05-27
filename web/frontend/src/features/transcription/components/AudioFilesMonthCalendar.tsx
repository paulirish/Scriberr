import { useEffect, useRef, useState } from "react";
import { type AudioFile } from "@/features/transcription/hooks/useAudioFiles";
import "@/components/calendar/audio-files-month-calendar";
import monthHtml from "@/components/calendar/audio-files-month-calendar.html?raw";

interface AudioFilesMonthCalendarProps {
  data: AudioFile[];
  onFileClick: (fileId: string) => void;
  onFileHoverStart?: (fileId: string) => void;
  onFileHoverEnd?: () => void;
}

export const AudioFilesMonthCalendar = ({ 
  data, 
  onFileClick
}: AudioFilesMonthCalendarProps) => {
  const [currentDate, setCurrentDate] = useState(new Date());
  const calendarRef = useRef<any>(null);

  useEffect(() => {
    const calendar = calendarRef.current;
    if (!calendar) return;

    calendar.data = data;
    calendar.currentDate = currentDate;

    const handleFileClick = (e: any) => onFileClick(e.detail.fileId);
    const handlePrevMonth = () => {
      setCurrentDate(prev => new Date(prev.getFullYear(), prev.getMonth() - 1, 1));
    };
    const handleNextMonth = () => {
      setCurrentDate(prev => new Date(prev.getFullYear(), prev.getMonth() + 1, 1));
    };

    calendar.addEventListener('file-click', handleFileClick);
    calendar.addEventListener('prev-month', handlePrevMonth);
    calendar.addEventListener('next-month', handleNextMonth);

    return () => {
      calendar.removeEventListener('file-click', handleFileClick);
      calendar.removeEventListener('prev-month', handlePrevMonth);
      calendar.removeEventListener('next-month', handleNextMonth);
    };
  }, [data, currentDate, onFileClick]);

  const CalendarTag = 'audio-files-month-calendar' as any;

  return (
    <CalendarTag
      ref={calendarRef}
      dangerouslySetInnerHTML={{ __html: monthHtml }}
    />
  );
};
