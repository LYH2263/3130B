import { useEffect, useState } from 'react';

export function CelebrationModal({ badges, onClose }) {
  const [visible, setVisible] = useState(false);
  const [showContent, setShowContent] = useState(false);

  useEffect(() => {
    if (badges && badges.length > 0) {
      setVisible(true);
      setTimeout(() => setShowContent(true), 100);
    }
  }, [badges]);

  if (!badges || badges.length === 0) {
    return null;
  }

  const handleClose = () => {
    setShowContent(false);
    setTimeout(() => {
      setVisible(false);
      if (onClose) onClose();
    }, 300);
  };

  if (!visible) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div
        className={`absolute inset-0 bg-black/50 backdrop-blur-sm transition-opacity duration-300 ${
          showContent ? 'opacity-100' : 'opacity-0'
        }`}
        onClick={handleClose}
      />

      <div
        className={`relative z-10 mx-4 w-full max-w-md transform transition-all duration-500 ${
          showContent
            ? 'translate-y-0 scale-100 opacity-100'
            : 'translate-y-8 scale-95 opacity-0'
        }`}
      >
        <div className="relative overflow-hidden rounded-3xl bg-gradient-to-br from-amber-400 via-orange-500 to-red-500 p-8 text-center shadow-2xl">
          <Confetti />

          <div className="relative z-10">
            <div className="mb-4 text-6xl">🎉</div>

            <h2 className="mb-2 text-2xl font-bold text-white">
              恭喜获得新徽章！
            </h2>
            <p className="mb-6 text-sm text-amber-100">
              继续保持学习打卡，解锁更多成就
            </p>

            <div className="mb-6 flex flex-wrap items-center justify-center gap-4">
              {badges.map((badge, index) => (
                <div
                  key={badge.badgeId}
                  className="animate-bounce"
                  style={{ animationDelay: `${index * 0.15}s` }}
                >
                  <div className="flex h-24 w-24 flex-col items-center justify-center rounded-2xl bg-white/90 shadow-lg">
                    <span className="text-4xl">{badge.icon}</span>
                    <span className="mt-1 text-xs font-bold text-amber-900">
                      {badge.name}
                    </span>
                  </div>
                </div>
              ))}
            </div>

            <button
              className="btn btn-secondary w-full"
              onClick={handleClose}
            >
              太棒了！
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

function Confetti() {
  const confettiPieces = Array.from({ length: 50 }, (_, i) => ({
    id: i,
    left: Math.random() * 100,
    delay: Math.random() * 2,
    duration: 2 + Math.random() * 2,
    color: ['#FFD700', '#FF6B6B', '#4ECDC4', '#45B7D1', '#96CEB4', '#FFEAA7'],
    colorIndex: Math.floor(Math.random() * 6),
    size: 6 + Math.random() * 8,
  }));

  return (
    <div className="pointer-events-none absolute inset-0 overflow-hidden">
      {confettiPieces.map((piece) => (
        <div
          key={piece.id}
          className="absolute rounded-sm"
          style={{
            left: `${piece.left}%`,
            top: '-20px',
            width: `${piece.size}px`,
            height: `${piece.size}px`,
            backgroundColor: piece.color[piece.colorIndex],
            animation: `confetti-fall ${piece.duration}s linear ${piece.delay}s infinite`,
          }}
        />
      ))}
      <style>{`
        @keyframes confetti-fall {
          0% {
            transform: translateY(0) rotate(0deg);
            opacity: 1;
          }
          100% {
            transform: translateY(400px) rotate(720deg);
            opacity: 0;
          }
        }
      `}</style>
    </div>
  );
}
