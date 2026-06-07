export function DifficultyDistributionChart({ distribution }) {
  if (!distribution) {
    return null;
  }

  const { easyCount, mediumCount, hardCount, noDataCount, total } = distribution;

  const data = [
    { label: '易', count: easyCount, color: 'bg-success', textColor: 'text-success', barClass: 'bg-success' },
    { label: '中', count: mediumCount, color: 'bg-warning', textColor: 'text-warning', barClass: 'bg-warning' },
    { label: '难', count: hardCount, color: 'bg-error', textColor: 'text-error', barClass: 'bg-error' },
    { label: '数据不足', count: noDataCount, color: 'bg-slate-300', textColor: 'text-slate-500', barClass: 'bg-slate-300' },
  ];

  const maxCount = Math.max(easyCount, mediumCount, hardCount, noDataCount, 1);

  const hasData = total > 0;

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-4 gap-3">
        {data.map((item) => {
          const percent = hasData ? ((item.count / total) * 100).toFixed(1) : 0;
          return (
            <div key={item.label} className="rounded-xl bg-slate-50 p-3 text-center">
              <div className={`text-2xl font-bold ${item.textColor}`}>{item.count}</div>
              <div className="text-xs text-slate-500">{item.label}</div>
              <div className="mt-1 text-xs text-slate-400">{percent}%</div>
            </div>
          );
        })}
      </div>

      <div className="space-y-2">
        {data.map((item) => {
          const heightPercent = hasData ? (item.count / maxCount) * 100 : 0;
          return (
            <div key={item.label} className="flex items-center gap-3">
              <div className="w-16 text-right text-sm text-slate-600">{item.label}</div>
              <div className="flex-1">
                <div className="h-6 w-full overflow-hidden rounded-full bg-slate-100">
                  <div
                    className={`h-full ${item.barClass} transition-all duration-500`}
                    style={{ width: `${heightPercent}%` }}
                  />
                </div>
              </div>
              <div className="w-16 text-sm font-medium text-slate-700">
                {item.count} 题
              </div>
            </div>
          );
        })}
      </div>

      <div className="pt-2 text-center text-xs text-slate-500">
        题库总量：<span className="font-semibold text-slate-700">{total}</span> 题
      </div>
    </div>
  );
}
