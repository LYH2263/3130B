import { useEffect, useState, useCallback } from 'react';
import { toast } from 'react-hot-toast';

import { enterExam, submitExam, getExamResult } from '../api/client';

function formatDateTime(dateStr) {
  if (!dateStr) return '';
  const d = new Date(dateStr);
  return d.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function formatTimeRemaining(seconds) {
  if (seconds <= 0) return '00:00';
  const mins = Math.floor(seconds / 60);
  const secs = seconds % 60;
  return `${String(mins).padStart(2, '0')}:${String(secs).padStart(2, '0')}`;
}

export function ExamPage({ exam, token, onBack, onFinish }) {
  const [phase, setPhase] = useState('intro');
  const [countdown, setCountdown] = useState(5);
  const [questions, setQuestions] = useState([]);
  const [answers, setAnswers] = useState({});
  const [currentIndex, setCurrentIndex] = useState(0);
  const [timeLeft, setTimeLeft] = useState(0);
  const [submitting, setSubmitting] = useState(false);
  const [result, setResult] = useState(null);
  const [loading, setLoading] = useState(false);
  const [examData, setExamData] = useState(exam);
  const [participantData, setParticipantData] = useState(null);

  const handleStartExam = useCallback(async () => {
    setLoading(true);
    try {
      const data = await enterExam(exam.id, token);
      setExamData(data.exam);
      setParticipantData(data.participant);

      const mockQuestions = [];
      for (let i = 1; i <= 10; i++) {
        mockQuestions.push({
          id: i,
          title: `第 ${i} 题：这是一道示例题目，请选择正确答案。`,
          options: [
            { id: i * 4 + 1, content: '选项 A' },
            { id: i * 4 + 2, content: '选项 B' },
            { id: i * 4 + 3, content: '选项 C' },
            { id: i * 4 + 4, content: '选项 D' },
          ],
        });
      }
      setQuestions(mockQuestions);

      const duration = data.exam.duration || 60;
      setTimeLeft(duration * 60);

      setPhase('countdown');
      setCountdown(3);
    } catch (error) {
      toast.error(error.message || '进入考试失败');
      onBack?.();
    } finally {
      setLoading(false);
    }
  }, [exam.id, token, onBack]);

  useEffect(() => {
    if (phase !== 'countdown') return;
    if (countdown <= 0) {
      setPhase('exam');
      return;
    }
    const timer = setTimeout(() => setCountdown(countdown - 1), 1000);
    return () => clearTimeout(timer);
  }, [phase, countdown]);

  useEffect(() => {
    if (phase !== 'exam') return;
    if (timeLeft <= 0) {
      handleSubmit();
      return;
    }
    const timer = setInterval(() => {
      setTimeLeft((prev) => prev - 1);
    }, 1000);
    return () => clearInterval(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [phase, timeLeft]);

  const handleSelectOption = (questionId, optionId) => {
    setAnswers((prev) => ({
      ...prev,
      [questionId]: optionId,
    }));
  };

  const handleSubmit = useCallback(async () => {
    if (submitting) return;

    const unanswered = questions.filter((q) => !answers[q.id]).length;
    if (unanswered > 0 && phase === 'exam') {
      if (!window.confirm(`还有 ${unanswered} 道题未作答，确定要交卷吗？`)) {
        return;
      }
    }

    setSubmitting(true);
    try {
      const answerList = Object.entries(answers).map(([questionId, optionId]) => ({
        questionId: Number(questionId),
        optionId: Number(optionId),
      }));

      const data = await submitExam(exam.id, answerList, token);
      setResult(data);
      setPhase('result');
      toast.success('交卷成功！');
    } catch (error) {
      toast.error(error.message || '交卷失败');
    } finally {
      setSubmitting(false);
    }
  }, [answers, exam.id, token, questions.length, phase, submitting]);

  const handleViewResult = useCallback(async () => {
    setLoading(true);
    try {
      const data = await getExamResult(exam.id, token);
      setResult(data);
      setPhase('result');
    } catch (error) {
      toast.error(error.message || '加载成绩失败');
    } finally {
      setLoading(false);
    }
  }, [exam.id, token]);

  useEffect(() => {
    if (exam.status === 'finished') {
      handleViewResult();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const answeredCount = Object.keys(answers).length;
  const currentQuestion = questions[currentIndex];
  const progress = questions.length > 0 ? (answeredCount / questions.length) * 100 : 0;

  if (phase === 'intro') {
    return (
      <div className="min-h-screen bg-board px-4 py-8">
        <div className="mx-auto max-w-2xl">
          <button className="mb-6 btn btn-ghost btn-sm" onClick={onBack}>
            ← 返回考试中心
          </button>

          <div className="rounded-3xl border border-white/70 bg-white/90 p-8 shadow-card">
            <div className="mb-6 text-center">
              <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-primary/10">
                <svg
                  className="h-8 w-8 text-primary"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
                  />
                </svg>
              </div>
              <h1 className="text-2xl font-bold text-slate-800">{exam.name}</h1>
              <p className="mt-2 text-slate-500">考试须知</p>
            </div>

            <div className="mb-8 space-y-4 rounded-2xl bg-slate-50 p-6">
              <div className="flex gap-3">
                <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary/20 text-sm font-bold text-primary">
                  1
                </div>
                <div>
                  <p className="font-medium text-slate-700">考试时间</p>
                  <p className="text-sm text-slate-500">
                    {formatDateTime(exam.startTime)} - {formatDateTime(exam.endTime)}
                  </p>
                </div>
              </div>
              <div className="flex gap-3">
                <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary/20 text-sm font-bold text-primary">
                  2
                </div>
                <div>
                  <p className="font-medium text-slate-700">考试时长</p>
                  <p className="text-sm text-slate-500">{exam.duration} 分钟</p>
                </div>
              </div>
              <div className="flex gap-3">
                <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary/20 text-sm font-bold text-primary">
                  3
                </div>
                <div>
                  <p className="font-medium text-slate-700">注意事项</p>
                  <ul className="mt-1 space-y-1 text-sm text-slate-500">
                    <li>• 请在考试开始后再进入答题页面</li>
                    <li>• 考试期间请勿刷新或关闭页面</li>
                    <li>• 每道题只有一个正确答案</li>
                    <li>• 交卷后无法修改答案</li>
                    <li>• 考试时间结束将自动交卷</li>
                  </ul>
                </div>
              </div>
            </div>

            <div className="flex gap-3">
              <button className="btn btn-ghost flex-1" onClick={onBack}>
                返回
              </button>
              <button
                className="btn btn-primary flex-1"
                onClick={handleStartExam}
                disabled={loading}
              >
                {loading ? '加载中...' : '开始考试'}
              </button>
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (phase === 'countdown') {
    return (
      <div className="flex min-h-screen items-center justify-center bg-board">
        <div className="text-center">
          <p className="mb-4 text-lg text-slate-600">考试即将开始</p>
          <div className="text-8xl font-bold text-primary">{countdown}</div>
          <p className="mt-4 text-slate-500">请做好准备...</p>
        </div>
      </div>
    );
  }

  if (phase === 'exam') {
    return (
      <div className="min-h-screen bg-board">
        <div className="sticky top-0 z-10 border-b border-white/50 bg-white/90 backdrop-blur">
          <div className="mx-auto flex max-w-5xl items-center justify-between px-4 py-3">
            <div>
              <h2 className="font-bold text-slate-800">{exam.name}</h2>
              <p className="text-sm text-slate-500">
                第 {currentIndex + 1} / {questions.length} 题
              </p>
            </div>
            <div className="flex items-center gap-4">
              <div
                className={`rounded-full px-4 py-2 font-mono text-lg font-bold ${
                  timeLeft < 300 ? 'bg-red-100 text-red-600' : 'bg-slate-100 text-slate-700'
                }`}
              >
                {formatTimeRemaining(timeLeft)}
              </div>
              <button
                className="btn btn-primary btn-sm"
                onClick={handleSubmit}
                disabled={submitting}
              >
                {submitting ? '交卷中...' : '交卷'}
              </button>
            </div>
          </div>
          <div className="h-1 w-full bg-slate-200">
            <div
              className="h-full bg-primary transition-all duration-300"
              style={{ width: `${progress}%` }}
            />
          </div>
        </div>

        <div className="mx-auto flex max-w-5xl gap-6 px-4 py-6">
          <div className="flex-1">
            {currentQuestion && (
              <div className="rounded-2xl border border-white/70 bg-white/90 p-6 shadow-card">
                <div className="mb-4 flex items-start gap-3">
                  <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary/10 font-bold text-primary">
                    {currentIndex + 1}
                  </span>
                  <p className="text-lg font-medium text-slate-800">{currentQuestion.title}</p>
                </div>

                <div className="space-y-3">
                  {currentQuestion.options.map((option, idx) => {
                    const selected = answers[currentQuestion.id] === option.id;
                    const optionLabel = String.fromCharCode(65 + idx);
                    return (
                      <button
                        key={option.id}
                        className={`flex w-full items-center gap-3 rounded-xl border p-4 text-left transition-all ${
                          selected
                            ? 'border-primary bg-primary/5 ring-2 ring-primary/30'
                            : 'border-slate-200 bg-slate-50 hover:border-slate-300 hover:bg-white'
                        }`}
                        onClick={() => handleSelectOption(currentQuestion.id, option.id)}
                      >
                        <span
                          className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-sm font-bold ${
                            selected
                              ? 'bg-primary text-white'
                              : 'bg-slate-200 text-slate-600'
                          }`}
                        >
                          {optionLabel}
                        </span>
                        <span className={selected ? 'text-primary' : 'text-slate-700'}>
                          {option.content}
                        </span>
                      </button>
                    );
                  })}
                </div>

                <div className="mt-6 flex justify-between">
                  <button
                    className="btn btn-ghost"
                    onClick={() => setCurrentIndex((prev) => Math.max(0, prev - 1))}
                    disabled={currentIndex === 0}
                  >
                    ← 上一题
                  </button>
                  <button
                    className="btn btn-primary"
                    onClick={() =>
                      setCurrentIndex((prev) => Math.min(questions.length - 1, prev + 1))
                    }
                    disabled={currentIndex === questions.length - 1}
                  >
                    下一题 →
                  </button>
                </div>
              </div>
            )}
          </div>

          <div className="hidden w-56 shrink-0 lg:block">
            <div className="sticky top-24 rounded-2xl border border-white/70 bg-white/90 p-4 shadow-card">
              <h3 className="mb-3 font-bold text-slate-700">答题卡</h3>
              <div className="grid grid-cols-5 gap-2">
                {questions.map((q, idx) => {
                  const answered = !!answers[q.id];
                  const isCurrent = idx === currentIndex;
                  return (
                    <button
                      key={q.id}
                      className={`h-9 w-9 rounded-lg text-sm font-medium transition-all ${
                        isCurrent
                          ? 'bg-primary text-white ring-2 ring-primary/30'
                          : answered
                          ? 'bg-green-100 text-green-700'
                          : 'bg-slate-100 text-slate-600 hover:bg-slate-200'
                      }`}
                      onClick={() => setCurrentIndex(idx)}
                    >
                      {idx + 1}
                    </button>
                  );
                })}
              </div>
              <div className="mt-4 text-sm text-slate-500">
                已答：{answeredCount} / {questions.length}
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (phase === 'result') {
    const score = result?.score ?? 0;
    const isPassed = score >= 60;

    return (
      <div className="min-h-screen bg-board px-4 py-8">
        <div className="mx-auto max-w-2xl">
          <div className="rounded-3xl border border-white/70 bg-white/90 p-8 shadow-card">
            <div className="mb-6 text-center">
              <div
                className={`mx-auto mb-4 flex h-20 w-20 items-center justify-center rounded-full ${
                  isPassed ? 'bg-green-100' : 'bg-red-100'
                }`}
              >
                <svg
                  className={`h-10 w-10 ${isPassed ? 'text-green-500' : 'text-red-500'}`}
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  {isPassed ? (
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
                    />
                  ) : (
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z"
                    />
                  )}
                </svg>
              </div>
              <h1 className="text-2xl font-bold text-slate-800">考试结束</h1>
              <p className="mt-1 text-slate-500">{exam.name}</p>
            </div>

            <div className="mb-6 rounded-2xl bg-slate-50 p-6 text-center">
              <p className="text-sm text-slate-500">你的得分</p>
              <p
                className={`text-5xl font-bold ${
                  isPassed ? 'text-green-500' : 'text-red-500'
                }`}
              >
                {score}
                <span className="text-2xl text-slate-400"> / 100</span>
              </p>
              <p className={`mt-2 text-sm ${isPassed ? 'text-green-600' : 'text-red-600'}`}>
                {isPassed ? '恭喜你，考试通过！' : '很遗憾，未达到及格线'}
              </p>
            </div>

            <div className="mb-6 grid grid-cols-2 gap-4">
              <div className="rounded-xl bg-slate-50 p-4 text-center">
                <p className="text-sm text-slate-500">答题数量</p>
                <p className="mt-1 text-xl font-bold text-slate-700">{answeredCount} 题</p>
              </div>
              <div className="rounded-xl bg-slate-50 p-4 text-center">
                <p className="text-sm text-slate-500">提交时间</p>
                <p className="mt-1 text-sm font-medium text-slate-700">
                  {result?.submittedAt ? formatDateTime(result.submittedAt) : '-'}
                </p>
              </div>
            </div>

            <button
              className="btn btn-primary w-full"
              onClick={() => onFinish?.()}
            >
              返回考试中心
            </button>
          </div>
        </div>
      </div>
    );
  }

  return null;
}
