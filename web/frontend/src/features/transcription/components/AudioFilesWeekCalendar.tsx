import { useEffect, useRef, useState } from "react";
import { type AudioFile } from "@/features/transcription/hooks/useAudioFiles";
import "@/components/calendar/audio-files-week-calendar";
import weekHtml from "@/components/calendar/audio-files-week-calendar.html?raw";

interface AudioFilesWeekCalendarProps {
  data: AudioFile[];
  onFileClick: (fileId: string) => void;
  onFileHoverStart?: (fileId: string) => void;
  onFileHoverEnd?: () => void;
}

export const AudioFilesWeekCalendar = ({ 
  data, 
  onFileClick
}: AudioFilesWeekCalendarProps) => {
  const [baseDate, setBaseDate] = useState(new Date());
  const calendarRef = useRef<any>(null);

  useEffect(() => {
    const calendar = calendarRef.current;
    if (!calendar) return;

    calendar.data = data;
    calendar.baseDate = baseDate;

    const handleFileClick = (e: any) => onFileClick(e.detail.fileId);
    const handlePrevWeeks = () => {
      setBaseDate(prev => {
        const newDate = new Date(prev);
        newDate.setDate(newDate.getDate() - 28);
        return newDate;
      });
    };
    const handleNextWeeks = () => {
      setBaseDate(prev => {
        const newDate = new Date(prev);
        newDate.setDate(newDate.getDate() + 28);
        return newDate;
      });
    };

    calendar.addEventListener('file-click', handleFileClick);
    calendar.addEventListener('prev-weeks', handlePrevWeeks);
    calendar.addEventListener('next-weeks', handleNextWeeks);

    return () => {
      calendar.removeEventListener('file-click', handleFileClick);
      calendar.removeEventListener('prev-weeks', handlePrevWeeks);
      calendar.removeEventListener('next-weeks', handleNextWeeks);
    };
  }, [data, baseDate, onFileClick]);

  const CalendarTag = 'audio-files-week-calendar' as any;

  return (
    <CalendarTag
      ref={calendarRef}
      dangerouslySetInnerHTML={{ __html: weekHtml }}
    />
  );
};
