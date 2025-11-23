import { useState, useMemo } from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";

interface AudioFile {
  id: string;
  title?: string;
  status: "uploaded" | "pending" | "processing" | "completed" | "failed";
  created_at: string;
}

interface AudioFilesCalendarProps {
  data: AudioFile[];
  onFileClick: (fileId: string) => void;
}

const daysOfWeek = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

export const AudioFilesCalendar = ({ data, onFileClick }: AudioFilesCalendarProps) => {
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
      const date = new Date(file.created_at).toDateString();
      if (!acc[date]) {
        acc[date] = [];
      }
      acc[date].push(file);
      return acc;
    }, {} as Record<string, AudioFile[]>);
  }, [data]);

  const renderDays = () => {
    const days = [];
    // Add empty cells for days before the start of the month
    for (let i = 0; i < startingDay; i++) {
      days.push(<div key={`empty-${i}`} className="border border-gray-200 dark:border-gray-700"></div>);
    }

    // Add cells for each day of the month
    for (let day = 1; day <= totalDays; day++) {
      const date = new Date(currentDate.getFullYear(), currentDate.getMonth(), day);
      const dateString = date.toDateString();
      const filesForDay = filesByDate[dateString] || [];

      days.push(
        <div key={day} className="border border-gray-200 dark:border-gray-700 p-2 flex flex-col">
          <span className="font-medium text-gray-900 dark:text-gray-100">{day}</span>
          <div className="mt-1 space-y-1">
            {filesForDay.map((file) => (
              <div
                key={file.id}
                onClick={() => onFileClick(file.id)}
                className="bg-blue-100 dark:bg-blue-900/50 p-1 rounded-md cursor-pointer hover:bg-blue-200 dark:hover:bg-blue-800/50"
              >
                <p className="text-xs text-blue-800 dark:text-blue-200 truncate">{file.title || `File ${file.id}`}</p>
              </div>
            ))}
          </div>
        </div>
      );
    }
    return days;
  };

  return (
    <div className="p-4 bg-white dark:bg-gray-800 rounded-lg">
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-50">
          {currentDate.toLocaleString("default", { month: "long", year: "numeric" })}
        </h2>
        <div className="flex space-x-2">
          <Button variant="outline" size="sm" onClick={handlePrevMonth}>
            <ChevronLeft className="h-4 w-4" />
            Prev
          </Button>
          <Button variant="outline" size="sm" onClick={handleNextMonth}>
            Next
            <ChevronRight className="h-4 w-4" />
          </Button>
        </div>
      </div>
      <div className="grid grid-cols-7 gap-px bg-gray-200 dark:bg-gray-700">
        {daysOfWeek.map((day) => (
          <div key={day} className="text-center font-medium text-gray-600 dark:text-gray-300 py-2 bg-gray-50 dark:bg-gray-700/50">
            {day}
          </div>
        ))}
        {renderDays()}
      </div>
    </div>
  );
};
