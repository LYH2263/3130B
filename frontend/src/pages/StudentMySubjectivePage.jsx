import { useEffect, useMemo, useState } from 'react';
import { toast } from 'react-hot-toast';

import { getStudentSubjectiveSubmissions } from '../api/client';
import { StatCard } from '../components/StatCard';

export function StudentMySubjectivePage({ user, token, onLogout, onNavigateToPractice }) {
  const [submissions, setSubmissions] = useState([]);
  const [loading, setLoading] = useState(true);
  const [selectedId, setSelectedId] = useState(null);

  const loadData = async () => {
    setLoading(true);
    try {
      const data = await getStudentSubjectiveSubmissions(token);
      setSubmissions(data);
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

  const stats = useMemo(() => {
    const total = submissions.length;
    const graded = submissions.filter((s) => s.status === 'graded').length;
    const pending = submissions.filter((s) => s.status === 'pending').length;
    let avgScore = '0%';
    if (graded > 0) {
      const totalScore = submissions
        .filter((s) => s.status === 'graded' && s.score != null)
        .reduce((sum, s) => sum + s.score, 0);
      const totalFull = submissions
        .filter((s) => s.status === 'graded' && s.question?.fullScore)
        .reduce((sum, s) => sum + (s.question?.fullScore || 0), 0);
      if (totalFull > 0) {
        avgScore = `${Math.round((totalScore / totalFull) * 100)}%`;
      }
    }
    return { total, graded, pending, avgScore };
  }, [submissions]);

  const selectedSubmission = useMemo(
    () => submissions.find((s) => s.id === selectedId),
    [submissions, selectedId]
  );

  const getStatusBadge = (status) => {
    if (status === 'pending') {
      return <span className="badge badge-warning">待批改</span>;
    }
    if (status === 'graded') {
      return <span className="badge badge-success">已批改</span>;
    }
    return <span className="badge">{status}</span>;
  };

  return (
    <div className="min-h-screen bg-board px-4 py-6 md:px-8 md:py-8">
      <header className="mx-auto mb-6 flex max-w-7xl flex-col gap-3 rounded-3xl border border-white/70 bg-white/90 px-6 py-5 shadow-card md:flex-row md:items-center md:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.25em] text-purple-700">
            My Subjective
          </p>
          <h1 className="mt-1 text-2xl font-bold text-slate-800">我的主观题</h1>
          <p className="text-sm text-slate-600">查看你的主观题作答记录和批改结果</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <button className="btn btn-outline btn-secondary" onClick={loadData}>
            刷新
          </button>
          <button className="btn btn-outline btn-primary" onClick={onNavigateToPractice}>
            去答题
          </button>
          <button className="btn btn-neutral" onClick={onLogout}>
            退出登录
          </button>
        </div>
      </header>

      <main className="mx-auto max-w-7xl space-y-5">
        <section className="grid gap-4 md:grid-cols-4">
          <StatCard title="总提交数" value={stats.total} />
          <StatCard title="待批改" value={stats.pending} />
          <StatCard title="已批改" value={stats.graded} />
          <StatCard title="平均得分率" value={stats.avgScore} />
        </section>

        <div className="grid gap-5 lg:grid-cols-[1fr,1.2fr]">
          <article className="rounded-3xl border border-slate-200 bg-white p-5 shadow-card">
            <h2 className="mb-3 text-lg font-semibold text-slate-800">作答记录</h2>
            <div className="max-h-[500px] overflow-auto rounded-xl border border-slate-200">
              {loading ? (
                <div className="p-4 space-y-2">
                  {Array.from({ length: 5 }).map((_, i) => (
                    <div key={i} className="h-16 animate-pulse rounded-lg bg-slate-100" />
                  ))}
                </div>
              ) : submissions.length === 0 ? (
                <div className="p-12 text-center">
                  <p className="text-slate-500">还没有主观题作答记录</p>
                  <button
                    className="btn btn-primary btn-sm mt-4"
                    onClick={onNavigateToPractice}
                  >
                    去答题
                  </button>
                </div>
              ) : (
                <ul className="divide-y divide-slate-100">
                  {submissions.map((sub) => (
                    <li
                      key={sub.id}
                      className={`cursor-pointer p-3 transition hover:bg-slate-50 ${
                        selectedId === sub.id ? 'bg-sky-50 border-l-4 border-l-sky-500' : ''
                      }`}
                      onClick={() => setSelectedId(sub.id)}
                    >
                      <div className="flex items-start justify-between gap-2">
                        <div className="min-w-0 flex-1">
                          <p className="truncate text-sm font-medium text-slate-800">
                            第{sub.questionId}题
                          </p>
                          <p className="mt-1 truncate text-xs text-slate-500">
                            {sub.question?.title || '...'}
                          </p>
                          <p className="mt-1 text-xs text-slate-400">
                            {new Date(sub.submittedAt).toLocaleString()}
                          </p>
                        </div>
                        <div className="flex flex-col items-end gap-1">
                          {getStatusBadge(sub.status)}
                          {sub.status === 'graded' && sub.score != null && (
                            <p className="text-xs font-semibold text-emerald-600">
                              {sub.score}/{sub.question?.fullScore || '?'}
                            </p>
                          )}
                        </div>
                      </div>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </article>

          <article className="rounded-3xl border border-slate-200 bg-white p-5 shadow-card min-h-[400px]">
            <h2 className="mb-3 text-lg font-semibold text-slate-800">详情</h2>
            {!selectedSubmission ? (
              <div className="flex h-full items-center justify-center py-20">
                <p className="text-slate-400">请从左侧选择一条记录查看详情</p>
              </div>
            ) : (
              <div className="space-y-4">
                <div className="flex items-center justify-between border-b border-slate-200 pb-3">
                  <div>
                    <h3 className="text-base font-semibold text-slate-800">
                      第{selectedSubmission.questionId}题
                    </h3>
                    <p className="text-xs text-slate-500">
                      满分 {selectedSubmission.question?.fullScore || '?'} 分
                    </p>
                  </div>
                  {getStatusBadge(selectedSubmission.status)}
                </div>

                <div className="rounded-xl bg-slate-50 p-4">
                  <p className="mb-2 text-xs font-medium text-slate-500">题目</p>
                  <div className="whitespace-pre-wrap text-sm text-slate-700">
                    {selectedSubmission.question?.title || '-'}
                  </div>
                </div>

                <div className="rounded-xl border border-amber-200 bg-amber-50/50 p-4">
                  <p className="mb-2 text-xs font-medium text-amber-700">
                    我的作答
                    <span className="ml-2 font-normal text-amber-600">
                      {new Date(selectedSubmission.submittedAt).toLocaleString()}
                    </span>
                  </p>
                  <div className="whitespace-pre-wrap text-sm text-slate-700">
                    {selectedSubmission.content}
                  </div>
                </div>

                {selectedSubmission.status === 'graded' && (
                  <div className="rounded-xl border border-emerald-200 bg-emerald-50/50 p-4 space-y-3">
                    <div className="flex items-center gap-3">
                      <p className="text-sm font-medium text-emerald-800">批改结果</p>
                      <span className="badge badge-success badge-lg">
                        {selectedSubmission.score}/{selectedSubmission.question?.fullScore || '?'} 分
                      </span>
                    </div>

                    {selectedSubmission.comment && (
                      <div>
                        <p className="mb-1 text-xs font-medium text-emerald-700">教师评语</p>
                        <p className="whitespace-pre-wrap text-sm text-emerald-800">
                          {selectedSubmission.comment}
                        </p>
                      </div>
                    )}

                    {selectedSubmission.grader && (
                      <p className="text-xs text-emerald-600">
                        批改教师：{selectedSubmission.grader.username}
                        {selectedSubmission.gradedAt &&
                          ` · ${new Date(selectedSubmission.gradedAt).toLocaleString()}`}
                      </p>
                    )}
                  </div>
                )}

                {selectedSubmission.status === 'pending' && (
                  <div className="rounded-xl border border-slate-200 bg-slate-50 p-6 text-center">
                    <p className="text-slate-500">⏳ 教师正在批改中，请耐心等待...</p>
                  </div>
                )}
              </div>
            )}
          </article>
        </div>
      </main>
    </div>
  );
}
