import { useEffect, useState, useMemo } from 'react';
import { toast } from 'react-hot-toast';

import {
  getExams,
  getProctorExamStats,
  getProctorStudentEvents,
  getProctorExamConfig,
  saveProctorExamConfig,
} from '../api/client';

const statusMap = {
  normal: { label: '正常', className: 'badge badge-success badge-outline' },
  warning: { label: '警告', className: 'badge badge-warning badge-outline' },
  suspicious: { label: '可疑', className: 'badge badge-error badge-outline' },
  force_submitted: { label: '强制交卷', className: 'badge badge-error' },
};

const eventLabelMap = {
  tab_switch: '切屏',
  blur: '失焦',
  copy: '复制',
  paste: '粘贴',
  fullscreen_exit: '全屏退出',
  reconnect: '断线重连',
};

const severityColorMap = {
  low: 'bg-blue-100 text-blue-700',
  medium: 'bg-amber-100 text-amber-700',
  high: 'bg-red-100 text-red-700',
};

function formatDateTime(dateStr) {
  if (!dateStr) return '-';
  const d = new Date(dateStr);
  return d.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
}

function getHeatColor(score, threshold) {
  if (score === 0) return 'bg-green-50';
  const ratio = score / threshold;
  if (ratio < 0.3) return 'bg-green-100';
  if (ratio < 0.5) return 'bg-yellow-100';
  if (ratio < 0.7) return 'bg-orange-200';
  if (ratio < 1) return 'bg-orange-300';
  return 'bg-red-400';
}

