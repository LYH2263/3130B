import { useEffect, useMemo, useState } from 'react';
import { toast } from 'react-hot-toast';

import {
  getSubjectiveSubmissions,
  getSubjectiveSubmission,
  gradeSubjectiveSubmission,
  getSubjectiveQuestions,
} from '../api/client';

export function TeacherGradingPage({ user, token, onLogout, onNavigateToQuestions }) {
  const [submissions, setSubmissions] = useState([]);
  const [questions, setQuestions] = useState([]);
  const [loading, setLoading] = useState(true);
  const [selectedId, setSelectedId] = useState(null);
  const [selectedSubmission, setSelectedSubmission] = useState(null);
  const [grading, setGrading] = useState(false);
  const [score, setScore] = useState('');
  const [comment, setComment] = useState('');
  const [filterQuestion, setFilterQuestion] = useState('');
  const [filterStatus, setFilterStatus] = useState('pending');

  const filteredSubmissions = useMemo(() => {
    let result = [...submissions];
    if (filterQuestion) {
      result = result.filter((s) => s.questionId === parseInt(filterQuestion));
    }
    if (filterStatus) {
      result = result.filter((s) => s.status === filterStatus);
    }
    return result;
  }, [submissions, filterQuestion, filterStatus]);

  const loadData = async () => {
    setLoading(true);
    try {
      const [subData, qData] = await Promise.all([
        getSubjectiveSubmissions({ pageSize: 100 }, token),
        getSubjectiveQuestions(token),
      ]);
      setSubmissions(subData.items || []);
      setQuestions(qData);
    } catch (error) {
      toast.error(error.message || '加载数据失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  const loadSubmissionDetail = async (id) => {
    try {
      const data = await getSubjectiveSubmission(id, token);
      setSelectedSubmission(data);
      setScore(data.score != null ? String(data.score) : '');
      setComment(data.comment || '');
    } catch (error) {
      toast.error(error.message || '加载详情失败');
    }
  };

  const handleSelect = (id) => {
    setSelectedId(id);
    loadSubmissionDetail(id);
  };

  const handleGrade = async () => {
    if (!selectedSubmission) return;

    const scoreNum = parseFloat(score);
    if (isNaN(scoreNum) || scoreNum < 0) {
      toast.error('请输入有效分数');
      return;
    }

    const fullScore = selectedSubmission.question?.fullScore || 10;
    if (scoreNum > fullScore) {
      toast.error(`分数不能超过满分 ${fullScore} 分`);
      return;
    }

    try {
      setGrading(true);
      await gradeSubjectiveSubmission(selectedSubmission.id, { score: scoreNum, comment }, token);
      toast.success('批改成功');
      await loadData();
      await loadSubmissionDetail(selectedSubmission.id);
    } catch (error) {
      toast.error(error.message || '批改失败');
    } finally {
      setGrading(false);
    }
  };

  const getStatusBadge = (status) => {
    if (status === 'pending') {
      return <span className="badge badge-warning badge-sm">待批改</span>;
    }
    if (status === 'graded') {
      return <span className="badge badge-success badge-sm">已批改</span>;
    }
    return <span className="badge badge-sm">{status}</span>;
  };

  const pendingCount = useMemo(
    () => submissions.filter((s) => s.status === 'pending').length,
    [submissions]
  );

  return (
    <div className="min-h-screen bg-board px-4 py-6 md:px-8 md:py-8">
      <header className="mx-auto mb-6 flex max-w-7xl flex-col gap-3 rounded-3xl border border-white/70 bg-white/90 px-6 py-5 shadow-card md:flex-row md:items-center md:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.25em] text-rose-700">Grading Console</p>
          <h1 className="mt-1 text-2xl font-bold text-slate-800">批改工作台</h1>
          <p className="text-sm text-slate-600">
            待批改 <span className="font-semibold text-rose-600">{pendingCount}</span> 份，共 {submissions.length} 份提交
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <button className="btn btn-outline btn-secondary" onClick={loadData}>
            刷新
          </button>
          <button className="btn btn-outline btn-primary" onClick={onNavigateToQuestions}>
            题库管理
          </button>
          <button className="btn btn-neutral" onClick={onLogout}>
            退出登录
          </button>
        </div>
      </header>

      <main className="mx-auto grid max-w-7xl gap-5 lg:grid-cols-[380px,1fr]">
        <aside className="rounded-3xl border border-slate-200 bg-white p-4 shadow-card">
          <div className="mb-3 space-y-2">
            <h2 className="text-lg font-semibold text-slate-800">待批改队列</h2>

            <div className="grid grid-cols-2 gap-2">
              <select
                className="select select-bordered select-sm w-full"
                value={filterQuestion}
                onChange={(e) => setFilterQuestion(e.target.value)}
              >
                <option value="">全部题目</option>
                {questions.map((q) => (
                  <option key={q.id} value={q.id}>
                    第{q.id}题
                  </option>
                ))}
              </select>

              <select
                className="select select-bordered select-sm w-full"
                value={filterStatus}
                onChange={(e) => setFilterStatus(e.target.value)}
              >
                <option value="">全部状态</option>
                <option value="pending">待批改</option>
                <option value="graded">已批改</option>
              </select>
            </div>
          </div>

          <div className="max-h-[calc(100vh-280px)] overflow-auto rounded-xl border border-slate-200">
            {loading ? (
              <div className="p-4 space-y-2">
                {Array.from({ length: 5 }).map((_, i) => (
                  <div key={i} className="h-16 animate-pulse rounded-lg bg-slate-100" />
                ))}
              </div>
            ) : filteredSubmissions.length === 0 ? (
              <div className="p-8 text-center text-sm text-slate-500">
                暂无符合条件的提交
              </div>
            ) : (
              <ul className="divide-y divide-slate-100">
                {filteredSubmissions.map((sub) => (
                  <li
                    key={sub.id}
                    className={`cursor-pointer p-3 transition hover:bg-slate-50 ${
                      selectedId === sub.id ? 'bg-sky-50 border-l-4 border-l-sky-500' : ''
                    }`}
                    onClick={() => handleSelect(sub.id)}
                  >
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-medium text-slate-800">
                          {sub.student?.username || '学生'}
                        </p>
                        <p className="mt-1 truncate text-xs text-slate-500">
                          第{sub.questionId}题 · {sub.question?.title?.slice(0, 20) || '...'}
                        </p>
                        <p className="mt-1 text-xs text-slate-400">
                          {new Date(sub.submittedAt).toLocaleString()}
                        </p>
                      </div>
                      <div className="flex-shrink-0">
                        {getStatusBadge(sub.status)}
                        {sub.status === 'graded' && sub.score != null && (
                          <p className="mt-1 text-right text-xs font-medium text-emerald-600">
                            {sub.score}/{sub.question?.fullScore}
                          </p>
                        )}
                      </div>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </aside>

        <section className="rounded-3xl border border-slate-200 bg-white p-5 shadow-card min-h-[400px]">
          {!selectedSubmission ? (
            <div className="flex h-full items-center justify-center py-20">
              <div className="text-center">
                <p className="text-slate-400">请从左侧选择一份提交进行批改</p>
              </div>
            </div>
          ) : (
            <div className="space-y-4">
              <div className="flex items-start justify-between border-b border-slate-200 pb-4">
                <div>
                  <div className="flex items-center gap-2">
                    <h2 className="text-lg font-semibold text-slate-800">学生作答</h2>
                    {getStatusBadge(selectedSubmission.status)}
                  </div>
                  <p className="mt-1 text-sm text-slate-500">
                    学生：{selectedSubmission.student?.username || '-'} 
                    {selectedSubmission.student?.classRoom?.name && 
                      ` · ${selectedSubmission.student.classRoom.name}`
                    }
                    {selectedSubmission.grader && 
                      ` · 批改人：${selectedSubmission.grader.username}`
                    }
                  </p>
                  <p className="mt-1 text-xs text-slate-400">
                    提交时间：{new Date(selectedSubmission.submittedAt).toLocaleString()}
                    {selectedSubmission.gradedAt && 
                      ` · 批改时间：${new Date(selectedSubmission.gradedAt).toLocaleString()}`
                    }
                  </p>
                </div>
              </div>

              <div className="rounded-xl border border-amber-200 bg-amber-50/50 p-4">
                <p className="mb-2 text-sm font-medium text-amber-800">
                  题目（满分 {selectedSubmission.question?.fullScore ?? 10} 分）
                </p>
                <div className="whitespace-pre-wrap text-sm text-slate-700">
                  {selectedSubmission.question?.title || '-'}
                </div>
                {selectedSubmission.question?.referenceAnswer && (
                  <details className="mt-3">
                    <summary className="cursor-pointer text-xs text-slate-500">
                      查看参考答案
                    </summary>
                    <p className="mt-2 whitespace-pre-wrap text-sm text-slate-600">
                      {selectedSubmission.question.referenceAnswer}
                    </p>
                  </details>
                )}
              </div>

              <div className="rounded-xl border border-slate-200 p-4">
                <p className="mb-2 text-sm font-medium text-slate-700">学生作答内容</p>
                <div className="min-h-32 whitespace-pre-wrap text-sm text-slate-800">
                  {selectedSubmission.content || <span className="text-slate-400">无作答内容</span>}
                </div>
              </div>

              <div className="rounded-xl border border-slate-200 bg-slate-50/50 p-4 space-y-4">
                <h3 className="text-sm font-semibold text-slate-700">评分与评语</h3>

                <div className="grid gap-4 md:grid-cols-[200px,1fr]">
                  <label className="form-control">
                    <span className="label-text mb-1 text-sm font-medium">得分</span>
                    <div className="flex items-center gap-2">
                      <input
                        type="number"
                        step="0.5"
                        min="0"
                        max={selectedSubmission.question?.fullScore || 10}
                        className="input input-bordered w-full"
                        value={score}
                        onChange={(e) => setScore(e.target.value)}
                        placeholder="输入分数"
                        disabled={selectedSubmission.status === 'graded'}
                      />
                      <span className="text-sm text-slate-500">
                        / {selectedSubmission.question?.fullScore ?? 10}
                      </span>
                    </div>
                  </label>
                </div>

                <label className="form-control">
                  <span className="label-text mb-1 text-sm font-medium">教师评语</span>
                  <textarea
                    className="textarea textarea-bordered min-h-24"
                    value={comment}
                    onChange={(e) => setComment(e.target.value)}
                    placeholder="请输入评语..."
                    disabled={selectedSubmission.status === 'graded'}
                  />
                </label>

                <div className="flex justify-end gap-2">
                  {selectedSubmission.status === 'graded' ? (
                    <button
                      className="btn btn-outline btn-secondary"
                      onClick={() => {
                        if (window.confirm('确定要重新批改吗？')) {
                          setGrading(false);
                          const submission = selectedSubmission;
                          submission.status = 'pending';
                          setSelectedSubmission(submission);
                        }
                      }}
                    >
                      重新批改
                    </button>
                  ) : (
                    <button
                      className="btn btn-primary"
                      onClick={handleGrade}
                      disabled={grading || !selectedId}
                    >
                      {grading ? '提交中...' : '提交批改'}
                    </button>
                  )}
                </div>
              </div>
            </div>
          )}
        </section>
      </main>
    </div>
  );
}
