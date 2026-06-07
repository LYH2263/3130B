import { useEffect, useState } from 'react';
import { toast } from 'react-hot-toast';

import {
  getSubjectiveQuestions,
  createSubjectiveQuestion,
  updateSubjectiveQuestion,
  deleteSubjectiveQuestion,
  getSubjectivePendingCount,
} from '../api/client';
import { SubjectiveQuestionEditor } from '../components/SubjectiveQuestionEditor';

export function TeacherSubjectivePage({ user, token, onLogout, onNavigateToGrading }) {
  const [questions, setQuestions] = useState([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingQuestion, setEditingQuestion] = useState(null);
  const [pendingCount, setPendingCount] = useState(0);

  const loadData = async () => {
    setLoading(true);
    try {
      const [questionData, pendingData] = await Promise.all([
        getSubjectiveQuestions(token),
        getSubjectivePendingCount(token),
      ]);
      setQuestions(questionData);
      setPendingCount(pendingData.count || 0);
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

  const handleSave = async (payload) => {
    try {
      setSaving(true);
      if (editingQuestion) {
        await updateSubjectiveQuestion(editingQuestion.id, payload, token);
        toast.success('题目已更新');
      } else {
        await createSubjectiveQuestion(payload, token);
        toast.success('题目已创建');
      }
      setModalOpen(false);
      setEditingQuestion(null);
      await loadData();
    } catch (error) {
      toast.error(error.message || '保存失败');
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (id) => {
    if (!window.confirm('确认删除该主观题？')) {
      return;
    }
    try {
      await deleteSubjectiveQuestion(id, token);
      toast.success('题目已删除');
      await loadData();
    } catch (error) {
      toast.error(error.message || '删除失败');
    }
  };

  const openCreate = () => {
    setEditingQuestion(null);
    setModalOpen(true);
  };

  const openEdit = (question) => {
    setEditingQuestion(question);
    setModalOpen(true);
  };

  const getStatusBadge = (status) => {
    if (status === 'active') {
      return <span className="badge badge-success badge-outline">启用</span>;
    }
    return <span className="badge badge-ghost badge-outline">停用</span>;
  };

  return (
    <div className="min-h-screen bg-board px-4 py-6 md:px-8 md:py-8">
      <header className="mx-auto mb-6 flex max-w-7xl flex-col gap-3 rounded-3xl border border-white/70 bg-white/90 px-6 py-5 shadow-card md:flex-row md:items-center md:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.25em] text-amber-700">Subjective Questions</p>
          <h1 className="mt-1 text-2xl font-bold text-slate-800">主观题管理</h1>
          <p className="text-sm text-slate-600">管理主观题题库，支持富文本题干和人工批改。</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <button className="btn btn-outline btn-secondary" onClick={loadData}>
            刷新
          </button>
          <button className="btn btn-outline btn-warning" onClick={onNavigateToGrading}>
            批改工作台
            {pendingCount > 0 && (
              <span className="badge badge-error badge-sm ml-1">{pendingCount}</span>
            )}
          </button>
          <button className="btn btn-neutral" onClick={onLogout}>
            退出登录
          </button>
        </div>
      </header>

      {loading ? (
        <div className="mx-auto max-w-7xl">
          <div className="h-64 animate-pulse rounded-2xl bg-white/80" />
        </div>
      ) : (
        <main className="mx-auto max-w-7xl">
          <article className="rounded-3xl border border-slate-200 bg-white p-5 shadow-card">
            <div className="mb-4 flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
              <div>
                <h2 className="text-lg font-semibold text-slate-800">题库列表</h2>
                <p className="text-sm text-slate-500">共 {questions.length} 道主观题</p>
              </div>
              <button className="btn btn-primary" onClick={openCreate}>
                新增主观题
              </button>
            </div>

            <div className="overflow-auto rounded-xl border border-slate-200">
              <table className="table table-sm">
                <thead>
                  <tr>
                    <th className="w-16">ID</th>
                    <th>题干</th>
                    <th className="w-24">满分</th>
                    <th className="w-24">状态</th>
                    <th className="w-28">创建人</th>
                    <th className="w-32">操作</th>
                  </tr>
                </thead>
                <tbody>
                  {questions.map((question) => (
                    <tr key={question.id}>
                      <td className="font-mono text-xs">{question.id}</td>
                      <td>
                        <div className="max-w-md">
                          <p className="truncate font-medium" title={question.title}>
                            {question.title}
                          </p>
                          {question.referenceAnswer && (
                            <p className="mt-1 truncate text-xs text-slate-400" title={question.referenceAnswer}>
                              参考答案：{question.referenceAnswer}
                            </p>
                          )}
                        </div>
                      </td>
                      <td>
                        <span className="badge badge-primary badge-outline">{question.fullScore} 分</span>
                      </td>
                      <td>{getStatusBadge(question.status)}</td>
                      <td className="text-sm text-slate-500">
                        {question.creator?.username || '-'}
                      </td>
                      <td>
                        <div className="flex gap-1">
                          <button
                            className="btn btn-xs btn-ghost"
                            onClick={() => openEdit(question)}
                          >
                            编辑
                          </button>
                          <button
                            className="btn btn-xs btn-ghost text-error"
                            onClick={() => handleDelete(question.id)}
                          >
                            删除
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                  {!questions.length ? (
                    <tr>
                      <td colSpan={6} className="text-center text-slate-500 py-8">
                        暂无主观题，请点击"新增主观题"创建。
                      </td>
                    </tr>
                  ) : null}
                </tbody>
              </table>
            </div>
          </article>
        </main>
      )}

      <SubjectiveQuestionEditor
        open={modalOpen}
        initialData={editingQuestion}
        onClose={() => {
          setModalOpen(false);
          setEditingQuestion(null);
        }}
        onSubmit={handleSave}
        loading={saving}
      />
    </div>
  );
}
