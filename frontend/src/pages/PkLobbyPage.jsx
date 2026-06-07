import { useState } from 'react';
import { toast } from 'react-hot-toast';
import { createPkRoom, joinPkRoom } from '../api/client';

export function PkLobbyPage({ user, token, onLogout, onEnterRoom }) {
  const [joinCode, setJoinCode] = useState('');
  const [creating, setCreating] = useState(false);
  const [joining, setJoining] = useState(false);
  const [questionCount, setQuestionCount] = useState(10);
  const [timePerQuestion, setTimePerQuestion] = useState(15);

  const handleCreateRoom = async () => {
    try {
      setCreating(true);
      const room = await createPkRoom(
        { questionCount, timePerQuestion },
        token
      );
      toast.success('房间创建成功！');
      onEnterRoom(room);
    } catch (error) {
      toast.error(error.message || '创建房间失败');
    } finally {
      setCreating(false);
    }
  };

  const handleJoinRoom = async () => {
    if (!joinCode || joinCode.length !== 6) {
      toast.error('请输入6位房间码');
      return;
    }
    try {
      setJoining(true);
      const room = await joinPkRoom(joinCode.toUpperCase(), token);
      toast.success('加入房间成功！');
      onEnterRoom(room);
    } catch (error) {
      toast.error(error.message || '加入房间失败');
    } finally {
      setJoining(false);
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-indigo-900 via-purple-900 to-pink-800 px-4 py-8">
      <div className="mx-auto max-w-4xl">
        <header className="mb-8 flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold text-white">🎮 双人 PK 对战</h1>
            <p className="mt-1 text-purple-200">实时答题对战，看看谁更厉害！</p>
          </div>
          <button className="btn btn-ghost text-white hover:bg-white/10" onClick={onLogout}>
            返回
          </button>
        </header>

        <div className="grid gap-6 md:grid-cols-2">
          <div className="rounded-3xl bg-white/10 backdrop-blur-lg p-6 border border-white/20 shadow-xl">
            <div className="mb-4 flex items-center gap-3">
              <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-emerald-500/20 text-2xl">
                🏠
              </div>
              <div>
                <h2 className="text-xl font-bold text-white">创建房间</h2>
                <p className="text-sm text-purple-200">邀请好友来挑战你</p>
              </div>
            </div>

            <div className="space-y-4">
              <div>
                <label className="mb-1 block text-sm font-medium text-purple-100">题目数量</label>
                <select
                  className="select select-bordered w-full bg-white/10 text-white border-white/30"
                  value={questionCount}
                  onChange={(e) => setQuestionCount(Number(e.target.value))}
                >
                  <option value={5}>5 题（快速局）</option>
                  <option value={10}>10 题（标准局）</option>
                  <option value={15}>15 题（耐力局）</option>
                  <option value={20}>20 题（马拉松）</option>
                </select>
              </div>

              <div>
                <label className="mb-1 block text-sm font-medium text-purple-100">每题时间</label>
                <select
                  className="select select-bordered w-full bg-white/10 text-white border-white/30"
                  value={timePerQuestion}
                  onChange={(e) => setTimePerQuestion(Number(e.target.value))}
                >
                  <option value={10}>10 秒（快速）</option>
                  <option value={15}>15 秒（标准）</option>
                  <option value={20}>20 秒（从容）</option>
                  <option value={30}>30 秒（思考）</option>
                </select>
              </div>

              <button
                className="btn btn-primary w-full bg-gradient-to-r from-emerald-500 to-teal-500 border-0 text-white font-bold"
                onClick={handleCreateRoom}
                disabled={creating}
              >
                {creating ? '创建中...' : '创建房间'}
              </button>
            </div>
          </div>

          <div className="rounded-3xl bg-white/10 backdrop-blur-lg p-6 border border-white/20 shadow-xl">
            <div className="mb-4 flex items-center gap-3">
              <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-blue-500/20 text-2xl">
                🚀
              </div>
              <div>
                <h2 className="text-xl font-bold text-white">加入房间</h2>
                <p className="text-sm text-purple-200">输入房间码加入对战</p>
              </div>
            </div>

            <div className="space-y-4">
              <div>
                <label className="mb-1 block text-sm font-medium text-purple-100">房间码</label>
                <input
                  type="text"
                  className="input input-bordered w-full bg-white/10 text-white border-white/30 uppercase tracking-widest text-center text-2xl font-bold"
                  placeholder="ABC123"
                  maxLength={6}
                  value={joinCode}
                  onChange={(e) => setJoinCode(e.target.value.toUpperCase())}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      handleJoinRoom();
                    }
                  }}
                />
              </div>

              <button
                className="btn btn-info w-full bg-gradient-to-r from-blue-500 to-indigo-500 border-0 text-white font-bold"
                onClick={handleJoinRoom}
                disabled={joining}
              >
                {joining ? '加入中...' : '加入房间'}
              </button>

              <div className="pt-2 text-center text-xs text-purple-300">
                提示：房间码不区分大小写
              </div>
            </div>
          </div>
        </div>

        <div className="mt-8 rounded-2xl bg-white/5 backdrop-blur p-5 border border-white/10">
          <h3 className="mb-3 font-bold text-white">📖 游戏规则</h3>
          <ul className="space-y-2 text-sm text-purple-200">
            <li>• 双方同时作答同一道题，答对越快得分越高</li>
            <li>• 3 秒内答对得 100 分，5 秒内 80 分，8 秒内 60 分，之后 40 分</li>
            <li>• 答错不扣分</li>
            <li>• 对手离场超过 5 秒判负，你将直接获胜</li>
            <li>• 全部题目答完后，得分高者获胜</li>
          </ul>
        </div>
      </div>
    </div>
  );
}
