import { useState, useMemo } from "react";
import { ChevronLeft, ChevronRight, FileAudio, Clock, Calendar as CalendarIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { formatAudioFileTitle, parseTitleForDate, cn } from "@/lib/utils";
import { type AudioFile } from "@/features/transcription/hooks/useAudioFiles";

interface AudioFilesWeekCalendarProps {
  data: AudioFile[];
  onFileClick: (fileId: string) => void;
}

const HOUR_HEIGHT = 60;

interface HourSlot {
  hour: number;
  gapBefore?: number;
}

const ZigZag = () => (
  <svg width="100%" height="16" viewBox="0 0 1200 16" preserveAspectRatio="none" className="absolute -top-8 left-0 right-0 text-[var(--brand-solid)]/30 pointer-events-none overflow-visible">
    <path
      d="M0 8 L10 0 L30 16 L50 0 L70 16 L90 0 L110 16 L130 0 L150 16 L170 0 L190 16 L210 0 L230 16 L250 0 L270 16 L290 0 L310 16 L330 0 L350 16 L370 0 L390 16 L410 0 L430 16 L450 0 L470 16 L490 0 L510 16 L530 0 L550 16 L570 0 L590 16 L610 0 L630 16 L650 0 L670 16 L690 0 L710 16 L730 0 L750 16 L770 0 L790 16 L810 0 L830 16 L850 0 L870 16 L890 0 L910 16 L930 0 L950 16 L970 0 L990 16 L1010 0 L1030 16 L1050 0 L1070 16 L1090 0 L1110 16 L1130 0 L1150 16 L1170 0 L1190 16 L1200 8"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
  </svg>
);

export const AudioFilesWeekCalendar = ({ data, onFileClick }: AudioFilesWeekCalendarProps) => {
  const [baseDate, setBaseDate] = useState(new Date());

  const weeks = useMemo(() => {
    const result = [];
    for (let i = 0; i < 4; i++) {
      const date = new Date(baseDate);
      date.setDate(baseDate.getDate() - (i * 7));
      const day = date.getDay();
      const diff = date.getDate() - day;
      const startOfWeek = new Date(new Date(date.setDate(diff)).setHours(0, 0, 0, 0));
      result.push(startOfWeek);
    }
    return result;
  }, [baseDate]);

  const handlePrev4Weeks = () => {
    const newDate = new Date(baseDate);
    newDate.setDate(newDate.getDate() - 28);
    setBaseDate(newDate);
  };

  const handleNext4Weeks = () => {
    const newDate = new Date(baseDate);
    newDate.setDate(newDate.getDate() + 28);
    setBaseDate(newDate);
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

  const renderWeek = (startOfWeek: Date) => {
    const endOfWeek = new Date(startOfWeek);
    endOfWeek.setDate(startOfWeek.getDate() + 7);

    const weekFiles = filesWithDateTime.filter(f => f.dateTime >= startOfWeek && f.dateTime < endOfWeek);

    // Calculate which hours have activity
    const hourHasActivity = new Array(24).fill(false);
    weekFiles.forEach(file => {
      const startHour = file.dateTime.getHours();
      const endHour = new Date(file.dateTime.getTime() + file.duration * 60000).getHours();
      for (let h = startHour; h <= endHour; h++) {
        if (h < 24) hourHasActivity[h] = true;
      }
    });

    const slots: HourSlot[] = [];
    let currentGap: number[] = [];

    for (let h = 0; h < 24; h++) {
      if (hourHasActivity[h]) {
        if (currentGap.length > 0) {
          if (currentGap.length < 2) {
            currentGap.forEach(gh => slots.push({ hour: gh }));
          } else {
            // Process gap. If slots is NOT empty, it's not the first activity of the week.
            if (slots.length > 0) {
              slots.push({ hour: h, gapBefore: currentGap.length });
            } else {
              slots.push({ hour: h });
            }
            currentGap = [];
            continue;
          }
          currentGap = [];
        }
        slots.push({ hour: h });
      } else {
        currentGap.push(h);
      }
    }
    // Handle trailing gap if it's small, otherwise ignore
    if (currentGap.length > 0 && currentGap.length < 2) {
      currentGap.forEach(gh => slots.push({ hour: gh }));
    }

    const hourToOffset: Record<number, number> = {};
    slots.forEach((slot, index) => {
      hourToOffset[slot.hour] = index * HOUR_HEIGHT;
    });

    const totalHeight = slots.length * HOUR_HEIGHT;

    const days = [];
    for (let i = 0; i < 7; i++) {
      const date = new Date(startOfWeek);
      date.setDate(startOfWeek.getDate() + i);
      days.push(date);
    }

    return (
      <div key={startOfWeek.toISOString()} className="mb-12 last:mb-0">
        <div className="flex items-center gap-2 mb-4 px-4">
          <CalendarIcon className="h-5 w-5 text-[var(--brand-solid)]" />
          <h3 className="text-lg font-bold text-[var(--text-primary)]">
            Week of {startOfWeek.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })}
          </h3>
        </div>

        <div className="glass-card rounded-[var(--radius-card)] overflow-hidden border border-[var(--border-subtle)] bg-[var(--bg-card)] shadow-sm">
          <div className="grid grid-cols-[60px_1fr] border-b border-[var(--border-subtle)] bg-[var(--bg-main)]">
            <div className="border-r border-[var(--border-subtle)] flex items-center justify-center">
              <Clock className="h-4 w-4 text-[var(--text-tertiary)]" />
            </div>
            <div className="grid grid-cols-7">
              {days.map(day => (
                <div key={day.toISOString()} className={cn(
                  "text-center py-3 border-r border-[var(--border-subtle)] last:border-r-0",
                  new Date().toDateString() === day.toDateString() ? "bg-[var(--brand-solid)]/5" : ""
                )}>
                  <div className="text-[10px] font-bold text-[var(--text-tertiary)] uppercase tracking-wider">
                    {day.toLocaleString("default", { weekday: "short" })}
                  </div>
                  <div className={cn(
                    "text-sm font-bold",
                    new Date().toDateString() === day.toDateString() ? "text-[var(--brand-solid)]" : "text-[var(--text-primary)]"
                  )}>{day.getDate()}</div>
                </div>
              ))}
            </div>
          </div>

          <div className="grid grid-cols-[60px_1fr]" style={{ height: `${totalHeight}px` }}>
            <div className="border-r border-[var(--border-subtle)] bg-[var(--bg-main)]/50">
              {slots.map((slot, idx) => {
                const hour = slot.hour;
                const isGapBefore = !!slot.gapBefore;
                return (
                  <div key={hour} className={cn(
                    "h-[60px] text-[10px] font-medium text-[var(--text-tertiary)] text-right pr-2 pt-1 border-b border-[var(--border-subtle)]/30 relative",
                    isGapBefore && "mt-0"
                  )}>
                    {isGapBefore && (
                      <div className="absolute -top-4 left-0 z-20">
                        <div className="bg-[var(--bg-card)]/90 backdrop-blur-sm px-1 py-0.5 rounded-r-md border border-l-0 border-[var(--border-subtle)] shadow-sm">
                          <span className="text-[8px] font-bold text-[var(--brand-solid)] uppercase whitespace-nowrap">
                            {slot.gapBefore}h skipped
                          </span>
                        </div>
                      </div>
                    )}
                    {hour === 0 ? '12 AM' : hour < 12 ? `${hour} AM` : hour === 12 ? '12 PM' : `${hour - 12} PM`}
                  </div>
                );
              })}
            </div>

            <div className="grid grid-cols-7 relative">
              {days.map(day => (
                <div key={day.toISOString()} className="relative border-r border-[var(--border-subtle)] last:border-r-0">
                  {slots.map((slot, idx) => (
                    <div key={slot.hour} className="h-[60px] border-b border-[var(--border-subtle)]/30 last:border-b-0 relative">
                      {slot.gapBefore && <ZigZag />}
                    </div>
                  ))}
                  {/* Events for this day */}
                  {weekFiles
                    .filter(f => f.dateTime.toDateString() === day.toDateString())
                    .map(event => {
                      const startHour = event.dateTime.getHours();
                      if (hourToOffset[startHour] === undefined) return null;

                      const top = hourToOffset[startHour] + (event.dateTime.getMinutes() / 60) * HOUR_HEIGHT;
                      const height = (event.duration / 60) * HOUR_HEIGHT;

                      return (
                        <div
                          key={event.id}
                          className="absolute left-1 right-1 p-1.5 bg-[#FFFAF0] dark:bg-orange-950/20 border border-orange-100 dark:border-orange-900/30 rounded-lg cursor-pointer hover:border-[var(--brand-solid)] hover:shadow-md transition-all z-10 overflow-hidden group shadow-sm"
                          style={{ top: `${top}px`, height: `${height}px`, minHeight: '24px' }}
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
                    })}
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    );
  };


  return (
    <div className="space-y-8 pb-12">
      <div className="flex justify-between items-center bg-[var(--bg-card)] p-4 rounded-xl border border-[var(--border-subtle)] shadow-sm">
        <div className="flex flex-col">
          <span className="text-sm font-medium text-[var(--text-tertiary)] uppercase tracking-wider">Viewing Period</span>
          <h2 className="text-xl font-bold text-[var(--text-primary)]">
            Last 4 Weeks
          </h2>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={handlePrev4Weeks} className="h-10 border-[var(--border-subtle)] px-4">
            <ChevronLeft className="h-4 w-4 mr-2" />
            Previous 4 Weeks
          </Button>
          <Button variant="outline" size="sm" onClick={handleNext4Weeks} className="h-10 border-[var(--border-subtle)] px-4">
            Next 4 Weeks
            <ChevronRight className="h-4 w-4 ml-2" />
          </Button>
        </div>
      </div>

      <div className="space-y-0">
        {weeks.map(weekStart => renderWeek(weekStart))}
      </div>
    </div>
  );
};


