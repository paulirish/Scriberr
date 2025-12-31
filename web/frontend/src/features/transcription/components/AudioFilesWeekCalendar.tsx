import { useState, useMemo } from "react";
import { ChevronLeft, ChevronRight, FileAudio, Clock } from "lucide-react";
import { Button } from "@/components/ui/button";
import { formatAudioFileTitle, parseTitleForDate, cn } from "@/lib/utils";
import { type AudioFile } from "@/features/transcription/hooks/useAudioFiles";

interface AudioFilesWeekCalendarProps {
  data: AudioFile[];
  onFileClick: (fileId: string) => void;
}

export const AudioFilesWeekCalendar = ({ data, onFileClick }: AudioFilesWeekCalendarProps) => {
  const [currentDate, setCurrentDate] = useState(new Date());

  const startOfWeek = useMemo(() => {
    const date = new Date(currentDate);
    const day = date.getDay();
    // Adjust to Sunday as start of week
    const diff = date.getDate() - day;
    return new Date(new Date(date.setDate(diff)).setHours(0, 0, 0, 0));
  }, [currentDate]);

  const days = useMemo(() => {
    const weekDays = [];
    for (let i = 0; i < 7; i++) {
      const date = new Date(startOfWeek);
      date.setDate(startOfWeek.getDate() + i);
      weekDays.push(date);
    }
    return weekDays;
  }, [startOfWeek]);

  const handlePrevWeek = () => {
    const newDate = new Date(currentDate);
    newDate.setDate(newDate.getDate() - 7);
    setCurrentDate(newDate);
  };

  const handleNextWeek = () => {
    const newDate = new Date(currentDate);
    newDate.setDate(newDate.getDate() + 7);
    setCurrentDate(newDate);
  };

  const filesWithDateTime = useMemo(() => {
    return data.map(file => {
      if (file.title) {
        const parsed = parseTitleForDate(file.title);
        if (parsed) {
          return { ...file, dateTime: parsed.date, duration: parsed.duration };
        }
      }
      return { ...file, dateTime: new Date(file.created_at), duration: file.duration || 15 };
    });
  }, [data]);

  const hours = Array.from({ length: 24 }, (_, i) => i);

  const renderEventsForDay = (day: Date) => {
    const dayEvents = filesWithDateTime.filter(file => file.dateTime.toDateString() === day.toDateString());
    return dayEvents.map(event => {
      const startMinutes = event.dateTime.getHours() * 60 + event.dateTime.getMinutes();
      const top = (startMinutes / 1440) * 100;
      const height = (Math.max(event.duration, 20) / 1440) * 100; // Minimum 20 mins for visibility

      return (
        <div
          key={event.id}
          className="absolute left-1 right-1 p-1.5 bg-[#FFFAF0] dark:bg-orange-950/20 border border-orange-100 dark:border-orange-900/30 rounded-lg cursor-pointer hover:border-[var(--brand-solid)] hover:shadow-md transition-all z-10 overflow-hidden group"
          style={{ top: `${top}%`, height: `${height}%`, minHeight: '24px' }}
          onClick={() => onFileClick(event.id)}
        >
          <div className="flex items-center gap-1 min-w-0">
            <FileAudio className="h-3 w-3 text-[#FF6D20] flex-shrink-0" />
            <p className="text-[10px] text-gray-700 dark:text-gray-300 truncate font-semibold group-hover:text-[#FF6D20]">
              {event.title ? formatAudioFileTitle(event.title) : `File ${event.id.substring(0, 8)}`}
            </p>
          </div>
        </div>
      );
    });
  };

  return (
    <div className="glass-card rounded-[var(--radius-card)] overflow-hidden border border-[var(--border-subtle)] shadow-[var(--shadow-float)] bg-[var(--bg-card)]">
      <div className="flex justify-between items-center p-4 border-b border-[var(--border-subtle)] bg-[var(--bg-card)]">
        <h2 className="text-lg font-bold text-[var(--text-primary)]">
          {startOfWeek.toLocaleString("default", { month: "short", day: "numeric" })} - {new Date(startOfWeek.getTime() + 6 * 24 * 60 * 60 * 1000).toLocaleString("default", { month: "short", day: "numeric", year: "numeric" })}
        </h2>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={handlePrevWeek} className="h-8 border-[var(--border-subtle)]">
            <ChevronLeft className="h-4 w-4 mr-1" />
            Prev
          </Button>
          <Button variant="outline" size="sm" onClick={handleNextWeek} className="h-8 border-[var(--border-subtle)]">
            Next
            <ChevronRight className="h-4 w-4 ml-1" />
          </Button>
        </div>
      </div>

      <div className="flex flex-col h-[600px] overflow-y-auto custom-scrollbar">
        <div className="grid grid-cols-[60px_1fr] sticky top-0 z-20 bg-[var(--bg-main)] border-b border-[var(--border-subtle)]">
          <div className="border-r border-[var(--border-subtle)] bg-[var(--bg-main)] flex items-center justify-center">
            <Clock className="h-4 w-4 text-[var(--text-tertiary)]" />
          </div>
          <div className="grid grid-cols-7">
            {days.map(day => (
              <div key={day.toISOString()} className={cn(
                "text-center py-3 border-r border-[var(--border-subtle)] last:border-r-0",
                new Date().toDateString() === day.toDateString() ? "bg-[var(--brand-solid)]/5" : ""
              )}>
                <div className="text-[10px] font-bold text-[var(--text-tertiary)] uppercase tracking-wider">{day.toLocaleString("default", { weekday: "short" })}</div>
                <div className={cn(
                  "text-sm font-bold",
                  new Date().toDateString() === day.toDateString() ? "text-[var(--brand-solid)]" : "text-[var(--text-primary)]"
                )}>{day.getDate()}</div>
              </div>
            ))}
          </div>
        </div>

        <div className="grid grid-cols-[60px_1fr] flex-1 min-h-[1440px]">
          <div className="border-r border-[var(--border-subtle)] bg-[var(--bg-main)]/50">
            {hours.map(hour => (
              <div key={hour} className="h-[60px] text-[10px] font-medium text-[var(--text-tertiary)] text-right pr-2 pt-1 border-b border-[var(--border-subtle)]/30">
                {hour === 0 ? '12 AM' : hour < 12 ? `${hour} AM` : hour === 12 ? '12 PM' : `${hour - 12} PM`}
              </div>
            ))}
          </div>
          <div className="grid grid-cols-7 relative">
            {days.map(day => (
              <div key={day.toISOString()} className="relative border-r border-[var(--border-subtle)] last:border-r-0">
                {hours.map(hour => (
                  <div key={hour} className="h-[60px] border-b border-[var(--border-subtle)]/30 last:border-b-0" />
                ))}
                {renderEventsForDay(day)}
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
};

