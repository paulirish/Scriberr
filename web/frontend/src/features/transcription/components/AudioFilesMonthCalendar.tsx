import { useState, useMemo } from "react";
import { ChevronLeft, ChevronRight, FileAudio } from "lucide-react";
import { Button } from "@/components/ui/button";
import { formatAudioFileTitle, parseTitleForDate, cn } from "@/lib/utils";
import { type AudioFile } from "@/features/transcription/hooks/useAudioFiles";

interface AudioFilesMonthCalendarProps {
  data: AudioFile[];
  onFileClick: (fileId: string) => void;
}

const daysOfWeek = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

export const AudioFilesMonthCalendar = ({ data, onFileClick }: AudioFilesMonthCalendarProps) => {
  const [currentDate, setCurrentDate] = useState(new Date());

  const firstDayOfMonth = new Date(currentDate.getFullYear(), currentDate.getMonth(), 1);
  const lastDayOfMonth = new Date(currentDate.getFullYear(), currentDate.getMonth() + 1, 0);

  const startingDay = firstDayOfMonth.getDay();
  const totalDays = lastDayOfMonth.getDate();

  const handlePrevMonth = () => {
    setCurrentDate(new Date(currentDate.getFullYear(), currentDate.getMonth() - 1, 1));
  };

  const handleNextMonth = () => {
    setCurrentDate(new Date(currentDate.getFullYear(), currentDate.getMonth() + 1, 1));
  };

  const filesByDate = useMemo(() => {
    return data.reduce((acc, file) => {
      let date: Date;
      if (file.title) {
        const parsed = parseTitleForDate(file.title);
        date = parsed ? parsed.date : new Date(file.created_at);
      } else {
        date = new Date(file.created_at);
      }
      const dateString = date.toDateString();
      if (!acc[dateString]) {
        acc[dateString] = [];
      }
      acc[dateString].push(file);
      return acc;
    }, {} as Record<string, AudioFile[]>);
  }, [data]);

  const renderDays = () => {
    const days = [];
    // Add empty cells for days before the start of the month
    for (let i = 0; i < startingDay; i++) {
      days.push(<div key={`empty-${i}`} className="border border-[var(--border-subtle)] bg-[var(--bg-main)]/30"></div>);
    }

    // Add cells for each day of the month
    for (let day = 1; day <= totalDays; day++) {
      const date = new Date(currentDate.getFullYear(), currentDate.getMonth(), day);
      const dateString = date.toDateString();
      const filesForDay = filesByDate[dateString] || [];
      const isToday = new Date().toDateString() === dateString;

      days.push(
        <div key={day} className={cn(
          "border border-[var(--border-subtle)] p-2 flex flex-col min-h-[120px] transition-colors",
          isToday ? "bg-[var(--brand-solid)]/5" : "bg-[var(--bg-card)]"
        )}>
          <span className={cn(
            "text-sm font-medium mb-1",
            isToday ? "text-[var(--brand-solid)]" : "text-[var(--text-secondary)]"
          )}>{day}</span>
          <div className="mt-1 space-y-1">
            {filesForDay.map((file) => (
              <div
                key={file.id}
                onClick={() => onFileClick(file.id)}
                className="group bg-[#FFFAF0] dark:bg-orange-950/20 border border-orange-100 dark:border-orange-900/30 p-1.5 rounded-lg cursor-pointer hover:border-[var(--brand-solid)] hover:shadow-sm transition-all"
              >
                <div className="flex items-center gap-1.5 min-w-0">
                  <FileAudio className="h-3 w-3 text-[#FF6D20] flex-shrink-0" />
                  <p className="text-[10px] text-gray-700 dark:text-gray-300 truncate font-medium group-hover:text-[#FF6D20]">
                    {file.title ? formatAudioFileTitle(file.title) : `File ${file.id.substring(0, 8)}`}
                  </p>
                </div>
              </div>
            ))}
          </div>
        </div>
      );
    }
    return days;
  };

  return (
    <div className="glass-card rounded-[var(--radius-card)] overflow-hidden border border-[var(--border-subtle)] shadow-[var(--shadow-float)]">
      <div className="flex justify-between items-center p-4 border-b border-[var(--border-subtle)] bg-[var(--bg-card)]">
        <h2 className="text-lg font-bold text-[var(--text-primary)]">
          {currentDate.toLocaleString("default", { month: "long", year: "numeric" })}
        </h2>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={handlePrevMonth} className="h-8 border-[var(--border-subtle)]">
            <ChevronLeft className="h-4 w-4 mr-1" />
            Prev
          </Button>
          <Button variant="outline" size="sm" onClick={handleNextMonth} className="h-8 border-[var(--border-subtle)]">
            Next
            <ChevronRight className="h-4 w-4 ml-1" />
          </Button>
        </div>
      </div>
      <div className="grid grid-cols-7 bg-[var(--border-subtle)]">
        {daysOfWeek.map((day) => (
          <div key={day} className="text-center text-xs font-bold text-[var(--text-tertiary)] py-3 bg-[var(--bg-main)] uppercase tracking-wider">
            {day}
          </div>
        ))}
        {renderDays()}
      </div>
    </div>
  );
};

