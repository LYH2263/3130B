import { useEffect, useMemo, useState } from 'react';
import { toast } from 'react-hot-toast';

import {
  getStudentSubjectiveQuestions,
  getStudentSubjectiveSubmissions,
  submitSubjectiveAnswer,
} from '../api/client';

import { DiscussionSection } from '../components/DiscussionSection';

export function StudentSubjectivePage({ user, token, onLogout, onNavigateToMy }) {
  const [questions, setQuestions] = useState([]);
  const [submissions, setSubmissions] = useState([]);
  const [loading, setLoading] = useState(true);
  const [selectedQuestion, setSelectedQuestion] = useState(null);
  const [answer, setAnswer] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const loadData = async () => {
    setLoading(true);
    try {
      const [qData, sData] = await Promise.all([
        getStudentSubjectiveQuestions(token),
        getStudentSubjectiveSubmissions(token),
      ]);
      setQuestions(qData);
      setSubmissions(sData);
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

  const submittedQuestionIds = useMemo(() => {
    return new Set(submissions.map((s) => s.questionId));
  }, [submissions]);

  const handleSelectQuestion = (question) => {
    setSelectedQuestion(question);
    const sub = submissions.find((s) => s.questionId === question.id);
    setAnswer(sub ? sub.content : '');
  };

  const handleSubmit = async () => {
    if (!selectedQuestion) return;
    if (!answer.trim()) {
      toast.error('请输入作答内容');
      return;
    }

    try {
      setSubmitting(true);
      await submitSubjectiveAnswer(
        { questionId: selectedQuestion.id, content: answer },
        token
      );
      toast.success('提交成功，等待教师批改');
      await loadData();
      setSelectedQuestion(null);
      setAnswer('');
    } catch (error) {
      toast.error(error.message || '提交失败');
    } finally {
      setSubmitting(false);
    }
  };

  const getStatusBadge = (questionId) => {
    const sub = submissions.find((s) => s.questionId === questionId);
    if (!sub) {
      return <span className="badge badge-ghost badge-sm">未作答</span>;
    }
    if (sub.status === 'pending') {
      return <span className="badge badge-warning badge-sm">待批改</span>;
    }
    if (sub.status === 'graded') {
      return (
        <span className="badge badge-success badge-sm">
          {sub.score}/{sub.question?.fullScore || '?'} 分
        </span>
      );
    }
    return null;
  };

  const isSubmitted = selectedQuestion && submittedQuestionIds.has(selectedQuestion.id);
  const currentSubmission = selectedQuestion
    ? submissions.find((s) => s.questionId === selectedQuestion.id)
    : null;

  return (
    <div className="min-h-screen bg-board px-4 py-6 md:px-8 md:py-8">
      <header className="mx-auto mb-6 flex max-w-7xl flex-col gap-3 rounded-3xl border border-white/70 bg-white/90 px-6 py-5 shadow-card md:flex-row md:items-center md:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.25em] text-emerald-700">
            Subjective Practice
          </p>
          <h1 className="mt-1 text-2xl font-bold text-slate-800">主观题练习</h1>
          <p className="text-sm text-slate-600">
            共 {questions.length} 道题，已提交 {submissions.length} 份
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <button className="btn btn-outline btn-secondary" onClick={loadData}>
            刷新
          </button>
          <button className="btn btn-outline btn-primary" onClick={onNavigateToMy}>
            我的主观题
          </button>
          <button className="btn btn-neutral" onClick={onLogout}>
            退出登录
          </button>
        </div>
      </header>

      <main className="mx-auto grid max-w-7xl gap-5 lg:grid-cols-[320px,1fr]">
        <aside className="rounded-3xl border border-slate-200 bg-white p-4 shadow-card">
          <h2 className="mb-3 text-lg font-semibold text-slate-800">题目列表</h2>
          <div className="max-h-[calc(100vh-280px)] overflow-auto rounded-xl border border-slate-200">
            {loading ? (
              <div className="p-4 space-y-2">
                {Array.from({ length: 5 }).map((_, i) => (
                  <div key={i} className="h-14 animate-pulse rounded-lg bg-slate-100" />
                ))}
              </div>
            ) : questions.length === 0 ? (
              <div className="p-8 text-center text-sm text-slate-500">
                暂无可用题目
              </div>
            ) : (
              <ul className="divide-y divide-slate-100">
                {questions.map((q) => (
                  <li
                    key={q.id}
                    className={`cursor-pointer p-3 transition hover:bg-slate-50 ${
                      selectedQuestion?.id === q.id
                        ? 'bg-sky-50 border-l-4 border-l-sky-500'
                        : ''
                    }`}
                    onClick={() => handleSelectQuestion(q)}
                  >
                    <div className="flex items-start justify-between gap-2">
                      <p className="truncate text-sm font-medium text-slate-800">
                        第{q.id}题
                      </p>
                      {getStatusBadge(q.id)}
                    </div>
                    <p className="mt-1 truncate text-xs text-slate-500" title={q.title}>
                      {q.title}
                    </p>
                    <p className="mt-1 text-xs text-slate-400">满分 {q.fullScore} 分</p>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </aside>

        <section className="rounded-3xl border border-slate-200 bg-white p-5 shadow-card min-h-[400px]">
          {!selectedQuestion ? (
            <div className="flex h-full items-center justify-center py-20">
              <div className="text-center">
                <p className="text-slate-400">请从左侧选择一道题目开始作答</p>
              </div>
            </div>
          ) : (
            <div className="space-y-4">
              <div className="border-b border-slate-200 pb-4">
                <div className="flex items-center justify-between">
                  <h2 className="text-lg font-semibold text-slate-800">
                    第{selectedQuestion.id}题（满分 {selectedQuestion.fullScore} 分）
                  </h2>
                  {getStatusBadge(selectedQuestion.id)}
                </div>
                <div className="mt-3 rounded-xl bg-slate-50 p-4">
                  <div className="whitespace-pre-wrap text-sm text-slate-700">
                    {selectedQuestion.title}
                  </div>
                </div>
              </div>

              {isSubmitted ? (
                <div className="space-y-4">
                  <div className="rounded-xl border border-amber-200 bg-amber-50/50 p-4">
                    <p className="mb-2 text-sm font-medium text-amber-800">
                      我的作答
                      {currentSubmission && (
                        <span className="ml-2 text-xs font-normal text-amber-600">
                          提交时间：{new Date(currentSubmission.submittedAt).toLocaleString()}
                        </span>
                      )}
                    </p>
                    <div className="whitespace-pre-wrap text-sm text-slate-700">
                      {answer}
                    </div>
                  </div>

                  {currentSubmission?.status === 'graded' && (
                    <div className="rounded-xl border border-emerald-200 bg-emerald-50/50 p-4 space-y-3">
                      <div className="flex items-center gap-2">
                        <p className="text-sm font-medium text-emerald-800">批改结果</p>
                        <span className="badge badge-success">
                          {currentSubmission.score}/{selectedQuestion.fullScore} 分
                        </span>
                      </div>
                      {currentSubmission.comment && (
                        <div>
                          <p className="mb-1 text-xs font-medium text-emerald-700">教师评语</p>
                          <p className="whitespace-pre-wrap text-sm text-emerald-800">
                            {currentSubmission.comment}
                          </p>
                        </div>
                      )}
                      {currentSubmission.grader && (
                        <p className="text-xs text-emerald-600">
                          批改教师：{currentSubmission.grader.username}
                          {currentSubmission.gradedAt &&
                            ` · ${new Date(currentSubmission.gradedAt).toLocaleString()}`}
                        </p>
                      )}
                    </div>
                  )}

                  {currentSubmission?.status === 'pending' && (
                    <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 text-center">
                      <p className="text-sm text-slate-500">
                        ⏳ 教师正在批改中，请耐心等待...
                      </p>
                    </div>
                  )}
                </div>
              ) : (
                <div className="space-y-4">
                  <label className="form-control">
                    <span className="label-text mb-2 text-sm font-medium">
                      请输入你的作答内容
                    </span>
                    <textarea
                      className="textarea textarea-bordered min-h-64"
                      value={answer}
                      onChange={(e) => setAnswer(e.target.value)}
                      placeholder="请在此输入你的答案..."
                    />
                    <p className="mt-1 text-xs text-slate-400">
                      支持文本格式作答，提交后将由教师人工批改。
                    </p>
                  </label>

                  <div className="flex justify-end gap-2">
                    <button
                      className="btn btn-outline btn-ghost"
                      onClick={() => {
                        setSelectedQuestion(null);
                        setAnswer('');
                      }}
                    >
                      取消
                    </button>
                    <button
                      className="btn btn-primary"
                      onClick={handleSubmit}
                      disabled={submitting || !answer.trim()}
                    >
                      {submitting ? '提交中...' : '提交作答'}
                    </button>
                  </div>
                </div>
              )}

              <DiscussionSection
                questionId={selectedQuestion.id}
                token={token}
                user={user}
                defaultOpen={isSubmitted}
              />
            </div>
          )}
        </section>
      </main>
    </div>
  );
}
