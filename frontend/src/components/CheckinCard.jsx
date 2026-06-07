import { useState } from 'react';
import { toast } from 'react-hot-toast';
import { manualCheckin } from '../api/client';

export function CheckinCard({ status, onCheckinSuccess, token }) {
  const [checkingIn, setCheckingIn] = useState(false);

  const handleManualCheckin = async () => {
    if (status?.todayCheckedIn) {
      toast('今日已打卡');
      return;
    }

    try {
      setCheckingIn(true);
      const result = await manualCheckin(
        { questionCount: 5, correctCount: 3 },
        token
      );
      if (result.checkedIn && onCheckinSuccess) {
        onCheckinSuccess(result);
      }
      if (result.isNewCheckin) {
        toast.success('打卡成功！');
      }
    } catch (error) {
      toast.error(error.message || '打卡失败');
    } finally {
      setCheckingIn(false);
    }
  };

  return (
    <div className="relative overflow-hidden rounded-3xl border border-slate-200 bg-gradient-to-br from-orange-50 via-amber-50 to-yellow-50 p-5 shadow-card">
      <div className="absolute -right-4 -top-4 text-8xl opacity-10">🔥</div>

      <div className="relative z-10">
        <div className="flex items-start justify-between">
          <div>
            <p className="text-xs font-semibold uppercase tracking-wide text-amber-700">
              学习打卡
            </p>
            <div className="mt-2 flex items-baseline gap-2">
              <span className="text-4xl font-bold text-amber-900">
                {status?.currentStreak || 0}
              </span>
              <span className="text-sm text-amber-700">天连续</span>
            </div>
          </div>

          <div
            className={`flex h-14 w-14 items-center justify-center rounded-2xl text-2xl shadow-md transition-transform ${
              status?.todayCheckedIn
                ? 'bg-gradient-to-br from-emerald-400 to-emerald-600 text-white'
                : 'bg-white text-slate-300'
            }`}
          >
            {status?.todayCheckedIn ? '✓' : '○'}
          </div>
        </div>

        <div className="mt-4 flex items-center justify-between">
          <div className="flex gap-4 text-xs text-amber-700">
            <div>
              <span className="font-semibold">历史最长：</span>
              <span>{status?.longestStreak || 0} 天</span>
            </div>
            <div>
              <span className="font-semibold">今日答题：</span>
              <span>{status?.questionCount || 0} 题</span>
            </div>
          </div>
        </div>

        <div className="mt-4">
          {status?.todayCheckedIn ? (
            <div className="flex items-center justify-center gap-2 rounded-xl bg-emerald-100 py-2.5 text-sm font-medium text-emerald-700">
              <span>✓</span>
              <span>今日已打卡</span>
            </div>
          ) : (
            <button
              className="btn btn-primary w-full"
              onClick={handleManualCheckin}
              disabled={checkingIn}
            >
              {checkingIn ? '打卡中...' : '立即打卡'}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