export function TeacherProctorPage({ user, token, onLogout, classes }) {
  const [exams, setExams] = useState([]);
  const [selectedExam, setSelectedExam] = useState(null);
  const [stats, setStats] = useState(null);
  const [loading, setLoading] = useState(true);
  const [statsLoading, setStatsLoading] = useState(false);
  const [selectedStudent, setSelectedStudent] = useState(null);
  const [studentEvents, setStudentEvents] = useState([]);
  const [eventsLoading, setEventsLoading] = useState(false);
  const [showConfig, setShowConfig] = useState(false);
  const [config, setConfig] = useState(null);
  const [configLoading, setConfigLoading] = useState(false);
  const [savingConfig, setSavingConfig] = useState(false);
  const [configForm, setConfigForm] = useState({
    warningThreshold: 3,
    forceSubmitThreshold: 5,
    tabSwitchWeight: 1,
    blurWeight: 1,
    copyWeight: 2,
    pasteWeight: 2,
    fullscreenExitWeight: 1,
    reconnectWeight: 1,
    autoForceSubmit: false,
    autoMarkSuspicious: true,
    enabled: true,
  });
  const [autoRefresh, setAutoRefresh] = useState(true);

  const loadExams = async () => {
    setLoading(true);
    try {
      const data = await getExams({ pageSize: 100 }, token);
      setExams(data.items || []);
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

  const loadStats = async (examId) => {
    setStatsLoading(true);
    try {
      const data = await getProctorExamStats(examId, token);
      setStats(data);
    } catch (error) {
      toast.error(error.message || '加载监控数据失败');
    } finally {
      setStatsLoading(false);
    }
  };

  useEffect(() => {
    if (selectedExam && autoRefresh) {
      loadStats(selectedExam.id);
      const interval = setInterval(() => {
        loadStats(selectedExam.id);
      }, 10000);
      return () => clearInterval(interval);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedExam?.id, autoRefresh]);

  const handleSelectExam = (exam) => {
    setSelectedExam(exam);
    setSelectedStudent(null);
    setStats(null);
    loadStats(exam.id);
  };

  const loadStudentEvents = async (examId, studentId) => {
    setEventsLoading(true);
    try {
      const data = await getProctorStudentEvents(examId, studentId, token, { pageSize: 50 });
      setStudentEvents(data.items || []);
    } catch (error) {
      toast.error(error.message || '加载事件详情失败');
    } finally {
      setEventsLoading(false);
    }
  };

  const handleViewStudent = (student) => {
    setSelectedStudent(student);
    if (selectedExam) {
      loadStudentEvents(selectedExam.id, student.studentId);
    }
  };

  const loadConfig = async (examId) => {
    setConfigLoading(true);
    try {
      const data = await getProctorExamConfig(examId, token);
      setConfig(data);
      setConfigForm({
        warningThreshold: data.warningThreshold,
        forceSubmitThreshold: data.forceSubmitThreshold,
        tabSwitchWeight: data.tabSwitchWeight,
        blurWeight: data.blurWeight,
        copyWeight: data.copyWeight,
        pasteWeight: data.pasteWeight,
        fullscreenExitWeight: data.fullscreenExitWeight,
        reconnectWeight: data.reconnectWeight,
        autoForceSubmit: data.autoForceSubmit,
        autoMarkSuspicious: data.autoMarkSuspicious,
        enabled: data.enabled,
      });
    } catch (error) {
      toast.error(error.message || '加载配置失败');
    } finally {
      setConfigLoading(false);
    }
  };

  const handleOpenConfig = () => {
    if (selectedExam) {
      loadConfig(selectedExam.id);
    }
    setShowConfig(true);
  };

  const handleSaveConfig = async () => {
    if (!selectedExam) return;
    setSavingConfig(true);
    try {
      const data = await saveProctorExamConfig(selectedExam.id, configForm, token);
      setConfig(data);
      toast.success('配置已保存');
      setShowConfig(false);
      loadStats(selectedExam.id);
    } catch (error) {
      toast.error(error.message || '保存配置失败');
    } finally {
      setSavingConfig(false);
    }
  };

  const sortedStudents = useMemo(() => {
    if (!stats?.studentStats) return [];
    return [...stats.studentStats].sort((a, b) => b.violationScore - a.violationScore);
  }, [stats]);

  const ongoingExams = exams.filter((e) => e.status === 'ongoing');
  const otherExams = exams.filter((e) => e.status !== 'ongoing');

  return (
    <div className="min-h-screen bg-board px-4 py-6 md:px-8 md:py-8">
      <header className="mx-auto mb-6 flex max-w-7xl flex-col gap-3 rounded-3xl border border-white/70 bg-white/90 px-6 py-5 shadow-card md:flex-row md:items-center md:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.25em] text-amber-700">Proctor Monitoring</p>
          <h1 className="mt-1 text-2xl font-bold text-slate-800">防作弊监控中心</h1>
          <p className="text-sm text-slate-600">实时监控考试中的违规行为，保障考试公平。</p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <button className="btn btn-ghost" onClick={onLogout}>
            退出登录
          </button>
          <label className="flex items-center gap-2 rounded-lg bg-white/80 px-3 py-2 text-sm">
            <input
              type="checkbox"
              checked={autoRefresh}
              onChange={(e) => setAutoRefresh(e.target.checked)}
              className="checkbox checkbox-sm"
            />
            自动刷新
          </label>
          {selectedExam && (
            <button className="btn btn-primary" onClick={handleOpenConfig}>
              ⚙️ 监控配置
            </button>
          )}
        </div>
      </header>

      <div className="mx-auto grid max-w-7xl gap-6 lg:grid-cols-4">
        <div className="lg:col-span-1">
          <div className="sticky top-6 rounded-2xl border border-white/70 bg-white/90 p-4 shadow-card">
            <h3 className="mb-3 font-bold text-slate-700">考试列表</h3>

            {loading ? (
              <div className="space-y-2">
                {[1, 2, 3].map((i) => (
                  <div key={i} className="h-12 animate-pulse rounded-lg bg-slate-100" />
                ))}
              </div>
            ) : (
              <div className="space-y-4">
                {ongoingExams.length > 0 && (
                  <div>
                    <p className="mb-2 text-xs font-medium text-green-600">进行中</p>
                    <div className="space-y-1">
                      {ongoingExams.map((exam) => (
                        <button
                          key={exam.id}
                          className={`w-full rounded-lg px-3 py-2 text-left text-sm transition-all ${
                            selectedExam?.id === exam.id
                              ? 'bg-primary/10 text-primary ring-1 ring-primary/30'
                              : 'hover:bg-slate-50'
                          }`}
                          onClick={() => handleSelectExam(exam)}
                        >
                          <p className="font-medium">{exam.name}</p>
                          <p className="text-xs text-slate-500">
                            {formatDateTime(exam.startTime)}
                          </p>
                        </button>
                      ))}
                    </div>
                  </div>
                )}

                {otherExams.length > 0 && (
                  <div>
                    <p className="mb-2 text-xs font-medium text-slate-500">其他考试</p>
                    <div className="space-y-1">
                      {otherExams.slice(0, 10).map((exam) => (
                        <button
                          key={exam.id}
                          className={`w-full rounded-lg px-3 py-2 text-left text-sm transition-all ${
                            selectedExam?.id === exam.id
                              ? 'bg-primary/10 text-primary ring-1 ring-primary/30'
                              : 'hover:bg-slate-50'
                          }`}
                          onClick={() => handleSelectExam(exam)}
                        >
                          <p className="font-medium text-slate-700">{exam.name}</p>
                          <p className="text-xs text-slate-400">
                            {formatDateTime(exam.startTime)}
                          </p>
                        </button>
                      ))}
                    </div>
                  </div>
                )}

                {exams.length === 0 && (
                  <p className="py-4 text-center text-sm text-slate-400">暂无考试</p>
                )}
              </div>
            )}
          </div>
        </div>

        <div className="lg:col-span-3">
          {!selectedExam ? (
            <div className="flex h-64 items-center justify-center rounded-2xl border border-white/70 bg-white/90 shadow-card">
              <div className="text-center">
                <div className="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-slate-100">
                  <svg
                    className="h-6 w-6 text-slate-400"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
                    />
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
                    />
                  </svg>
                </div>
                <p className="text-slate-500">请从左侧选择一场考试查看监控</p>
              </div>
            </div>
          ) : statsLoading && !stats ? (
            <div className="animate-pulse rounded-2xl border border-white/70 bg-white/90 p-8 shadow-card">
              <div className="h-6 w-1/3 rounded bg-slate-200" />
              <div className="mt-6 grid grid-cols-4 gap-4">
                {[1, 2, 3, 4].map((i) => (
                  <div key={i} className="h-20 rounded-lg bg-slate-100" />
                ))}
              </div>
              <div className="mt-6 h-80 rounded-lg bg-slate-50" />
            </div>
          ) : (
            <div className="space-y-6">
              <div className="rounded-2xl border border-white/70 bg-white/90 p-6 shadow-card">
                <div className="flex items-start justify-between">
                  <div>
                    <h2 className="text-xl font-bold text-slate-800">{stats?.examName}</h2>
                    <p className="mt-1 text-sm text-slate-500">
                      共 {stats?.totalStudents} 名考生 · {stats?.totalEvents} 条事件记录
                    </p>
                  </div>
                  <div className="flex gap-3">
                    <div className="text-center">
                      <p className="text-2xl font-bold text-amber-500">{stats?.warningCount || 0}</p>
                      <p className="text-xs text-slate-500">有违规记录</p>
                    </div>
                    <div className="text-center">
                      <p className="text-2xl font-bold text-red-500">{stats?.suspiciousCount || 0}</p>
                      <p className="text-xs text-slate-500">可疑/强制</p>
                    </div>
                  </div>
                </div>

                <div className="mt-4 flex flex-wrap gap-2">
                  <span className="badge badge-success badge-sm">正常</span>
                  <span className="badge badge-warning badge-sm">警告</span>
                  <span className="badge badge-error badge-sm">可疑</span>
                  <span className="badge badge-error">强制交卷</span>
                </div>
              </div>

              <div className="rounded-2xl border border-white/70 bg-white/90 p-6 shadow-card">
                <h3 className="mb-4 font-bold text-slate-700">违规热力图</h3>
                <div className="grid grid-cols-5 gap-3 md:grid-cols-8 lg:grid-cols-10">
                  {sortedStudents.map((student) => {
                    const heatColor = getHeatColor(
                      student.violationScore,
                      stats?.warningThreshold || 3
                    );
                    return (
                      <button
                        key={student.studentId}
                        className={`group relative aspect-square rounded-lg ${heatColor} transition-all hover:ring-2 hover:ring-primary/50`}
                        onClick={() => handleViewStudent(student)}
                        title={`${student.studentName} - ${student.violationScore} 分`}
                      >
                        <span className="absolute inset-0 flex items-center justify-center text-xs font-medium text-slate-700 opacity-0 group-hover:opacity-100">
                          {student.studentName?.slice(0, 2)}
                        </span>
                      </button>
                    );
                  })}
                </div>
                {sortedStudents.length === 0 && (
                  <p className="py-8 text-center text-sm text-slate-400">暂无考生数据</p>
                )}
              </div>

              <div className="rounded-2xl border border-white/70 bg-white/90 p-6 shadow-card">
                <h3 className="mb-4 font-bold text-slate-700">考生违规详情</h3>
                <div className="overflow-x-auto">
                  <table className="table w-full">
                    <thead>
                      <tr>
                        <th>考生</th>
                        <th>状态</th>
                        <th>违规得分</th>
                        <th>事件数</th>
                        <th>切屏</th>
                        <th>复制/粘贴</th>
                        <th>失焦</th>
                        <th>最近事件</th>
                        <th>操作</th>
                      </tr>
                    </thead>
                    <tbody>
                      {sortedStudents.map((student) => (
                        <tr key={student.studentId}>
                          <td className="font-medium">{student.studentName || '未知'}</td>
                          <td>
                            <span
                              className={
                                statusMap[student.status]?.className ||
                                'badge badge-ghost badge-outline'
                              }
                            >
                              {statusMap[student.status]?.label || student.status}
                            </span>
                          </td>
                          <td>
                            <span
                              className={`font-bold ${
                                student.violationScore >= (stats?.warningThreshold || 3)
                                  ? 'text-red-500'
                                  : student.violationScore > 0
                                  ? 'text-amber-500'
                                  : 'text-green-500'
                              }`}
                            >
                              {student.violationScore}
                            </span>
                          </td>
                          <td>{student.totalEvents}</td>
                          <td>{student.eventBreakdown?.tab_switch || 0}</td>
                          <td>
                            {(student.eventBreakdown?.copy || 0) +
                              (student.eventBreakdown?.paste || 0)}
                          </td>
                          <td>{student.eventBreakdown?.blur || 0}</td>
                          <td className="text-sm text-slate-500">
                            {student.lastEventTime ? formatDateTime(student.lastEventTime) : '-'}
                          </td>
                          <td>
                            <button
                              className="btn btn-xs btn-ghost"
                              onClick={() => handleViewStudent(student)}
                            >
                              详情
                            </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {selectedStudent && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
          <div className="w-full max-w-2xl rounded-2xl bg-white p-6 shadow-xl">
            <div className="mb-4 flex items-center justify-between">
              <div>
                <h3 className="text-xl font-bold text-slate-800">
                  {selectedStudent.studentName} 的违规记录
                </h3>
                <p className="text-sm text-slate-500">
                  违规得分：{selectedStudent.violationScore} · 共 {selectedStudent.totalEvents} 条事件
                </p>
              </div>
              <button
                className="btn btn-sm btn-ghost"
                onClick={() => setSelectedStudent(null)}
              >
                关闭
              </button>
            </div>

            {eventsLoading ? (
              <div className="py-8 text-center text-slate-500">加载中...</div>
            ) : studentEvents.length === 0 ? (
              <div className="py-8 text-center text-slate-400">暂无违规事件记录</div>
            ) : (
              <div className="max-h-96 overflow-auto">
                <div className="space-y-2">
                  {studentEvents.map((event) => (
                    <div
                      key={event.id}
                      className="flex items-center gap-3 rounded-lg border border-slate-100 bg-slate-50 p-3"
                    >
                      <span
                        className={`rounded px-2 py-1 text-xs font-medium ${
                          severityColorMap[event.severity] || 'bg-slate-100 text-slate-600'
                        }`}
                      >
                        {eventLabelMap[event.eventType] || event.eventType}
                      </span>
                      <span className="flex-1 text-sm text-slate-600">
                        {event.extraInfo || '-'}
                      </span>
                      <span className="text-xs text-slate-400">
                        {formatDateTime(event.eventTime)}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {showConfig && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
          <div className="w-full max-w-lg rounded-2xl bg-white p-6 shadow-xl">
            <div className="mb-4 flex items-center justify-between">
              <h3 className="text-xl font-bold text-slate-800">监控配置</h3>
              <button
                className="btn btn-sm btn-ghost"
                onClick={() => setShowConfig(false)}
              >
                关闭
              </button>
            </div>

            {configLoading ? (
              <div className="py-8 text-center text-slate-500">加载中...</div>
            ) : (
              <div className="space-y-4">
                <div className="flex items-center justify-between rounded-lg bg-slate-50 p-3">
                  <span className="text-sm font-medium text-slate-700">启用监控</span>
                  <input
                    type="checkbox"
                    className="toggle"
                    checked={configForm.enabled}
                    onChange={(e) =>
                      setConfigForm({ ...configForm, enabled: e.target.checked })
                    }
                  />
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="mb-1 block text-sm font-medium text-slate-700">
                      警告阈值
                    </label>
                    <input
                      type="number"
                      min="1"
                      className="input input-bordered w-full"
                      value={configForm.warningThreshold}
                      onChange={(e) =>
                        setConfigForm({
                          ...configForm,
                          warningThreshold: Number(e.target.value),
                        })
                      }
                    />
                  </div>
                  <div>
                    <label className="mb-1 block text-sm font-medium text-slate-700">
                      强制交卷阈值
                    </label>
                    <input
                      type="number"
                      min="1"
                      className="input input-bordered w-full"
                      value={configForm.forceSubmitThreshold}
                      onChange={(e) =>
                        setConfigForm({
                          ...configForm,
                          forceSubmitThreshold: Number(e.target.value),
                        })
                      }
                    />
                  </div>
                </div>

                <div className="border-t border-slate-100 pt-4">
                  <p className="mb-3 text-sm font-medium text-slate-700">事件权重配置</p>
                  <div className="grid grid-cols-2 gap-3">
                    {[
                      { key: 'tabSwitchWeight', label: '切屏' },
                      { key: 'blurWeight', label: '失焦' },
                      { key: 'copyWeight', label: '复制' },
                      { key: 'pasteWeight', label: '粘贴' },
                      { key: 'fullscreenExitWeight', label: '全屏退出' },
                      { key: 'reconnectWeight', label: '断线重连' },
                    ].map(({ key, label }) => (
                      <div key={key}>
                        <label className="mb-1 block text-xs text-slate-500">{label}</label>
                        <input
                          type="number"
                          min="0"
                          className="input input-bordered input-sm w-full"
                          value={configForm[key]}
                          onChange={(e) =>
                            setConfigForm({ ...configForm, [key]: Number(e.target.value) })
                          }
                        />
                      </div>
                    ))}
                  </div>
                </div>

                <div className="flex items-center justify-between rounded-lg bg-slate-50 p-3">
                  <span className="text-sm font-medium text-slate-700">自动标记可疑</span>
                  <input
                    type="checkbox"
                    className="toggle"
                    checked={configForm.autoMarkSuspicious}
                    onChange={(e) =>
                      setConfigForm({ ...configForm, autoMarkSuspicious: e.target.checked })
                    }
                  />
                </div>

                <div className="flex items-center justify-between rounded-lg bg-red-50 p-3">
                  <div>
                    <span className="text-sm font-medium text-red-700">自动强制交卷</span>
                    <p className="text-xs text-red-500">触发阈值后自动提交试卷（0分）</p>
                  </div>
                  <input
                    type="checkbox"
                    className="toggle"
                    checked={configForm.autoForceSubmit}
                    onChange={(e) =>
                      setConfigForm({ ...configForm, autoForceSubmit: e.target.checked })
                    }
                  />
                </div>
              </div>
            )}

            <div className="mt-6 flex justify-end gap-2">
              <button
                className="btn btn-ghost"
                onClick={() => setShowConfig(false)}
              >
                取消
              </button>
              <button
                className="btn btn-primary"
                onClick={handleSaveConfig}
                disabled={savingConfig}
              >
                {savingConfig ? '保存中...' : '保存配置'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
