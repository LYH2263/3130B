import { useEffect, useRef, useState, useCallback } from 'react';
import { toast } from 'react-hot-toast';

export function PkGamePage({ room: initialRoom, user, token, onBack }) {
  const [gameState, setGameState] = useState('waiting');
  const [room, setRoom] = useState(initialRoom);
  const [isPlayerA, setIsPlayerA] = useState(false);
  const [currentQuestion, setCurrentQuestion] = useState(null);
  const [selectedOption, setSelectedOption] = useState(null);
  const [timeLeft, setTimeLeft] = useState(0);
  const [scoreA, setScoreA] = useState(0);
  const [scoreB, setScoreB] = useState(0);
  const [roundResult, setRoundResult] = useState(null);
  const [showResultAnimation, setShowResultAnimation] = useState(false);
  const [finalResult, setFinalResult] = useState(null);
  const [roundHistory, setRoundHistory] = useState([]);
  const [opponentName, setOpponentName] = useState('');
  const [myName, setMyName] = useState('');
  const [showReplay, setShowReplay] = useState(false);

  const wsRef = useRef(null);
  const timerRef = useRef(null);
  const roundStartRef = useRef(0);

  const myScore = isPlayerA ? scoreA : scoreB;
  const opponentScore = isPlayerA ? scoreB : scoreA;

  const connectWebSocket = useCallback(() => {
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsBase = import.meta.env.VITE_WS_BASE || `${wsProtocol}//${window.location.host}`;
    const wsUrl = `${wsBase}/api/pk/ws/${initialRoom.roomCode}?token=${encodeURIComponent(token)}`;
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      console.log('PK WebSocket connected');
    };

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        handleMessage(msg);
      } catch (e) {
        console.error('Failed to parse WS message', e);
      }
    };

    ws.onclose = () => {
      console.log('PK WebSocket closed');
    };

    ws.onerror = (error) => {
      console.error('PK WebSocket error', error);
    };
  }, [initialRoom.roomCode, token]);

  const handleMessage = (msg) => {
    switch (msg.type) {
      case 'welcome':
        setIsPlayerA(msg.data.isPlayerA);
        if (msg.data.isPlayerA) {
          setMyName(initialRoom.playerAName || user.username);
          setOpponentName(initialRoom.playerBName || '');
        } else {
          setMyName(initialRoom.playerBName || user.username);
          setOpponentName(initialRoom.playerAName || '');
        }
        setScoreA(msg.data.scoreA || 0);
        setScoreB(msg.data.scoreB || 0);
        break;

      case 'room_info':
        if (msg.data.playerA && msg.data.playerB) {
          if (isPlayerA) {
            setOpponentName(msg.data.playerB.username);
          } else {
            setOpponentName(msg.data.playerA.username);
          }
        }
        setScoreA(msg.data.scoreA || 0);
        setScoreB(msg.data.scoreB || 0);
        break;

      case 'game_start':
        setGameState('countdown');
        setOpponentName(isPlayerA ? msg.data.playerBName : msg.data.playerAName);
        break;

      case 'round_start':
        handleRoundStart(msg.data);
        break;

      case 'round_result':
        handleRoundResult(msg.data);
        break;

      case 'game_over':
        handleGameOver(msg.data);
        break;

      case 'player_leave':
        toast.error(msg.data.reason || '对手离开了');
        break;

      case 'error':
        toast.error(msg.data || '发生错误');
        break;
    }
  };

  const handleRoundStart = (data) => {
    setGameState('playing');
    setCurrentQuestion(data);
    setSelectedOption(null);
    setRoundResult(null);
    setShowResultAnimation(false);
    setTimeLeft(data.timeLimitSec);
    roundStartRef.current = data.startAt || Date.now();

    if (timerRef.current) {
      clearInterval(timerRef.current);
    }

    const startTime = data.startAt || Date.now();
    const totalMs = data.timeLimitSec * 1000;

    timerRef.current = setInterval(() => {
      const elapsed = Date.now() - startTime;
      const remaining = Math.max(0, Math.ceil((totalMs - elapsed) / 1000));
      setTimeLeft(remaining);
      if (remaining <= 0) {
        clearInterval(timerRef.current);
      }
    }, 100);
  };

  const handleRoundResult = (data) => {
    if (timerRef.current) {
      clearInterval(timerRef.current);
    }

    setScoreA(data.scoreA);
    setScoreB(data.scoreB);
    setRoundResult(data);
    setShowResultAnimation(true);
    setGameState('result');

    setRoundHistory((prev) => [
      ...prev,
      {
        roundIndex: data.roundIndex,
        questionId: data.questionId,
        myCorrect: isPlayerA ? data.playerACorrect : data.playerBCorrect,
        opponentCorrect: isPlayerA ? data.playerBCorrect : data.playerACorrect,
        myTime: isPlayerA ? data.playerATimeMs : data.playerBTimeMs,
        opponentTime: isPlayerA ? data.playerBTimeMs : data.playerATimeMs,
        correctOptionId: data.correctOptionId,
      },
    ]);
  };

  const handleGameOver = (data) => {
    if (timerRef.current) {
      clearInterval(timerRef.current);
    }
    setFinalResult(data);
    setGameState('finished');
  };

  const handleSelectOption = (optionId) => {
    if (gameState !== 'playing' || selectedOption !== null) return;
    setSelectedOption(optionId);

    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(
        JSON.stringify({
          type: 'answer',
          data: {
            questionId: currentQuestion.questionId,
            optionId,
          },
        })
      );
    }
  };

  const handleStartGame = () => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ type: 'game_start', data: {} }));
    }
  };

  const copyRoomCode = () => {
    navigator.clipboard.writeText(room.roomCode).then(() => {
      toast.success('房间码已复制');
    });
  };

  const shareRoom = () => {
    const text = `来和我 PK 答题吧！房间码：${room.roomCode}`;
    if (navigator.share) {
      navigator.share({ title: 'PK答题对战', text });
    } else {
      navigator.clipboard.writeText(text).then(() => {
        toast.success('分享信息已复制');
      });
    }
  };

  useEffect(() => {
    connectWebSocket();
    return () => {
      if (wsRef.current) {
        wsRef.current.close();
      }
      if (timerRef.current) {
        clearInterval(timerRef.current);
      }
    };
  }, [connectWebSocket]);

  const isRoomOwner = isPlayerA;
  const opponentReady = !!opponentName;

  const getTimeColor = () => {
    if (timeLeft <= 3) return 'text-red-500';
    if (timeLeft <= 5) return 'text-orange-500';
    return 'text-emerald-500';
  };

  const getOptionClass = (option) => {
    const base = 'flex items-center gap-3 rounded-2xl border-2 px-4 py-3 transition-all cursor-pointer';

    if (gameState === 'result' && roundResult) {
      if (option.id === roundResult.correctOptionId) {
        return `${base} border-emerald-500 bg-emerald-500/20`;
      }
      if (option.id === selectedOption && option.id !== roundResult.correctOptionId) {
        return `${base} border-red-500 bg-red-500/20`;
      }
      return `${base} border-white/10 opacity-50`;
    }

    if (selectedOption === option.id) {
      return `${base} border-blue-500 bg-blue-500/20`;
    }

    return `${base} border-white/20 hover:border-blue-400 hover:bg-white/5`;
  };

  const renderWaitingRoom = () => (
    <div className="min-h-screen bg-gradient-to-br from-indigo-900 via-purple-900 to-pink-800 px-4 py-8">
      <div className="mx-auto max-w-2xl">
        <button className="btn btn-ghost text-white hover:bg-white/10 mb-6" onClick={onBack}>
          ← 返回大厅
        </button>

        <div className="rounded-3xl bg-white/10 backdrop-blur-lg p-8 border border-white/20 shadow-xl text-center">
          <h2 className="text-2xl font-bold text-white mb-2">
            {isRoomOwner ? '房间已创建' : '已加入房间'}
          </h2>
          <p className="text-purple-200 mb-6">等待对手加入...</p>

          <div className="mb-6">
            <p className="text-sm text-purple-200 mb-2">房间码</p>
            <div className="flex items-center justify-center gap-3">
              <span className="text-5xl font-bold text-white tracking-[0.5em] font-mono">
                {room.roomCode}
              </span>
              <button
                className="btn btn-square btn-outline border-white/30 text-white hover:bg-white/10"
                onClick={copyRoomCode}
              >
                📋
              </button>
            </div>
          </div>

          <button
            className="btn btn-outline border-white/30 text-white hover:bg-white/10 mb-8"
            onClick={shareRoom}
          >
            📤 分享房间
          </button>

          <div className="flex justify-around mb-8">
            <div className="text-center">
              <div className="w-20 h-20 mx-auto rounded-full bg-gradient-to-br from-blue-500 to-cyan-400 flex items-center justify-center text-3xl font-bold text-white mb-2 shadow-lg">
                {myName?.[0]?.toUpperCase() || '?'}
              </div>
              <p className="text-white font-medium">{myName || '你'}</p>
              <p className="text-xs text-purple-300">{isRoomOwner ? '房主' : '玩家'}</p>
            </div>

            <div className="flex items-center">
              <span className="text-4xl text-white/50">VS</span>
            </div>

            <div className="text-center">
              <div className={`w-20 h-20 mx-auto rounded-full flex items-center justify-center text-3xl font-bold mb-2 shadow-lg ${
                opponentReady
                  ? 'bg-gradient-to-br from-pink-500 to-rose-400 text-white'
                  : 'bg-white/10 text-white/30 border-2 border-dashed border-white/20'
              }`}>
                {opponentReady ? opponentName?.[0]?.toUpperCase() : '?'}
              </div>
              <p className={`font-medium ${opponentReady ? 'text-white' : 'text-white/50'}`}>
                {opponentReady ? opponentName : '等待中...'}
              </p>
              <p className="text-xs text-purple-300">
                {opponentReady ? '对手' : '等待加入'}
              </p>
            </div>
          </div>

          {isRoomOwner && (
            <button
              className={`btn w-full text-white font-bold ${
                opponentReady
                  ? 'bg-gradient-to-r from-emerald-500 to-teal-500 hover:from-emerald-600 hover:to-teal-600'
                  : 'bg-white/20 cursor-not-allowed'
              }`}
              onClick={handleStartGame}
              disabled={!opponentReady}
            >
              {opponentReady ? '🎮 开始对战' : '等待对手加入...'}
            </button>
          )}

          {!isRoomOwner && (
            <div className="text-center text-purple-200 text-sm">
              等待房主开始游戏...
            </div>
          )}
        </div>

        <div className="mt-6 rounded-2xl bg-white/5 backdrop-blur p-4 border border-white/10">
          <h3 className="text-white font-medium mb-2">🎯 本场设置</h3>
          <div className="flex gap-6 text-sm text-purple-200">
            <span>题目数量：{room.questionCount} 题</span>
            <span>每题时间：{room.timePerQuestion} 秒</span>
          </div>
        </div>
      </div>
    </div>
  );

  const renderPlaying = () => (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 via-indigo-950 to-slate-900 px-4 py-6">
      <div className="mx-auto max-w-2xl">
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-center gap-3">
            <div className="w-12 h-12 rounded-full bg-gradient-to-br from-blue-500 to-cyan-400 flex items-center justify-center text-lg font-bold text-white">
              {myName?.[0]?.toUpperCase()}
            </div>
            <div>
              <p className="text-white font-medium text-sm">{myName}</p>
              <p className="text-blue-400 font-bold">{myScore} 分</p>
            </div>
          </div>

          <div className="text-center">
            <p className="text-xs text-slate-400">第 {currentQuestion?.roundIndex + 1} / {room.questionCount} 题</p>
            <p className={`text-4xl font-bold font-mono ${getTimeColor()}`}>
              {timeLeft}
            </p>
          </div>

          <div className="flex items-center gap-3">
            <div className="text-right">
              <p className="text-white font-medium text-sm">{opponentName}</p>
              <p className="text-pink-400 font-bold">{opponentScore} 分</p>
            </div>
            <div className="w-12 h-12 rounded-full bg-gradient-to-br from-pink-500 to-rose-400 flex items-center justify-center text-lg font-bold text-white">
              {opponentName?.[0]?.toUpperCase()}
            </div>
          </div>
        </div>

        <div className="h-2 bg-white/10 rounded-full mb-6 overflow-hidden">
          <div
            className="h-full bg-gradient-to-r from-blue-500 to-purple-500 transition-all duration-300"
            style={{ width: `${((currentQuestion?.roundIndex + 1) / room.questionCount) * 100}%` }}
          />
        </div>

        <div className="rounded-3xl bg-white/10 backdrop-blur-lg p-6 border border-white/20 shadow-xl mb-6">
          <h2 className="text-xl font-bold text-white mb-6 leading-relaxed">
            {currentQuestion?.title}
          </h2>

          <div className="space-y-3">
            {currentQuestion?.options?.map((option, idx) => (
              <div
                key={option.id}
                className={getOptionClass(option)}
                onClick={() => handleSelectOption(option.id)}
              >
                <span className="w-8 h-8 rounded-full bg-white/10 flex items-center justify-center text-white font-bold text-sm flex-shrink-0">
                  {String.fromCharCode(65 + idx)}
                </span>
                <span className="text-white">{option.content}</span>
                {gameState === 'result' && option.id === roundResult?.correctOptionId && (
                  <span className="ml-auto text-emerald-400 text-xl">✓</span>
                )}
                {gameState === 'result' && option.id === selectedOption && option.id !== roundResult?.correctOptionId && (
                  <span className="ml-auto text-red-400 text-xl">✗</span>
                )}
              </div>
            ))}
          </div>
        </div>

        {showResultAnimation && roundResult && (
          <div className="fixed inset-0 pointer-events-none flex items-center justify-center z-50">
            <div className="animate-bounce text-center">
              <div className="text-8xl mb-4">
                {(isPlayerA ? roundResult.playerACorrect : roundResult.playerBCorrect) ? '✅' : '❌'}
              </div>
              <p className={`text-2xl font-bold ${
                (isPlayerA ? roundResult.playerACorrect : roundResult.playerBCorrect)
                  ? 'text-emerald-400'
                  : 'text-red-400'
              }`}>
                {(isPlayerA ? roundResult.playerACorrect : roundResult.playerBCorrect)
                  ? '答对了！'
                  : '答错了'}
              </p>
            </div>
          </div>
        )}

        {gameState === 'result' && roundResult && (
          <div className="rounded-2xl bg-white/5 backdrop-blur p-4 border border-white/10">
            <h3 className="text-white font-medium mb-3">本轮结果</h3>
            <div className="grid grid-cols-2 gap-4">
              <div className="text-center">
                <p className="text-xs text-slate-400">你的用时</p>
                <p className="text-lg font-bold text-white">
                  {(isPlayerA ? roundResult.playerATimeMs : roundResult.playerBTimeMs) / 1000}s
                </p>
              </div>
              <div className="text-center">
                <p className="text-xs text-slate-400">对手用时</p>
                <p className="text-lg font-bold text-white">
                  {(isPlayerA ? roundResult.playerBTimeMs : roundResult.playerATimeMs) / 1000}s
                </p>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );

  const renderFinished = () => (
    <div className="min-h-screen bg-gradient-to-br from-indigo-900 via-purple-900 to-pink-800 px-4 py-8">
      <div className="mx-auto max-w-2xl">
        <div className="text-center mb-8">
          <div className="text-7xl mb-4 animate-bounce">
            {finalResult?.isDraw ? '🤝' : (finalResult?.winnerName === myName ? '🏆' : '😢')}
          </div>
          <h1 className="text-4xl font-bold text-white mb-2">
            {finalResult?.isDraw
              ? '平局！'
              : finalResult?.winnerName === myName
              ? '你赢了！'
              : '你输了...'}
          </h1>
          <p className="text-purple-200">
            {finalResult?.isDraw ? '势均力敌，再来一局？' : finalResult?.winnerName === myName ? '太棒了，继续保持！' : '别灰心，下次再战！'}
          </p>
        </div>

        <div className="rounded-3xl bg-white/10 backdrop-blur-lg p-8 border border-white/20 shadow-xl mb-6">
          <div className="flex items-center justify-around">
            <div className="text-center">
              <div className={`w-24 h-24 mx-auto rounded-full flex items-center justify-center text-4xl font-bold mb-3 ${
                finalResult?.winnerName === myName
                  ? 'bg-gradient-to-br from-yellow-400 to-orange-500 text-white'
                  : 'bg-gradient-to-br from-blue-500 to-cyan-400 text-white'
              }`}>
                {myName?.[0]?.toUpperCase()}
              </div>
              <p className="text-white font-bold text-lg">{myName}</p>
              <p className="text-3xl font-bold text-white mt-2">{myScore}</p>
              <p className="text-purple-300 text-sm">分</p>
              {finalResult?.winnerName === myName && (
                <span className="badge badge-warning mt-2">👑 胜者</span>
              )}
            </div>

            <div className="text-5xl text-white/30 font-bold">VS</div>

            <div className="text-center">
              <div className={`w-24 h-24 mx-auto rounded-full flex items-center justify-center text-4xl font-bold mb-3 ${
                finalResult?.winnerName === opponentName
                  ? 'bg-gradient-to-br from-yellow-400 to-orange-500 text-white'
                  : 'bg-gradient-to-br from-pink-500 to-rose-400 text-white'
              }`}>
                {opponentName?.[0]?.toUpperCase()}
              </div>
              <p className="text-white font-bold text-lg">{opponentName}</p>
              <p className="text-3xl font-bold text-white mt-2">{opponentScore}</p>
              <p className="text-purple-300 text-sm">分</p>
              {finalResult?.winnerName === opponentName && (
                <span className="badge badge-warning mt-2">👑 胜者</span>
              )}
            </div>
          </div>
        </div>

        <div className="rounded-2xl bg-white/5 backdrop-blur p-5 border border-white/10 mb-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-white font-bold">📊 答题回顾</h3>
            <button
              className="btn btn-sm btn-outline border-white/30 text-white"
              onClick={() => setShowReplay(!showReplay)}
            >
              {showReplay ? '收起' : '展开'}
            </button>
          </div>

          {showReplay && (
            <div className="space-y-2 max-h-80 overflow-y-auto">
              {roundHistory.map((round, idx) => (
                <div
                  key={idx}
                  className="flex items-center justify-between p-3 rounded-xl bg-white/5"
                >
                  <span className="text-white/70 text-sm">第 {round.roundIndex + 1} 题</span>
                  <div className="flex items-center gap-4">
                    <span className={`text-sm ${round.myCorrect ? 'text-emerald-400' : 'text-red-400'}`}>
                      {round.myCorrect ? '✓ 对' : '✗ 错'}
                      {round.myTime ? ` (${(round.myTime / 1000).toFixed(1)}s)` : ''}
                    </span>
                    <span className="text-white/30">|</span>
                    <span className={`text-sm ${round.opponentCorrect ? 'text-emerald-400' : 'text-red-400'}`}>
                      {round.opponentCorrect ? '✓ 对' : '✗ 错'}
                      {round.opponentTime ? ` (${(round.opponentTime / 1000).toFixed(1)}s)` : ''}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          )}

          {!showReplay && (
            <div className="flex justify-around text-center">
              <div>
                <p className="text-2xl font-bold text-emerald-400">
                  {roundHistory.filter(r => r.myCorrect).length}
                </p>
                <p className="text-xs text-purple-300">你答对</p>
              </div>
              <div>
                <p className="text-2xl font-bold text-red-400">
                  {roundHistory.filter(r => !r.myCorrect).length}
                </p>
                <p className="text-xs text-purple-300">你答错</p>
              </div>
              <div>
                <p className="text-2xl font-bold text-pink-400">
                  {roundHistory.filter(r => r.opponentCorrect).length}
                </p>
                <p className="text-xs text-purple-300">对手答对</p>
              </div>
            </div>
          )}
        </div>

        <div className="flex gap-3">
          <button className="btn btn-primary flex-1 bg-gradient-to-r from-emerald-500 to-teal-500 border-0 text-white font-bold" onClick={onBack}>
            返回大厅
          </button>
          <button className="btn btn-outline flex-1 border-white/30 text-white hover:bg-white/10" onClick={() => window.location.reload()}>
            再来一局
          </button>
        </div>
      </div>
    </div>
  );

  if (gameState === 'waiting') {
    return renderWaitingRoom();
  }

  if (gameState === 'finished') {
    return renderFinished();
  }

  return renderPlaying();
}
