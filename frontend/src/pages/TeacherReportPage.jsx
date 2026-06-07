import { useEffect, useState, useCallback } from 'react';
import { toast } from 'react-hot-toast';

import {
  createExport,
  getExportTasks,
  getExportTask,
  downloadExportWithFetch,
  apiRequest,
} from '../api/client';

const DIMENSION_OPTIONS = [
  { value: 'class', label: '按班级' },
  { value: 'exam', label: '按考试' },
  { value: 'time', label: '按时间区间' },
];

const FORMAT_OPTIONS = [
  { value: 'xlsx', label: 'Excel (.xlsx)' },
  { value: 'csv', label: 'CSV (.csv)' },
];

const STATUS_MAP = {
  processing: { label: '处理中', color: 'bg-sky-100 text-sky-700' },
  completed: { label: '已完成', color: 'bg-green-100 text-green-700' },
  failed: { label: '失败', color: 'bg-red-100 text-red-700' },
};

export function TeacherReportPage({ user, token, onLogout, classes }) {
  const [dimension, setDimension] = useState('class');
  const [format, setFormat] = useState('xlsx');
  const [selectedClassIds, setSelectedClassIds] = useState([]);
  const [selectedExamId, setSelectedExamId] = useState('');
  const [startTime, setStartTime] = useState('');
  const [endTime, setEndTime] = useState('');
  const [exporting, setExporting] = useState(false);
  const [exams, setExams] = useState([]);
  const [tasks, setTasks] = useState([]);
  const [tasksTotal, setTasksTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(10);
  const [loadingTasks, setLoadingTasks] = useState(false);
  const [activeTab, setActiveTab] = useState('export');

  const loadExams = useCallback(async () => {
    try {
      const data = await apiRequest('/teacher/exams?pageSize=50', { token });
      setExams(data.items || []);
    } catch (error) {
      toast.error(error.message || '加载考试列表失败');
    }
  }, [token]);

  const loadTasks = useCallback(async () => {
    setLoadingTasks(true);
    try {
      const data = await getExportTasks({ page, pageSize }, token);
      setTasks(data.items || []);
      setTasksTotal(data.total || 0);
    } catch (error) {
      toast.error(error.message || '加载导出历史失败');
    } finally {
      setLoadingTasks(false);
    }
  }, [token, page, pageSize]);

  useEffect(() => {
    loadExams();
    loadTasks();
  }, [loadExams, loadTasks]);

  useEffect(() => {
    if (activeTab !== 'history') return;

    const interval = setInterval(() => {
      const hasProcessing = tasks.some((t) => t.status === 'processing');
      if (hasProcessing) {
        loadTasks();
      }
    }, 3000);

    return () => clearInterval(interval);
  }, [activeTab, tasks, loadTasks]);

  const handleClassToggle = (classId) => {
    setSelectedClassIds((prev) =>
      prev.includes(classId)
        ? prev.filter((id) => id !== classId)
        : [...prev, classId]
    );
  };

  const validateForm = () => {
    if (dimension === 'class' && selectedClassIds.length === 0) {
      toast.error('请至少选择一个班级');
      return false;
    }
    if (dimension === 'exam' && !selectedExamId) {
      toast.error('请选择考试');
      return false;
    }
    if (dimension === 'time') {
      if (!startTime || !endTime) {
        toast.error('请选择开始和结束时间');
        return false;
      }
      if (new Date(startTime) > new Date(endTime)) {
        toast.error('开始时间不能晚于结束时间');
        return false;
      }
    }
    return true;
  };

  const handleExport = async () => {
    if (!validateForm()) return;

    const payload = {
      format,
      dimension,
      classIds: selectedClassIds,
    };

    if (dimension === 'exam') {
      payload.examId = selectedExamId ? Number(selectedExamId) : null;
    }
    if (dimension === 'time') {
      payload.startTime = startTime;
      payload.endTime = endTime;
    }

    setExporting(true);
    try {
      const result = await createExport(payload, token);

      if (result.isAsync) {
        toast.success('报表生成中，请稍候在导出历史中查看');
        setActiveTab('history');
        loadTasks();
      } else {
        if (result.status === 'completed') {
          toast.success('导出成功，正在下载...');
          await downloadExportWithFetch(result.id, token);
        } else if (result.status === 'failed') {
          toast.error(result.errorMsg || '导出失败');
        }
      }
    } catch (error) {
      toast.error(error.message || '导出失败');
    } finally {
      setExporting(false);
    }
  };

  const handleDownload = async (task) => {
    if (task.status !== 'completed') {
      toast.error('报表尚未生成完成');
      return;
    }
    try {
      await downloadExportWithFetch(task.id, token);
      toast.success('开始下载');
    } catch (error) {
      toast.error(error.message || '下载失败');
    }
  };

  const formatFileSize = (bytes) => {
    if (bytes < 1024) return bytes + ' B';
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  };

  const isExpired = (task) => {
    if (!task.expiresAt) return false;
    return new Date(task.expiresAt) < new Date();
  };

  const totalPages = Math.ceil(tasksTotal / pageSize);

  return (
    <div className="min-h-screen bg-board px-4 py-6 md:px-8 md:py-8">
      <header className="mx-auto mb-6 flex max-w-6xl flex-col gap-3 rounded-3xl border border-white/70 bg-white/90 px-6 py-5 shadow-card md:flex-row md:items-center md:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.25em] text-sky-700">Report Center</p>
          <h1 className="mt-1 text-2xl font-bold text-slate-800">成绩报表导出</h1>
          <p className="text-sm text-slate-600">按维度导出成绩报表，支持 Excel 和 CSV 格式。</p>
        </div>
        <button className="btn btn-neutral" onClick={onLogout}>
          退出登录
        </button>
      </header>

      <main className="mx-auto max-w-6xl">
        <div className="tabs mb-4 tabs-boxed bg-white/90">
          <button
            className={`tab ${activeTab === 'export' ? 'tab-active' : ''}`}
            onClick={() => setActiveTab('export')}
          >
            报表导出
          </button>
          <button
            className={`tab ${activeTab === 'history' ? 'tab-active' : ''}`}
            onClick={() => setActiveTab('history')}
          >
            导出历史
            {tasks.some((t) => t.status === 'processing') && (
              <span className="ml-2 badge badge-sm badge-secondary">处理中</span>
            )}
          </button>
        </div>

        {activeTab === 'export' ? (
          <div className="grid gap-5 lg:grid-cols-[1fr,1fr]">
            <article className="rounded-3xl border border-slate-200 bg-white p-6 shadow-card">
              <h2 className="mb-4 text-lg font-semibold text-slate-800">导出设置</h2>

              <div className="space-y-4">
                <div>
                  <label className="label">
                    <span className="label-text font-medium">导出维度</span>
                  </label>
                  <div className="flex gap-2">
                    {DIMENSION_OPTIONS.map((opt) => (
                      <button
                        key={opt.value}
                        className={`btn btn-outline flex-1 ${
                          dimension === opt.value ? 'btn-primary' : ''
                        }`}
                        onClick={() => setDimension(opt.value)}
                      >
                        {opt.label}
                      </button>
                    ))}
                  </div>
                </div>

                <div>
                  <label className="label">
                    <span className="label-text font-medium">导出格式</span>
                  </label>
                  <div className="flex gap-2">
                    {FORMAT_OPTIONS.map((opt) => (
                      <button
                        key={opt.value}
                        className={`btn btn-outline flex-1 ${
                          format === opt.value ? 'btn-accent' : ''
                        }`}
                        onClick={() => setFormat(opt.value)}
                      >
                        {opt.label}
                      </button>
                    ))}
                  </div>
                </div>

                {dimension === 'class' && (
                  <div>
                    <label className="label">
                      <span className="label-text font-medium">选择班级</span>
                      <span className="label-text-alt">
                        已选 {selectedClassIds.length} 个
                      </span>
                    </label>
                    <div className="max-h-48 overflow-auto rounded-xl border border-slate-200 p-2">
                      {classes.map((cls) => (
                        <label
                          key={cls.id}
                          className="flex cursor-pointer items-center gap-2 rounded-lg p-2 hover:bg-slate-50"
                        >
                          <input
                            type="checkbox"
                            className="checkbox checkbox-sm"
                            checked={selectedClassIds.includes(cls.id)}
                            onChange={() => handleClassToggle(cls.id)}
                          />
                          <span>{cls.name}</span>
                        </label>
                      ))}
                    </div>
                  </div>
                )}

                {dimension === 'exam' && (
                  <div>
                    <label className="label">
                      <span className="label-text font-medium">选择考试</span>
                    </label>
                    <select
                      className="select select-bordered w-full"
                      value={selectedExamId}
                      onChange={(e) => setSelectedExamId(e.target.value)}
                    >
                      <option value="">请选择考试</option>
                      {exams.map((exam) => (
                        <option key={exam.id} value={exam.id}>
                          {exam.name}
                        </option>
                      ))}
                    </select>
                  </div>
                )}

                {dimension === 'time' && (
                  <div className="space-y-3">
                    <div>
                      <label className="label">
                        <span className="label-text font-medium">开始日期</span>
                      </label>
                      <input
                        type="date"
                        className="input input-bordered w-full"
                        value={startTime}
                        onChange={(e) => setStartTime(e.target.value)}
                      />
                    </div>
                    <div>
                      <label className="label">
                        <span className="label-text font-medium">结束日期</span>
                      </label>
                      <input
                        type="date"
                        className="input input-bordered w-full"
                        value={endTime}
                        onChange={(e) => setEndTime(e.target.value)}
                      />
                    </div>
                  </div>
                )}
              </div>

              <div className="mt-6">
                <button
                  className="btn btn-primary w-full"
                  onClick={handleExport}
                  disabled={exporting}
                >
                  {exporting ? (
                    <>
                      <span className="loading loading-spinner loading-sm"></span>
                      生成中...
                    </>
                  ) : (
                    '生成报表'
                  )}
                </button>
                <p className="mt-2 text-xs text-slate-500">
                  💡 小数据量报表直接下载，大数据量将异步生成并可在"导出历史"中查看进度
                </p>
              </div>
            </article>

            <article className="rounded-3xl border border-slate-200 bg-white p-6 shadow-card">
              <h2 className="mb-4 text-lg font-semibold text-slate-800">报表说明</h2>

              <div className="space-y-4 text-sm text-slate-600">
                <div className="rounded-xl bg-sky-50 p-4">
                  <h3 className="mb-2 font-semibold text-sky-700">📊 总览 Sheet</h3>
                  <ul className="space-y-1 text-sky-900">
                    <li>• 班级人数</li>
                    <li>• 平均分</li>
                    <li>• 最高分 / 最低分</li>
                    <li>• 及格率（60分及以上）</li>
                    <li>• 总分</li>
                  </ul>
                </div>

                <div className="rounded-xl bg-emerald-50 p-4">
                  <h3 className="mb-2 font-semibold text-emerald-700">📋 明细 Sheet</h3>
                  <ul className="space-y-1 text-emerald-900">
                    <li>• 按班级分 Sheet 展示</li>
                    <li>• 每位学生每次答题成绩</li>
                    <li>• 得分、总分、正确率</li>
                    <li>• 答题时间</li>
                  </ul>
                </div>

                <div className="rounded-xl bg-amber-50 p-4">
                  <h3 className="mb-2 font-semibold text-amber-700">⏰ 有效期说明</h3>
                  <ul className="space-y-1 text-amber-900">
                    <li>• 导出文件保留 24 小时</li>
                    <li>• 过期后自动清理，请及时下载</li>
                    <li>• 大报表采用流式生成，不占内存</li>
                  </ul>
                </div>
              </div>
            </article>
          </div>
        ) : (
          <article className="rounded-3xl border border-slate-200 bg-white p-6 shadow-card">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-lg font-semibold text-slate-800">导出历史</h2>
              <button
                className="btn btn-outline btn-sm"
                onClick={loadTasks}
                disabled={loadingTasks}
              >
                刷新
              </button>
            </div>

            <div className="overflow-x-auto rounded-xl border border-slate-200">
              <table className="table table-sm">
                <thead>
                  <tr>
                    <th>ID</th>
                    <th>维度</th>
                    <th>格式</th>
                    <th>状态</th>
                    <th>进度</th>
                    <th>记录数</th>
                    <th>文件大小</th>
                    <th>过期时间</th>
                    <th>操作</th>
                  </tr>
                </thead>
                <tbody>
                  {loadingTasks && tasks.length === 0 ? (
                    <tr>
                      <td colSpan={9} className="text-center py-8">
                        <span className="loading loading-spinner"></span>
                        <p className="mt-2 text-slate-500">加载中...</p>
                      </td>
                    </tr>
                  ) : tasks.length === 0 ? (
                    <tr>
                      <td colSpan={9} className="text-center text-slate-500 py-8">
                        暂无导出记录
                      </td>
                    </tr>
                  ) : (
                    tasks.map((task) => {
                      const statusInfo = STATUS_MAP[task.status] || STATUS_MAP.processing;
                      const expired = isExpired(task);
                      return (
                        <tr key={task.id} className={expired ? 'opacity-50' : ''}>
                          <td className="font-mono text-xs">#{task.id}</td>
                          <td>
                            {task.dimension === 'class' && '按班级'}
                            {task.dimension === 'exam' && '按考试'}
                            {task.dimension === 'time' && '按时间'}
                          </td>
                          <td>
                            <span className="badge badge-outline badge-sm">
                              {task.format.toUpperCase()}
                            </span>
                          </td>
                          <td>
                            <span className={`badge badge-sm ${statusInfo.color}`}>
                              {statusInfo.label}
                            </span>
                          </td>
                          <td>
                            {task.status === 'processing' ? (
                              <div className="flex items-center gap-2">
                                <progress
                                  className="progress progress-sm w-20"
                                  value={task.progress}
                                  max="100"
                                ></progress>
                                <span className="text-xs">{task.progress}%</span>
                              </div>
                            ) : (
                              <span className="text-xs">{task.progress}%</span>
                            )}
                          </td>
                          <td>{task.totalRecords || '-'}</td>
                          <td>
                            {task.fileSize ? formatFileSize(task.fileSize) : '-'}
                          </td>
                          <td className="text-xs">
                            {task.expiresAt ? (
                              <>
                                {expired ? (
                                  <span className="text-red-500">已过期</span>
                                ) : (
                                  task.expiresAt
                                )}
                              </>
                            ) : (
                              '-'
                            )}
                          </td>
                          <td>
                            {task.status === 'completed' && !expired ? (
                              <button
                                className="btn btn-xs btn-primary"
                                onClick={() => handleDownload(task)}
                              >
                                下载
                              </button>
                            ) : task.status === 'failed' ? (
                              <span className="text-xs text-red-500" title={task.errorMsg}>
                                {task.errorMsg || '失败'}
                              </span>
                            ) : (
                              <span className="text-xs text-slate-400">处理中</span>
                            )}
                          </td>
                        </tr>
                      );
                    })
                  )}
                </tbody>
              </table>
            </div>

            {totalPages > 1 && (
              <div className="mt-4 flex justify-center">
                <div className="join">
                  <button
                    className="join-item btn btn-sm"
                    onClick={() => setPage((p) => Math.max(1, p - 1))}
                    disabled={page === 1}
                  >
                    上一页
                  </button>
                  <button className="join-item btn btn-sm btn-disabled">
                    {page} / {totalPages}
                  </button>
                  <button
                    className="join-item btn btn-sm"
                    onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                    disabled={page === totalPages}
                  >
                    下一页
                  </button>
                </div>
              </div>
            )}

            <div className="mt-4 rounded-lg bg-amber-50 p-3 text-xs text-amber-700">
              💡 提示：导出文件保留 24 小时后自动清理，请在有效期内下载保存。
            </div>
          </article>
        )}
      </main>
    </div>
  );
}
