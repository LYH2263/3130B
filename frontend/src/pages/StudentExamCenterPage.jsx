import { useEffect, useState } from 'react';
import { toast } from 'react-hot-toast';

import { getStudentExams, getExamResult } from '../api/client';

function formatDateTime(dateStr) {
  if (!dateStr) return '';
  const d = new Date(dateStr);
  return d.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function formatDate(dateStr) {
  if (!dateStr) return '';
  const d = new Date(dateStr);
  return d.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  });
}

function parseClassIds(classIdsStr) {
  try {
    return JSON.parse(classIdsStr || '[]');
  } catch {
    return [];
  }
}

export function StudentExamCenterPage({ user, token, onLogout, classes, onEnterExam, onViewResult }) {
  const [exams, setExams] = useState({ pending: [], ongoing: [], finished: [] });
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState('ongoing');
  const [results, setResults] = useState({});

  const loadExams = async () => {
    setLoading(true);
    try {
      const data = await getStudentExams(token);
      setExams({
        pending: data.pending || [],
        ongoing: data.ongoing || [],
        finished: data.finished || [],
      });

      const finishedExams = data.finished || [];
      const resultPromises = finishedExams.map(async (exam) => {
        try {
          const result = await getExamResult(exam.id, token);
          return { examId: exam.id, result };
        } catch {
          return { examId: exam.id, result: null };
        }
      });
      const resultData = await Promise.all(resultPromises);
      const resultMap = {};
      resultData.forEach(({ examId, result }) => {
        if (result) {
          resultMap[examId] = result;
        }
      });
      setResults(resultMap);
    } catch (error) {
      toast.error(error.message || '加载考试失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadExams();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  const getClassNames = (classIdsStr) => {
    const ids = parseClassIds(classIdsStr);
    return ids
      .map((id) => classes.find((c) => c.id === id)?.name)
      .filter(Boolean)
      .join('、');
  };

  const handleEnterExam = (exam) => {
    if (onEnterExam) {
      onEnterExam(exam);
    }
  };

  const handleViewResult = (exam) => {
    if (onViewResult) {
      onViewResult(exam);
    }
  };

  const ExamCard = ({ exam, status }) => {
    const result = results[exam.id];
    return (
      <div className="rounded-2xl border border-white/70 bg-white/90 p-5 shadow-card transition-all hover:shadow-lg">
        <div className="mb-3 flex items-start justify-between">
          <div>
            <h3 className="text-lg font-bold text-slate-800">{exam.name}</h3>
            <p className="mt-1 text-sm text-slate-500">{getClassNames(exam.classIds)}</p>
          </div>
          <span
            className={`badge ${
              status === 'ongoing'
                ? 'badge-success'
                : status === 'pending'
                ? 'badge-info'
                : 'badge-ghost'
            }`}
          >
            {status === 'ongoing' ? '进行中' : status === 'pending' ? '未开始' : '已结束'}
          </span>
        </div>

        <div className="mb-4 space-y-2 text-sm text-slate-600">
          <div className="flex items-center gap-2">
            <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            <span>
              {formatDate(exam.startTime)} - {formatDate(exam.endTime)}
            </span>
          </div>
          <div className="flex items-center gap-2">
            <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <span>时长 {exam.duration} 分钟</span>
          </div>
          {status === 'pending' && (
            <div className="text-amber-600">
              开始时间：{formatDateTime(exam.startTime)}
            </div>
          )}
          {status === 'finished' && result && (
            <div className="flex items-center gap-2 font-medium text-primary">
              <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              得分：{result.score !== null && result.score !== undefined ? result.score : '-'} 分
            </div>
          )}
        </div>

        <div className="flex gap-2">
          {status === 'ongoing' && (
            <button
              className="btn btn-primary btn-sm w-full"
              onClick={() => handleEnterExam(exam)}
            >
              进入考试
            </button>
          )}
          {status === 'pending' && (
            <button className="btn btn-ghost btn-sm w-full" disabled>
              等待开始
            </button>
          )}
          {status === 'finished' && (
            <button
              className="btn btn-outline btn-sm w-full"
              onClick={() => handleViewResult(exam)}
            >
              查看成绩
            </button>
          )}
        </div>
      </div>
    );
  };

  const tabConfigs = [
    { key: 'ongoing', label: '进行中', count: exams.ongoing.length, color: 'text-success' },
    { key: 'pending', label: '未开始', count: exams.pending.length, color: 'text-info' },
    { key: 'finished', label: '已结束', count: exams.finished.length, color: 'text-slate-500' },
  ];

  return (
    <div className="min-h-screen bg-board px-4 py-6 md:px-8 md:py-8">
      <header className="mx-auto mb-6 flex max-w-5xl flex-col gap-3 rounded-3xl border border-white/70 bg-white/90 px-6 py-5 shadow-card md:flex-row md:items-center md:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.25em] text-amber-700">Exam Center</p>
          <h1 className="mt-1 text-2xl font-bold text-slate-800">考试中心</h1>
          <p className="text-sm text-slate-600">查看和参加你的考试，查看历史考试成绩。</p>
        </div>
        <button className="btn btn-ghost" onClick={onLogout}>
          退出登录
        </button>
      </header>

      <div className="mx-auto max-w-5xl">
        <div className="mb-6 flex gap-2">
          {tabConfigs.map((tab) => (
            <button
              key={tab.key}
              className={`flex items-center gap-2 rounded-xl px-4 py-2 font-medium transition-all ${
                activeTab === tab.key
                  ? 'bg-white text-primary shadow-md'
                  : 'bg-white/50 text-slate-600 hover:bg-white/80'
              }`}
              onClick={() => setActiveTab(tab.key)}
            >
              <span className={activeTab === tab.key ? 'text-primary' : tab.color}>
                {tab.label}
              </span>
              <span
                className={`rounded-full px-2 py-0.5 text-xs ${
                  activeTab === tab.key ? 'bg-primary/10 text-primary' : 'bg-slate-200 text-slate-600'
                }`}
              >
                {tab.count}
              </span>
            </button>
          ))}
        </div>

        {loading ? (
          <div className="grid gap-4 md:grid-cols-2">
            {[1, 2, 3, 4].map((i) => (
              <div key={i} className="animate-pulse rounded-2xl bg-white/80 p-5">
                <div className="h-6 w-2/3 rounded bg-slate-200" />
                <div className="mt-3 h-4 w-full rounded bg-slate-100" />
                <div className="mt-2 h-4 w-3/4 rounded bg-slate-100" />
                <div className="mt-4 h-8 w-full rounded bg-slate-200" />
              </div>
            ))}
          </div>
        ) : exams[activeTab]?.length === 0 ? (
          <div className="rounded-2xl border border-dashed border-slate-300 bg-white/50 p-12 text-center">
            <svg
              className="mx-auto h-16 w-16 text-slate-300"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1.5}
                d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"
              />
            </svg>
            <p className="mt-4 text-slate-500">
              {activeTab === 'ongoing'
                ? '暂无进行中的考试'
                : activeTab === 'pending'
                ? '暂无待开始的考试'
                : '暂无已结束的考试'}
            </p>
          </div>
        ) : (
          <div className="grid gap-4 md:grid-cols-2">
            {exams[activeTab]?.map((exam) => (
              <ExamCard key={exam.id} exam={exam} status={activeTab} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
