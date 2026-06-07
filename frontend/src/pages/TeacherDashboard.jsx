import { useEffect, useMemo, useState } from 'react';
import { toast } from 'react-hot-toast';

import { apiRequest } from '../api/client';
import { QuestionEditorModal } from '../components/QuestionEditorModal';
import { StatCard } from '../components/StatCard';
import { DifficultyDistributionChart } from '../components/DifficultyDistributionChart';
import { questionSchema } from '../utils/validators';
import {
  getDifficultyLevel,
  getDifficultyLabel,
  getDifficultyBadgeClass,
  formatDifficultyValue,
  getAbnormalTypes,
} from '../utils/difficultyUtils';

export function TeacherDashboard({ user, token, onLogout, onNavigateToSubjective, onNavigateToGrading, onNavigateToExam, onNavigateToReport, onNavigateToProctor, onNavigateToPaper }) {
  const [overview, setOverview] = useState(null);
  const [questions, setQuestions] = useState([]);
  const [stats, setStats] = useState([]);
  const [attempts, setAttempts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [savingQuestion, setSavingQuestion] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingQuestion, setEditingQuestion] = useState(null);
  const [uploading, setUploading] = useState(false);

  const [difficultyDistribution, setDifficultyDistribution] = useState(null);
  const [abnormalQuestions, setAbnormalQuestions] = useState([]);
  const [recalculating, setRecalculating] = useState(false);
  const [difficultyFilter, setDifficultyFilter] = useState('all');
  const [sortBy, setSortBy] = useState('default');
  const [activeTab, setActiveTab] = useState('all');

  const topStats = useMemo(() => stats.slice(0, 12), [stats]);

  const loadDashboard = async () => {
    setLoading(true);
    try {
      const [overviewData, questionData, statData, attemptData, distData, abnormalData] = await Promise.all([
        apiRequest('/teacher/overview', { token }),
        apiRequest('/teacher/question-stats/questions?pageSize=100', { token }),
        apiRequest('/teacher/class-stats', { token }),
        apiRequest('/teacher/attempts?limit=50', { token }),
        apiRequest('/teacher/question-stats/distribution', { token }),
        apiRequest('/teacher/question-stats/abnormal', { token }),
      ]);
      setOverview(overviewData);
      setQuestions(questionData.items || []);
      setStats(statData);
      setAttempts(attemptData);
      setDifficultyDistribution(distData);
      setAbnormalQuestions(abnormalData.items || []);
    } catch (error) {
      toast.error(error.message || '加载教师看板失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadDashboard();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  const handleSaveQuestion = async (payload, changeNote) => {
    try {
      questionSchema.parse(payload);
      setSavingQuestion(true);
      if (editingQuestion) {
        await apiRequest(`/teacher/questions/${editingQuestion.id}`, {
          method: 'PUT',
          token,
          body: { ...payload, changeNote },
        });
        toast.success('题目已更新');
      } else {
        await apiRequest('/teacher/questions', {
          method: 'POST',
          token,
          body: payload,
        });
        toast.success('题目已创建');
      }
      setModalOpen(false);
      setEditingQuestion(null);
      await loadDashboard();
    } catch (error) {
      toast.error(error?.issues?.[0]?.message || error.message || '保存题目失败');
    } finally {
      setSavingQuestion(false);
    }
  };

  const handleDeleteQuestion = async (questionId) => {
    if (!window.confirm('确认删除该题目？')) {
      return;
    }
    try {
      await apiRequest(`/teacher/questions/${questionId}`, {
        method: 'DELETE',
        token,
      });
      toast.success('题目已删除');
      await loadDashboard();
    } catch (error) {
      toast.error(error.message || '删除失败');
    }
  };

  const handleUpload = async (event) => {
    const file = event.target.files?.[0];
    if (!file) {
      return;
    }
    const formData = new FormData();
    formData.append('file', file);

    try {
      setUploading(true);
      const data = await apiRequest('/teacher/questions/upload', {
        method: 'POST',
        token,
        body: formData,
        isForm: true,
      });
      toast.success(`导入成功，新增 ${data.count || 0} 题`);
      await loadDashboard();
    } catch (error) {
      toast.error(error.message || '上传失败');
    } finally {
      setUploading(false);
      event.target.value = '';
    }
  };

  const handleRecalculate = async () => {
    if (!window.confirm('确认重新计算所有题目的难度系数？这可能需要一些时间。')) {
      return;
    }
    try {
      setRecalculating(true);
      const result = await apiRequest('/teacher/question-stats/recalculate', {
        method: 'POST',
        token,
      });
      toast.success(`重算完成，更新 ${result.updatedCount} 题，失败 ${result.failedCount} 题`);
      await loadDashboard();
    } catch (error) {
      toast.error(error.message || '重算失败');
    } finally {
      setRecalculating(false);
    }
  };

  const openCreateModal = () => {
    setEditingQuestion(null);
    setModalOpen(true);
  };

  const openEditModal = (question) => {
    setEditingQuestion(question);
    setModalOpen(true);
  };

  const filteredQuestions = useMemo(() => {
    let result = [...questions];

    if (activeTab === 'abnormal') {
      result = result.filter((q) => {
        if (!q.stats || !q.stats.hasEnoughData) return false;
        const types = getAbnormalTypes(q.stats);
        return types.length > 0;
      });
    } else if (difficultyFilter !== 'all') {
      result = result.filter((q) => {
        const level = getDifficultyLevel(q.stats?.difficulty, q.stats?.hasEnoughData);
        return level === difficultyFilter;
      });
    }

    if (sortBy === 'difficulty_asc') {
      result.sort((a, b) => {
        const aHas = a.stats?.hasEnoughData && a.stats?.difficulty != null;
        const bHas = b.stats?.hasEnoughData && b.stats?.difficulty != null;
        if (!aHas && !bHas) return 0;
        if (!aHas) return 1;
        if (!bHas) return -1;
        return b.stats.difficulty - a.stats.difficulty;
      });
    } else if (sortBy === 'difficulty_desc') {
      result.sort((a, b) => {
        const aHas = a.stats?.hasEnoughData && a.stats?.difficulty != null;
        const bHas = b.stats?.hasEnoughData && b.stats?.difficulty != null;
        if (!aHas && !bHas) return 0;
        if (!aHas) return 1;
        if (!bHas) return -1;
        return a.stats.difficulty - b.stats.difficulty;
      });
    } else if (sortBy === 'attempts_desc') {
      result.sort((a, b) => (b.stats?.totalAttempts || 0) - (a.stats?.totalAttempts || 0));
    }

    return result;
  }, [questions, difficultyFilter, sortBy, activeTab]);

  const renderDifficultyBadge = (question) => {
    const stats = question.stats;
    const level = getDifficultyLevel(stats?.difficulty, stats?.hasEnoughData);
    const label = getDifficultyLabel(level);
    const badgeClass = getDifficultyBadgeClass(level);
    const value = formatDifficultyValue(stats?.difficulty, stats?.hasEnoughData);

    return (
      <span className={`badge badge-sm ${badgeClass} badge-outline`}>
        {label} {stats?.hasEnoughData ? `(${value})` : ''}
      </span>
    );
  };

  const renderAttemptBadge = (question) => {
    const count = question.stats?.totalAttempts || 0;
    return (
      <span className="badge badge-sm badge-ghost">
        作答 {count} 次
      </span>
    );
  };

  return (
    <div className="min-h-screen bg-board px-4 py-6 md:px-8 md:py-8">
      <header className="mx-auto mb-6 flex max-w-7xl flex-col gap-3 rounded-3xl border border-white/70 bg-white/90 px-6 py-5 shadow-card md:flex-row md:items-center md:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.25em] text-sky-700">Teacher Console</p>
          <h1 className="mt-1 text-2xl font-bold text-slate-800">教师机管理员面板</h1>
          <p className="text-sm text-slate-600">题库修改后学生机拉取即同步。</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <button className="btn btn-outline btn-secondary" onClick={loadDashboard}>
            刷新看板
          </button>
          <button className="btn btn-outline btn-amber" onClick={onNavigateToSubjective}>
            主观题管理
          </button>
          <button className="btn btn-outline btn-warning" onClick={onNavigateToGrading}>
            批改工作台
          </button>
          <button className="btn btn-outline btn-info" onClick={onNavigateToExam}>
            考试排考
          </button>
          <button className="btn btn-outline btn-success" onClick={onNavigateToReport}>
            成绩报表
          </button>
          <button className="btn btn-outline btn-error" onClick={onNavigateToProctor}>
            🛡️ 防作弊监控
          </button>
          <button className="btn btn-outline btn-purple" onClick={onNavigateToPaper}>
            📝 自动组卷
          </button>
          <button className="btn btn-neutral" onClick={onLogout}>
            退出登录
          </button>
        </div>
      </header>

      {loading ? (
        <div className="mx-auto max-w-7xl">
          <div className="grid gap-4 md:grid-cols-4">
            {Array.from({ length: 4 }).map((_, idx) => (
              <div key={`skeleton-${idx}`} className="h-28 animate-pulse rounded-2xl bg-white/80" />
            ))}
          </div>
        </div>
      ) : (
        <main className="mx-auto grid max-w-7xl gap-5">
          <section className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            <StatCard title="学生总数" value={overview?.studentCount ?? 0} />
            <StatCard title="班级数量" value={overview?.classCount ?? 0} />
            <StatCard title="题库题量" value={overview?.questionCount ?? 0} />
            <StatCard title="作答次数" value={overview?.attemptCount ?? 0} />
          </section>

          <section className="grid gap-5 lg:grid-cols-[1.2fr,0.8fr]">
            <article className="rounded-3xl border border-slate-200 bg-white p-5 shadow-card">
              <div className="mb-3 flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
                <h2 className="text-lg font-semibold text-slate-800">题库管理</h2>
                <div className="flex flex-wrap gap-2">
                  <label className="btn btn-outline btn-secondary">
                    {uploading ? '上传中...' : '上传 JSON 题库'}
                    <input type="file" className="hidden" accept="application/json" disabled={uploading} onChange={handleUpload} />
                  </label>
                  <button
                    className="btn btn-outline btn-info"
                    onClick={handleRecalculate}
                    disabled={recalculating}
                  >
                    {recalculating ? '重算中...' : '🔄 重算难度'}
                  </button>
                  <button className="btn btn-primary" onClick={openCreateModal}>
                    新增题目
                  </button>
                </div>
              </div>

              <div className="mb-3 flex flex-wrap items-center gap-2">
                <div className="tabs tabs-boxed">
                  <a
                    className={`tab ${activeTab === 'all' ? 'tab-active' : ''}`}
                    onClick={() => setActiveTab('all')}
                  >
                    全部题目
                  </a>
                  <a
                    className={`tab ${activeTab === 'abnormal' ? 'tab-active' : ''}`}
                    onClick={() => setActiveTab('abnormal')}
                  >
                    异常题目 ({abnormalQuestions.length})
                  </a>
                </div>

                <div className="flex-1" />

                <select
                  className="select select-sm select-bordered"
                  value={difficultyFilter}
                  onChange={(e) => setDifficultyFilter(e.target.value)}
                  disabled={activeTab === 'abnormal'}
                >
                  <option value="all">全部难度</option>
                  <option value="easy">简单</option>
                  <option value="medium">中等</option>
                  <option value="hard">困难</option>
                  <option value="no_data">数据不足</option>
                </select>

                <select
                  className="select select-sm select-bordered"
                  value={sortBy}
                  onChange={(e) => setSortBy(e.target.value)}
                >
                  <option value="default">默认排序</option>
                  <option value="difficulty_asc">难度从易到难</option>
                  <option value="difficulty_desc">难度从难到易</option>
                  <option value="attempts_desc">作答量从高到低</option>
                </select>
              </div>

              <div className="max-h-[460px] overflow-auto rounded-xl border border-slate-200">
                <table className="table table-sm">
                  <thead>
                    <tr>
                      <th>ID</th>
                      <th>题干</th>
                      <th>难度</th>
                      <th>作答量</th>
                      <th>操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredQuestions.map((question) => {
                      const abnormalTypes = getAbnormalTypes(question.stats);
                      const isAbnormal = abnormalTypes.length > 0;
                      return (
                        <tr key={question.id} className={isAbnormal ? 'bg-amber-50/50' : ''}>
                          <td className="font-mono text-xs">{question.id}</td>
                          <td className="max-w-sm truncate" title={question.title}>
                            {question.title}
                            {isAbnormal && (
                              <div className="mt-1">
                                <span className="badge badge-xs badge-error badge-outline">
                                  {abnormalTypes.join('/')}
                                </span>
                              </div>
                            )}
                          </td>
                          <td>{renderDifficultyBadge(question)}</td>
                          <td>{renderAttemptBadge(question)}</td>
                          <td>
                            <div className="flex gap-1">
                              <button className="btn btn-xs btn-ghost" onClick={() => openEditModal(question)}>
                                编辑
                              </button>
                              <button
                                className="btn btn-xs btn-ghost text-error"
                                onClick={() => handleDeleteQuestion(question.id)}
                              >
                                删除
                              </button>
                            </div>
                          </td>
                        </tr>
                      );
                    })}
                    {!filteredQuestions.length ? (
                      <tr>
                        <td colSpan={5} className="text-center text-slate-500">
                          {activeTab === 'abnormal' ? '暂无异常题目' : '当前没有题目，请先新增或上传题库。'}
                        </td>
                      </tr>
                    ) : null}
                  </tbody>
                </table>
              </div>
            </article>

            <article className="rounded-3xl border border-slate-200 bg-white p-5 shadow-card">
              <h2 className="mb-3 text-lg font-semibold text-slate-800">难度分布</h2>
              <DifficultyDistributionChart distribution={difficultyDistribution} />
            </article>
          </section>

          <section className="grid gap-5 lg:grid-cols-2">
            <article className="rounded-3xl border border-slate-200 bg-white p-5 shadow-card">
              <h2 className="mb-3 text-lg font-semibold text-slate-800">最近成绩同步</h2>
              <div className="max-h-[320px] overflow-auto rounded-xl border border-slate-200">
                <table className="table table-sm">
                  <thead>
                    <tr>
                      <th>学生</th>
                      <th>班级</th>
                      <th>成绩</th>
                      <th>时间</th>
                    </tr>
                  </thead>
                  <tbody>
                    {attempts.map((item) => (
                      <tr key={item.id}>
                        <td>{item.student}</td>
                        <td>{item.className}</td>
                        <td>
                          <span className="badge badge-outline">{item.score}/{item.total}</span>
                        </td>
                        <td className="text-xs text-slate-500">{item.createdAt}</td>
                      </tr>
                    ))}
                    {!attempts.length ? (
                      <tr>
                        <td colSpan={4} className="text-center text-slate-500">
                          暂无成绩记录
                        </td>
                      </tr>
                    ) : null}
                  </tbody>
                </table>
              </div>
            </article>

            <article className="rounded-3xl border border-slate-200 bg-white p-5 shadow-card">
              <h2 className="mb-3 text-lg font-semibold text-slate-800">班级错题热区（自动统计）</h2>
              <div className="max-h-[320px] overflow-auto rounded-xl border border-slate-200">
                <table className="table table-sm">
                  <thead>
                    <tr>
                      <th>班级</th>
                      <th>题目ID</th>
                      <th>题目</th>
                      <th>错误次数</th>
                    </tr>
                  </thead>
                  <tbody>
                    {topStats.map((item, index) => (
                      <tr key={`${item.classId}-${item.questionId}-${index}`}>
                        <td>{item.className || '-'}</td>
                        <td>{item.questionId}</td>
                        <td className="max-w-3xl truncate" title={item.question}>
                          {item.question}
                        </td>
                        <td>
                          <span className="badge badge-warning badge-outline">{item.wrongCount}</span>
                        </td>
                      </tr>
                    ))}
                    {!topStats.length ? (
                      <tr>
                        <td colSpan={4} className="text-center text-slate-500">
                          暂无错题统计数据
                        </td>
                      </tr>
                    ) : null}
                  </tbody>
                </table>
              </div>
            </article>
          </section>
        </main>
      )}

      <QuestionEditorModal
        open={modalOpen}
        initialData={editingQuestion}
        onClose={() => {
          setModalOpen(false);
          setEditingQuestion(null);
        }}
        onSubmit={handleSaveQuestion}
        loading={savingQuestion}
        token={token}
        onRollback={async () => {
          await loadDashboard();
        }}
      />
    </div>
  );
}
