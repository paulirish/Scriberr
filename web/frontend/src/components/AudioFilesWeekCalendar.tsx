import { useState, useMemo } from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";

interface AudioFile {
  id: string;
  title?: string;
  status: "uploaded" | "pending" | "processing" | "completed" | "failed";
  created_at: string;
}

interface AudioFilesWeekCalendarProps {
  data: AudioFile[];
  onFileClick: (fileId: string) => void;
}

const parseTitleForDate = (title: string): { date: Date; duration: number } | null => {
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


export const AudioFilesWeekCalendar = ({ data, onFileClick }: AudioFilesWeekCalendarProps) => {
  const [currentDate, setCurrentDate] = useState(new Date());

  const startOfWeek = useMemo(() => {
    const date = new Date(currentDate);
    const day = date.getDay();
    const diff = date.getDate() - day + (day === 0 ? -6 : 1); // adjust when day is sunday
    return new Date(date.setDate(diff));
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
    setCurrentDate(new Date(currentDate.setDate(currentDate.getDate() - 7)));
  };

  const handleNextWeek = () => {
    setCurrentDate(new Date(currentDate.setDate(currentDate.getDate() + 7)));
  };

  const filesWithDateTime = useMemo(() => {
    return data.map(file => {
      if (file.title) {
        const parsed = parseTitleForDate(file.title);
        if (parsed) {
          return { ...file, dateTime: parsed.date, duration: parsed.duration };
        }
      }
      return { ...file, dateTime: new Date(file.created_at), duration: 15 }; // Default 15 min duration
    });
  }, [data]);

  const renderEventsForDay = (day: Date) => {
    const dayEvents = filesWithDateTime.filter(file => file.dateTime.toDateString() === day.toDateString());
    return dayEvents.map(event => {
      const top = (event.dateTime.getHours() + event.dateTime.getMinutes() / 60) * 60; // 60px per hour
      const height = event.duration * 1; // 1px per minute
      return (
        <div
          key={event.id}
          className="absolute left-0 right-0 p-1 bg-blue-100 dark:bg-blue-900/50 rounded-md cursor-pointer hover:bg-blue-200 dark:hover:bg-blue-800/50"
          style={{ top: `${top}px`, height: `${height}px` }}
          onClick={() => onFileClick(event.id)}
        >
          <p className="text-xs text-blue-800 dark:text-blue-200 truncate">{event.title || `File ${event.id}`}</p>
        </div>
      );
    });
  };

  return (
    <div className="p-4 bg-white dark:bg-gray-800 rounded-lg">
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-50">
          {startOfWeek.toLocaleString("default", { month: "long", day: "numeric" })} - {new Date(startOfWeek.getTime() + 6 * 24 * 60 * 60 * 1000).toLocaleString("default", { month: "long", day: "numeric", year: "numeric" })}
        </h2>
        <div className="flex space-x-2">
          <Button variant="outline" size="sm" onClick={handlePrevWeek}>
            <ChevronLeft className="h-4 w-4" />
            Prev
          </Button>
          <Button variant="outline" size="sm" onClick={handleNextWeek}>
            Next
            <ChevronRight className="h-4 w-4" />
          </Button>
        </div>
      </div>
      <div className="grid grid-cols-7 border-t border-l border-gray-200 dark:border-gray-700">
        {days.map(day => (
          <div key={day.toISOString()} className="text-center font-medium text-gray-600 dark:text-gray-300 py-2 border-b border-r border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-700/50">
            {day.toLocaleString("default", { weekday: "short" })} {day.getDate()}
          </div>
        ))}
        {days.map(day => (
          <div key={day.toISOString()} className="relative border-r border-gray-200 dark:border-gray-700 h-[1440px]">
            {renderEventsForDay(day)}
          </div>
        ))}
      </div>
    </div>
  );
};
