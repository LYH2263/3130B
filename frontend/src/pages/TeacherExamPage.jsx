import { useEffect, useMemo, useState } from 'react';
import { toast } from 'react-hot-toast';

import {
  getExams,
  createExam,
  updateExam,
  deleteExam,
  getExamParticipants,
} from '../api/client';

const statusMap = {
  pending: { label: '未开始', className: 'badge badge-info badge-outline' },
  ongoing: { label: '进行中', className: 'badge badge-success badge-outline' },
  finished: { label: '已结束', className: 'badge badge-ghost badge-outline' },
  cancelled: { label: '已取消', className: 'badge badge-error badge-outline' },
};

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

function parseClassIds(classIdsStr) {
  try {
    return JSON.parse(classIdsStr || '[]');
  } catch {
    return [];
  }
}

export function TeacherExamPage({ user, token, onLogout, classes }) {
  const [exams, setExams] = useState([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [viewMode, setViewMode] = useState('list');
  const [modalOpen, setModalOpen] = useState(false);
  const [editingExam, setEditingExam] = useState(null);
  const [selectedExam, setSelectedExam] = useState(null);
  const [participants, setParticipants] = useState([]);
  const [participantsLoading, setParticipantsLoading] = useState(false);
  const [currentMonth, setCurrentMonth] = useState(() => {
    const now = new Date();
    return { year: now.getFullYear(), month: now.getMonth() };
  });

  const [formData, setFormData] = useState({
    name: '',
    startTime: '',
    endTime: '',
    duration: 60,
    classIds: [],
    questionSetId: null,
  });

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

  const openCreate = () => {
    setEditingExam(null);
    setFormData({
      name: '',
      startTime: '',
      endTime: '',
      duration: 60,
      classIds: [],
      questionSetId: null,
    });
    setModalOpen(true);
  };

  const openEdit = (exam) => {
    setEditingExam(exam);
    const classIds = parseClassIds(exam.classIds);
    const start = new Date(exam.startTime);
    const end = new Date(exam.endTime);
    setFormData({
      name: exam.name,
      startTime: start.toISOString().slice(0, 16),
      endTime: end.toISOString().slice(0, 16),
      duration: exam.duration,
      classIds: classIds,
      questionSetId: exam.questionSetId,
    });
    setModalOpen(true);
  };

  const handleSave = async () => {
    if (!formData.name.trim()) {
      toast.error('请输入考试名称');
      return;
    }
    if (!formData.startTime || !formData.endTime) {
      toast.error('请选择考试时间');
      return;
    }
    if (formData.classIds.length === 0) {
      toast.error('请选择参与班级');
      return;
    }
    if (formData.duration <= 0) {
      toast.error('考试时长必须大于0');
      return;
    }

    const payload = {
      name: formData.name.trim(),
      startTime: new Date(formData.startTime).toISOString(),
      endTime: new Date(formData.endTime).toISOString(),
      duration: Number(formData.duration),
      classIds: formData.classIds,
      questionSetId: formData.questionSetId,
    };

    try {
      setSaving(true);
      if (editingExam) {
        await updateExam(editingExam.id, payload, token);
        toast.success('考试已更新');
      } else {
        await createExam(payload, token);
        toast.success('考试已创建');
      }
      setModalOpen(false);
      await loadExams();
    } catch (error) {
      toast.error(error.message || '保存失败');
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (exam) => {
    if (!window.confirm(`确认删除考试"${exam.name}"？`)) {
      return;
    }
    try {
      await deleteExam(exam.id, token);
      toast.success('考试已删除');
      await loadExams();
    } catch (error) {
      toast.error(error.message || '删除失败');
    }
  };

  const handleViewParticipants = async (exam) => {
    setSelectedExam(exam);
    setParticipantsLoading(true);
    try {
      const data = await getExamParticipants(exam.id, token);
      setParticipants(data);
    } catch (error) {
      toast.error(error.message || '加载参与名单失败');
    } finally {
      setParticipantsLoading(false);
    }
  };

  const toggleClass = (classId) => {
    const id = Number(classId);
    setFormData((prev) => {
      if (prev.classIds.includes(id)) {
        return { ...prev, classIds: prev.classIds.filter((cid) => cid !== id) };
      }
      return { ...prev, classIds: [...prev.classIds, id] };
    });
  };

  const calendarDays = useMemo(() => {
    const { year, month } = currentMonth;
    const firstDay = new Date(year, month, 1);
    const lastDay = new Date(year, month + 1, 0);
    const days = [];
    const startWeekDay = firstDay.getDay();

    for (let i = 0; i < startWeekDay; i++) {
      days.push(null);
    }
    for (let i = 1; i <= lastDay.getDate(); i++) {
      days.push(new Date(year, month, i));
    }
    return days;
  }, [currentMonth]);

  const examsByDay = useMemo(() => {
    const map = {};
    exams.forEach((exam) => {
      const start = new Date(exam.startTime);
      const dateStr = start.toDateString();
      if (!map[dateStr]) {
        map[dateStr] = [];
      }
      map[dateStr].push(exam);
    });
    return map;
  }, [exams]);

  const prevMonth = () => {
    setCurrentMonth((prev) => {
      const d = new Date(prev.year, prev.month - 1, 1);
      return { year: d.getFullYear(), month: d.getMonth() };
    });
  };

  const nextMonth = () => {
    setCurrentMonth((prev) => {
      const d = new Date(prev.year, prev.month + 1, 1);
      return { year: d.getFullYear(), month: d.getMonth() };
    });
  };

  const getClassNames = (classIdsStr) => {
    const ids = parseClassIds(classIdsStr);
    return ids
      .map((id) => classes.find((c) => c.id === id)?.name)
      .filter(Boolean)
      .join('、');
  };

  return (
    <div className="min-h-screen bg-board px-4 py-6 md:px-8 md:py-8">
      <header className="mx-auto mb-6 flex max-w-7xl flex-col gap-3 rounded-3xl border border-white/70 bg-white/90 px-6 py-5 shadow-card md:flex-row md:items-center md:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.25em] text-amber-700">Exam Management</p>
          <h1 className="mt-1 text-2xl font-bold text-slate-800">考试排考管理</h1>
          <p className="text-sm text-slate-600">创建和管理考试场次，支持日历视图查看排考安排。</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <button className="btn btn-ghost" onClick={onLogout}>
            退出登录
          </button>
          <button className="btn btn-primary" onClick={openCreate}>
            + 新建考试
          </button>
        </div>
      </header>

      <div className="mx-auto max-w-7xl">
        <div className="mb-4 inline-flex rounded-xl bg-white/80 p-1 shadow-sm">
          <button
            className={`btn btn-sm ${viewMode === 'list' ? 'btn-primary' : 'btn-ghost'}`}
            onClick={() => setViewMode('list')}
          >
            列表视图
          </button>
          <button
            className={`btn btn-sm ${viewMode === 'calendar' ? 'btn-primary' : 'btn-ghost'}`}
            onClick={() => setViewMode('calendar')}
          >
            日历视图
          </button>
        </div>

        {loading ? (
          <div className="animate-pulse rounded-2xl bg-white/80 p-8">
            <div className="h-6 w-1/3 rounded bg-slate-200" />
            <div className="mt-4 space-y-3">
              {[1, 2, 3].map((i) => (
                <div key={i} className="h-16 rounded-lg bg-slate-100" />
              ))}
            </div>
          </div>
        ) : viewMode === 'list' ? (
          <div className="overflow-hidden rounded-2xl border border-white/70 bg-white/90 shadow-card">
            <table className="table w-full">
              <thead>
                <tr>
                  <th>考试名称</th>
                  <th>开始时间</th>
                  <th>结束时间</th>
                  <th>时长</th>
                  <th>参与班级</th>
                  <th>状态</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                {exams.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="text-center text-slate-500">
                      暂无考试，点击右上角"新建考试"创建第一场考试
                    </td>
                  </tr>
                ) : (
                  exams.map((exam) => {
                    const status = statusMap[exam.status] || statusMap.pending;
                    return (
                      <tr key={exam.id}>
                        <td className="font-medium">{exam.name}</td>
                        <td>{formatDateTime(exam.startTime)}</td>
                        <td>{formatDateTime(exam.endTime)}</td>
                        <td>{exam.duration} 分钟</td>
                        <td>{getClassNames(exam.classIds)}</td>
                        <td>
                          <span className={status.className}>{status.label}</span>
                        </td>
                        <td>
                          <div className="flex gap-1">
                            <button
                              className="btn btn-xs btn-ghost"
                              onClick={() => handleViewParticipants(exam)}
                            >
                              参与名单
                            </button>
                            {exam.status !== 'ongoing' && (
                              <>
                                <button
                                  className="btn btn-xs btn-ghost"
                                  onClick={() => openEdit(exam)}
                                >
                                  编辑
                                </button>
                                <button
                                  className="btn btn-xs btn-ghost text-red-500"
                                  onClick={() => handleDelete(exam)}
                                >
                                  删除
                                </button>
                              </>
                            )}
                          </div>
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="rounded-2xl border border-white/70 bg-white/90 p-4 shadow-card">
            <div className="mb-4 flex items-center justify-between">
              <button className="btn btn-sm btn-ghost" onClick={prevMonth}>
                ← 上月
              </button>
              <h3 className="text-lg font-semibold">
                {currentMonth.year}年{currentMonth.month + 1}月
              </h3>
              <button className="btn btn-sm btn-ghost" onClick={nextMonth}>
                下月 →
              </button>
            </div>
            <div className="grid grid-cols-7 gap-1 text-center text-sm">
              {['日', '一', '二', '三', '四', '五', '六'].map((day) => (
                <div key={day} className="py-2 font-medium text-slate-500">
                  {day}
                </div>
              ))}
              {calendarDays.map((day, idx) => {
                if (!day) {
                  return <div key={`empty-${idx}`} className="min-h-[100px] py-2" />;
                }
                const dateStr = day.toDateString();
                const dayExams = examsByDay[dateStr] || [];
                const isToday = day.toDateString() === new Date().toDateString();
                return (
                  <div
                    key={idx}
                    className={`min-h-[100px] rounded-lg p-2 text-left ${
                      isToday ? 'bg-amber-50 ring-1 ring-amber-200' : 'bg-slate-50/50'
                    }`}
                  >
                    <div className={`text-sm font-medium ${isToday ? 'text-amber-700' : 'text-slate-700'}`}>
                      {day.getDate()}
                    </div>
                    <div className="mt-1 space-y-1">
                      {dayExams.slice(0, 2).map((exam) => (
                        <div
                          key={exam.id}
                          className="truncate rounded bg-blue-100 px-1.5 py-0.5 text-xs text-blue-700"
                          title={exam.name}
                        >
                          {new Date(exam.startTime).getHours()}:
                          {String(new Date(exam.startTime).getMinutes()).padStart(2, '0')} {exam.name}
                        </div>
                      ))}
                      {dayExams.length > 2 && (
                        <div className="text-xs text-slate-500">+{dayExams.length - 2} 场</div>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        )}
      </div>

      {modalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
          <div className="w-full max-w-lg rounded-2xl bg-white p-6 shadow-xl">
            <h3 className="mb-4 text-xl font-bold text-slate-800">
              {editingExam ? '编辑考试' : '新建考试'}
            </h3>
            <div className="space-y-4">
              <div>
                <label className="mb-1 block text-sm font-medium text-slate-700">考试名称</label>
                <input
                  type="text"
                  className="input input-bordered w-full"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  placeholder="请输入考试名称"
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="mb-1 block text-sm font-medium text-slate-700">开始时间</label>
                  <input
                    type="datetime-local"
                    className="input input-bordered w-full"
                    value={formData.startTime}
                    onChange={(e) => setFormData({ ...formData, startTime: e.target.value })}
                  />
                </div>
                <div>
                  <label className="mb-1 block text-sm font-medium text-slate-700">结束时间</label>
                  <input
                    type="datetime-local"
                    className="input input-bordered w-full"
                    value={formData.endTime}
                    onChange={(e) => setFormData({ ...formData, endTime: e.target.value })}
                  />
                </div>
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium text-slate-700">
                  考试时长（分钟）
                </label>
                <input
                  type="number"
                  className="input input-bordered w-full"
                  value={formData.duration}
                  onChange={(e) => setFormData({ ...formData, duration: e.target.value })}
                  min="1"
                />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium text-slate-700">参与班级</label>
                <div className="flex flex-wrap gap-2">
                  {classes.map((cls) => (
                    <label
                      key={cls.id}
                      className={`cursor-pointer rounded-lg border px-3 py-2 text-sm ${
                        formData.classIds.includes(cls.id)
                          ? 'border-primary bg-primary/10 text-primary'
                          : 'border-slate-200 bg-slate-50 text-slate-700'
                      }`}
                    >
                      <input
                        type="checkbox"
                        className="hidden"
                        checked={formData.classIds.includes(cls.id)}
                        onChange={() => toggleClass(cls.id)}
                      />
                      {cls.name}
                    </label>
                  ))}
                </div>
              </div>
            </div>
            <div className="mt-6 flex justify-end gap-2">
              <button
                className="btn btn-ghost"
                onClick={() => setModalOpen(false)}
                disabled={saving}
              >
                取消
              </button>
              <button className="btn btn-primary" onClick={handleSave} disabled={saving}>
                {saving ? '保存中...' : '保存'}
              </button>
            </div>
          </div>
        </div>
      )}

      {selectedExam && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
          <div className="w-full max-w-2xl rounded-2xl bg-white p-6 shadow-xl">
            <div className="mb-4 flex items-center justify-between">
              <h3 className="text-xl font-bold text-slate-800">
                {selectedExam.name} - 参与名单
              </h3>
              <button
                className="btn btn-sm btn-ghost"
                onClick={() => {
                  setSelectedExam(null);
                  setParticipants([]);
                }}
              >
                关闭
              </button>
            </div>
            {participantsLoading ? (
              <div className="py-8 text-center text-slate-500">加载中...</div>
            ) : participants.length === 0 ? (
              <div className="py-8 text-center text-slate-500">暂无参与记录</div>
            ) : (
              <div className="max-h-96 overflow-auto">
                <table className="table w-full">
                  <thead>
                    <tr>
                      <th>学生</th>
                      <th>状态</th>
                      <th>得分</th>
                      <th>开始时间</th>
                      <th>提交时间</th>
                    </tr>
                  </thead>
                  <tbody>
                    {participants.map((p) => (
                      <tr key={p.id}>
                        <td>{p.student?.username || `学生${p.studentId}`}</td>
                        <td>
                          <span
                            className={`badge badge-outline ${
                              p.status === 'submitted'
                                ? 'badge-success'
                                : p.status === 'ongoing'
                                ? 'badge-info'
                                : 'badge-ghost'
                            }`}
                          >
                            {p.status === 'submitted'
                              ? '已交卷'
                              : p.status === 'ongoing'
                              ? '进行中'
                              : '未参加'}
                          </span>
                        </td>
                        <td>{p.score !== null && p.score !== undefined ? p.score : '-'}</td>
                        <td>{p.startedAt ? formatDateTime(p.startedAt) : '-'}</td>
                        <td>{p.submittedAt ? formatDateTime(p.submittedAt) : '-'}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
