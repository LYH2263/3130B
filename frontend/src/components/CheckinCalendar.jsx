import { useEffect, useState } from 'react';
import { getCheckinCalendar } from '../api/client';

const WEEKDAYS = ['日', '一', '二', '三', '四', '五', '六'];

function getHeatLevel(questionCount) {
  if (questionCount === 0) return 0;
  if (questionCount < 5) return 1;
  if (questionCount < 10) return 2;
  if (questionCount < 20) return 3;
  return 4;
}

function getHeatColor(level) {
  const colors = [
    'bg-slate-100',
    'bg-emerald-200',
    'bg-emerald-400',
    'bg-emerald-500',
    'bg-emerald-700',
  ];
  return colors[level] || colors[0];
}

export function CheckinCalendar({ token }) {
  const today = new Date();
  const [year, setYear] = useState(today.getFullYear());
  const [month, setMonth] = useState(today.getMonth() + 1);
  const [calendarData, setCalendarData] = useState(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    loadCalendar();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [year, month, token]);

  const loadCalendar = async () => {
    setLoading(true);
    try {
      const data = await getCheckinCalendar(year, month, token);
      setCalendarData(data);
    } catch (error) {
      console.error('加载日历失败', error);
    } finally {
      setLoading(false);
    }
  };

  const prevMonth = () => {
    if (month === 1) {
      setMonth(12);
      setYear(year - 1);
    } else {
      setMonth(month - 1);
    }
  };

  const nextMonth = () => {
    if (month === 12) {
      setMonth(1);
      setYear(year + 1);
    } else {
      setMonth(month + 1);
    }
  };

  const getFirstDayOfMonth = () => {
    const firstDay = new Date(year, month - 1, 1);
    return firstDay.getDay();
  };

  const getDaysInMonth = () => {
    const lastDay = new Date(year, month, 0);
    return lastDay.getDate();
  };

  const getDayData = (day) => {
    if (!calendarData?.days) return null;
    const dateStr = `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
    return calendarData.days.find((d) => d.date === dateStr) || null;
  };

  const isToday = (day) => {
    const now = new Date();
    return (
      year === now.getFullYear() &&
      month === now.getMonth() + 1 &&
      day === now.getDate()
    );
  };

  const daysInMonth = getDaysInMonth();
  const firstDayOfWeek = getFirstDayOfMonth();
  const totalCells = Math.ceil((firstDayOfWeek + daysInMonth) / 7) * 7;

  return (
    <div className="rounded-3xl border border-slate-200 bg-white p-5 shadow-card">
      <div className="mb-4 flex items-center justify-between">
        <h3 className="text-lg font-semibold text-slate-800">打卡日历</h3>
        <div className="flex items-center gap-2">
          <button
            className="btn btn-sm btn-ghost"
            onClick={prevMonth}
          >
            ‹
          </button>
          <span className="min-w-[100px] text-center text-sm font-medium text-slate-700">
            {year}年{month}月
          </span>
          <button
            className="btn btn-sm btn-ghost"
            onClick={nextMonth}
          >
            ›
          </button>
        </div>
      </div>

      <div className="grid grid-cols-7 gap-1">
        {WEEKDAYS.map((day) => (
          <div
            key={day}
            className="py-2 text-center text-xs font-medium text-slate-500"
          >
            {day}
          </div>
        ))}
      </div>

      <div className="grid grid-cols-7 gap-1">
        {Array.from({ length: totalCells }).map((_, index) => {
          const day = index - firstDayOfWeek + 1;
          const isValid = day > 0 && day <= daysInMonth;
          const dayData = isValid ? getDayData(day) : null;
          const heatLevel = dayData ? getHeatLevel(dayData.questionCount) : 0;
          const today = isToday(day);

          return (
            <div
              key={index}
              className={`relative aspect-square rounded-lg transition-all ${
                isValid
                  ? `${getHeatColor(heatLevel)} ${
                      today ? 'ring-2 ring-amber-400 ring-offset-1' : ''
                    }`
                  : 'bg-transparent'
              }`}
            >
              {isValid && (
                <div className="flex h-full flex-col items-center justify-center">
                  <span
                    className={`text-xs font-medium ${
                      dayData?.isCheckedIn
                        ? heatLevel >= 3
                          ? 'text-white'
                          : 'text-emerald-900'
                        : 'text-slate-600'
                    }`}
                  >
                    {day}
                  </span>
                  {dayData?.isCheckedIn && (
                    <span
                      className={`text-[10px] ${
                        heatLevel >= 3 ? 'text-emerald-100' : 'text-emerald-700'
                      }`}
                    >
                      {dayData.questionCount}题
                    </span>
                  )}
                </div>
              )}
            </div>
          );
        })}
      </div>

      <div className="mt-4 flex items-center justify-end gap-2 text-xs text-slate-500">
        <span>少</span>
        <div className="flex gap-1">
          {[0, 1, 2, 3, 4].map((level) => (
            <div
              key={level}
              className={`h-4 w-4 rounded ${getHeatColor(level)}`}
            />
          ))}
        </div>
        <span>多</span>
      </div>
    </div>
  );
}
