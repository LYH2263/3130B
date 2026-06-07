import { useEffect, useState } from 'react';
import { getUserBadges } from '../api/client';

export function BadgeWall({ token, refreshTrigger }) {
  const [badges, setBadges] = useState([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    loadBadges();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token, refreshTrigger]);

  const loadBadges = async () => {
    setLoading(true);
    try {
      const data = await getUserBadges(token);
      setBadges(data || []);
    } catch (error) {
      console.error('加载徽章失败', error);
    } finally {
      setLoading(false);
    }
  };

  const ownedCount = badges.filter((b) => b.owned).length;

  return (
    <div className="rounded-3xl border border-slate-200 bg-white p-5 shadow-card">
      <div className="mb-4 flex items-center justify-between">
        <h3 className="text-lg font-semibold text-slate-800">徽章墙</h3>
        <span className="text-sm text-slate-500">
          已获得 {ownedCount}/{badges.length}
        </span>
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-8">
          <div className="h-8 w-8 animate-spin rounded-full border-2 border-emerald-500 border-t-transparent" />
        </div>
      ) : (
        <div className="grid grid-cols-3 gap-3">
          {badges.map((badge) => (
            <div
              key={badge.badgeId}
              className={`group relative flex flex-col items-center gap-2 rounded-2xl border p-3 text-center transition-all ${
                badge.owned
                  ? 'border-amber-200 bg-gradient-to-br from-amber-50 to-orange-100 shadow-sm'
                  : 'border-slate-200 bg-slate-50 opacity-60'
              }`}
            >
              <div
                className={`flex h-14 w-14 items-center justify-center rounded-full text-3xl transition-transform ${
                  badge.owned ? 'animate-pulse scale-110' : 'grayscale'
                }`}
              >
                {badge.icon}
              </div>
              <div>
                <p
                  className={`text-xs font-semibold ${
                    badge.owned ? 'text-amber-900' : 'text-slate-500'
                  }`}
                >
                  {badge.name}
                </p>
                <p className="text-[10px] text-slate-500">{badge.description}</p>
              </div>
              {badge.owned && badge.awardedAt && (
                <p className="text-[10px] text-amber-600">
                  {badge.awardedAt.split(' ')[0]}
                </p>
              )}
              {!badge.owned && (
                <div className="absolute inset-0 flex items-center justify-center">
                  <span className="text-2xl opacity-30">🔒</span>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
