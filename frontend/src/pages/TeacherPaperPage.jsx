import { useEffect, useState } from 'react';
import { toast } from 'react-hot-toast';

import {
  getPaperBlueprints,
  createPaperBlueprint,
  updatePaperBlueprint,
  deletePaperBlueprint,
  generatePaper,
  replacePaperQuestion,
  savePaperSnapshot,
  getPaperSnapshots,
  deletePaperSnapshot,
  getKnowledgeTags,
} from '../api/client';

const difficultyOptions = [
  { value: 'easy', label: '易', color: 'success' },
  { value: 'medium', label: '中', color: 'warning' },
  { value: 'hard', label: '难', color: 'error' },
];

const questionTypeOptions = [
  { value: 'single_choice', label: '单选题' },
  { value: 'multiple_choice', label: '多选题' },
  { value: 'true_false', label: '判断题' },
];

function getDifficultyBadge(diff) {
  const map = {
    easy: 'badge badge-success badge-outline',
    medium: 'badge badge-warning badge-outline',
    hard: 'badge badge-error badge-outline',
  };
  return map[diff] || 'badge badge-ghost badge-outline';
}

function getDifficultyLabel(diff) {
  const map = { easy: '易', medium: '中', hard: '难' };
  return map[diff] || diff;
}

function getQuestionTypeLabel(type) {
  const map = { single_choice: '单选题', multiple_choice: '多选题', true_false: '判断题' };
  return map[type] || type;
}

export function TeacherPaperPage({ user, token, onLogout }) {
  const [activeTab, setActiveTab] = useState('blueprints');
  const [blueprints, setBlueprints] = useState([]);
  const [snapshots, setSnapshots] = useState([]);
  const [loading, setLoading] = useState(false);
  const [knowledgeTags, setKnowledgeTags] = useState([]);

  const [viewMode, setViewMode] = useState('list');
  const [editingBlueprint, setEditingBlueprint] = useState(null);
  const [generatedPaper, setGeneratedPaper] = useState(null);
  const [generating, setGenerating] = useState(false);
  const [saving, setSaving] = useState(false);

  const [formData, setFormData] = useState({
    name: '',
    description: '',
    totalScore: 100,
    avoidRepeatDays: 0,
    totalQuestions: 10,
    perQuestionScore: 10,
    difficulty: [
      { level: 'easy', count: 3, ratio: 0.3 },
      { level: 'medium', count: 5, ratio: 0.5 },
      { level: 'hard', count: 2, ratio: 0.2 },
    ],
    questionTypes: [
      { type: 'single_choice', count: 7, ratio: 0.7 },
      { type: 'multiple_choice', count: 2, ratio: 0.2 },
      { type: 'true_false', count: 1, ratio: 0.1 },
    ],
    knowledgeTags: [],
  });

  const [savePaperName, setSavePaperName] = useState('');
  const [savePaperDesc, setSavePaperDesc] = useState('');
  const [showSaveModal, setShowSaveModal] = useState(false);

  const loadBlueprints = async () => {
    setLoading(true);
    try {
      const data = await getPaperBlueprints({ pageSize: 100 }, token);
      setBlueprints(data.items || []);
    } catch (error) {
      toast.error(error.message || '加载蓝图失败');
    } finally {
      setLoading(false);
    }
  };

  const loadSnapshots = async () => {
    setLoading(true);
    try {
      const data = await getPaperSnapshots({ pageSize: 100 }, token);
      setSnapshots(data.items || []);
    } catch (error) {
      toast.error(error.message || '加载试卷失败');
    } finally {
      setLoading(false);
    }
  };

  const loadKnowledgeTags = async () => {
    try {
      const data = await getKnowledgeTags(token);
      setKnowledgeTags(data || []);
    } catch (error) {
      console.warn('加载知识点标签失败', error);
    }
  };

  useEffect(() => {
    loadBlueprints();
    loadKnowledgeTags();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  useEffect(() => {
    if (activeTab === 'snapshots') {
      loadSnapshots();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeTab]);

  const openCreate = () => {
    setEditingBlueprint(null);
    setFormData({
      name: '',
      description: '',
      totalScore: 100,
      avoidRepeatDays: 0,
      totalQuestions: 10,
      perQuestionScore: 10,
      difficulty: [
        { level: 'easy', count: 3, ratio: 0.3 },
        { level: 'medium', count: 5, ratio: 0.5 },
        { level: 'hard', count: 2, ratio: 0.2 },
      ],
      questionTypes: [
        { type: 'single_choice', count: 7, ratio: 0.7 },
        { type: 'multiple_choice', count: 2, ratio: 0.2 },
        { type: 'true_false', count: 1, ratio: 0.1 },
      ],
      knowledgeTags: [],
    });
    setViewMode('editor');
  };

  const openEdit = (bp) => {
    setEditingBlueprint(bp);
    const rule = bp.rule || {};
    setFormData({
      name: bp.name,
      description: bp.description || '',
      totalScore: bp.totalScore,
      avoidRepeatDays: bp.avoidRepeatDays || 0,
      totalQuestions: rule.totalQuestions || 10,
      perQuestionScore: rule.perQuestionScore || 10,
      difficulty: rule.difficulty && rule.difficulty.length > 0
        ? rule.difficulty
        : [
            { level: 'easy', count: 3, ratio: 0.3 },
            { level: 'medium', count: 5, ratio: 0.5 },
            { level: 'hard', count: 2, ratio: 0.2 },
          ],
      questionTypes: rule.questionTypes && rule.questionTypes.length > 0
        ? rule.questionTypes
        : [
            { type: 'single_choice', count: 7, ratio: 0.7 },
            { type: 'multiple_choice', count: 2, ratio: 0.2 },
            { type: 'true_false', count: 1, ratio: 0.1 },
          ],
      knowledgeTags: rule.knowledgeTags || [],
    });
    setViewMode('editor');
  };

  const handleSaveBlueprint = async () => {
    if (!formData.name.trim()) {
      toast.error('请输入蓝图名称');
      return;
    }
    if (formData.totalQuestions <= 0) {
      toast.error('总题数必须大于0');
      return;
    }

    const totalDiff = formData.difficulty.reduce((sum, d) => sum + (d.count || 0), 0);
    const totalType = formData.questionTypes.reduce((sum, t) => sum + (t.count || 0), 0);

    if (totalDiff > 0 && totalDiff !== formData.totalQuestions) {
      toast.error('难度配置题数之和必须等于总题数');
      return;
    }

    const payload = {
      name: formData.name.trim(),
      description: formData.description.trim(),
      totalScore: Number(formData.totalScore),
      avoidRepeatDays: Number(formData.avoidRepeatDays),
      rule: {
        totalQuestions: Number(formData.totalQuestions),
        perQuestionScore: Number(formData.perQuestionScore),
        difficulty: formData.difficulty.map((d) => ({
          ...d,
          count: Number(d.count) || 0,
          ratio: formData.totalQuestions > 0 ? (Number(d.count) || 0) / formData.totalQuestions : 0,
        })),
        questionTypes: formData.questionTypes.map((t) => ({
          ...t,
          count: Number(t.count) || 0,
          ratio: formData.totalQuestions > 0 ? (Number(t.count) || 0) / formData.totalQuestions : 0,
        })),
        knowledgeTags: formData.knowledgeTags.map((k) => ({
          ...k,
          count: Number(k.count) || 0,
          ratio: formData.totalQuestions > 0 ? (Number(k.count) || 0) / formData.totalQuestions : 0,
        })),
      },
    };

    try {
      setSaving(true);
      if (editingBlueprint) {
        await updatePaperBlueprint(editingBlueprint.id, payload, token);
        toast.success('蓝图已更新');
      } else {
        await createPaperBlueprint(payload, token);
        toast.success('蓝图已创建');
      }
      setViewMode('list');
      await loadBlueprints();
    } catch (error) {
      toast.error(error.message || '保存失败');
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (bp) => {
    if (!window.confirm(`确定要删除蓝图「${bp.name}」吗？`)) return;
    try {
      await deletePaperBlueprint(bp.id, token);
      toast.success('蓝图已删除');
      await loadBlueprints();
    } catch (error) {
      toast.error(error.message || '删除失败');
    }
  };

  const handleDeleteSnapshot = async (snap) => {
    if (!window.confirm(`确定要删除试卷「${snap.name}」吗？`)) return;
    try {
      await deletePaperSnapshot(snap.id, token);
      toast.success('试卷已删除');
      await loadSnapshots();
    } catch (error) {
      toast.error(error.message || '删除失败');
    }
  };

  const handleGenerate = async (bp) => {
    try {
      setGenerating(true);
      const result = await generatePaper(bp.id, token);
      if (result.gapReport && !result.gapReport.canGenerate) {
        toast.error('题量不足，无法生成试卷');
        return;
      }
      setGeneratedPaper(result);
      setViewMode('preview');
    } catch (error) {
      toast.error(error.message || '生成失败');
    } finally {
      setGenerating(false);
    }
  };

  const handleReplaceQuestion = async (index) => {
    if (!generatedPaper) return;
    const currentQ = generatedPaper.questions[index];
    try {
      const replacement = await replacePaperQuestion(
        generatedPaper.blueprintId,
        currentQ.questionId,
        token
      );
      const newQuestions = [...generatedPaper.questions];
      newQuestions[index] = replacement;
      setGeneratedPaper({ ...generatedPaper, questions: newQuestions });
      toast.success('已替换题目');
    } catch (error) {
      toast.error(error.message || '替换失败');
    }
  };

  const handleSavePaper = async () => {
    if (!savePaperName.trim()) {
      toast.error('请输入试卷名称');
      return;
    }
    try {
      setSaving(true);
      await savePaperSnapshot(
        {
          blueprintId: generatedPaper.blueprintId,
          questions: generatedPaper.questions,
          name: savePaperName.trim(),
          description: savePaperDesc.trim(),
          status: 'published',
        },
        token
      );
      toast.success('试卷已保存');
      setShowSaveModal(false);
      setViewMode('list');
      setActiveTab('snapshots');
    } catch (error) {
      toast.error(error.message || '保存失败');
    } finally {
      setSaving(false);
    }
  };

  const updateDifficultyCount = (level, count) => {
    const val = Math.max(0, parseInt(count) || 0);
    setFormData((prev) => ({
      ...prev,
      difficulty: prev.difficulty.map((d) =>
        d.level === level ? { ...d, count: val } : d
      ),
    }));
  };

  const updateQuestionTypeCount = (type, count) => {
    const val = Math.max(0, parseInt(count) || 0);
    setFormData((prev) => ({
      ...prev,
      questionTypes: prev.questionTypes.map((t) =>
        t.type === type ? { ...t, count: val } : t
      ),
    }));
  };

  const addKnowledgeTag = () => {
    setFormData((prev) => ({
      ...prev,
      knowledgeTags: [...prev.knowledgeTags, { tag: '', count: 0, ratio: 0 }],
    }));
  };

  const updateKnowledgeTag = (index, field, value) => {
    setFormData((prev) => {
      const newTags = [...prev.knowledgeTags];
      if (field === 'count') {
        value = Math.max(0, parseInt(value) || 0);
      }
      newTags[index] = { ...newTags[index], [field]: value };
      return { ...prev, knowledgeTags: newTags };
    });
  };

  const removeKnowledgeTag = (index) => {
    setFormData((prev) => ({
      ...prev,
      knowledgeTags: prev.knowledgeTags.filter((_, i) => i !== index),
    }));
  };

  const difficultyTotal = formData.difficulty.reduce((sum, d) => sum + (d.count || 0), 0);
  const typeTotal = formData.questionTypes.reduce((sum, t) => sum + (t.count || 0), 0);

  const renderBlueprintList = () => (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <h3 className="text-lg font-semibold text-slate-800">组卷蓝图</h3>
        <button className="btn btn-primary" onClick={openCreate}>
          + 新建蓝图
        </button>
      </div>

      {loading ? (
        <div className="py-8 text-center text-slate-500">加载中...</div>
      ) : blueprints.length === 0 ? (
        <div className="py-12 text-center text-slate-500">
          <div className="text-4xl mb-2">📋</div>
          <p>暂无组卷蓝图，点击上方按钮创建</p>
        </div>
      ) : (
        <div className="grid gap-3">
          {blueprints.map((bp) => {
            const rule = bp.rule || {};
            return (
              <div key={bp.id} className="card p-4">
                <div className="flex items-start justify-between">
                  <div>
                    <h4 className="font-medium text-slate-800">{bp.name}</h4>
                    <p className="text-sm text-slate-500 mt-1">
                      {bp.description || '暂无描述'}
                    </p>
                    <div className="flex flex-wrap gap-2 mt-3">
                      <span className="badge badge-outline badge-info">
                        {rule.totalQuestions || 0} 题
                      </span>
                      <span className="badge badge-outline badge-success">
                        {bp.totalScore} 分
                      </span>
                      {bp.avoidRepeatDays > 0 && (
                        <span className="badge badge-outline badge-warning">
                          避免 {bp.avoidRepeatDays} 天内不重复
                        </span>
                      )}
                      {rule.difficulty && rule.difficulty.map((d) => (
                        <span
                          key={d.level}
                          className={`badge ${getDifficultyBadge(d.level)}`}
                        >
                          {getDifficultyLabel(d.level)}: {d.count}题
                        </span>
                      ))}
                    </div>
                  </div>
                  <div className="flex gap-2">
                    <button
                      className="btn btn-ghost btn-sm"
                      onClick={() => handleGenerate(bp)}
                      disabled={generating}
                    >
                      生成试卷
                    </button>
                    <button
                      className="btn btn-ghost btn-sm"
                      onClick={() => openEdit(bp)}
                    >
                      编辑
                    </button>
                    <button
                      className="btn btn-ghost btn-sm text-red-500"
                      onClick={() => handleDelete(bp)}
                    >
                      删除
                    </button>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );

  const renderEditor = () => (
    <div>
      <div className="mb-4 flex items-center gap-2">
        <button className="btn btn-ghost btn-sm" onClick={() => setViewMode('list')}>
          ← 返回列表
        </button>
        <h3 className="text-lg font-semibold text-slate-800">
          {editingBlueprint ? '编辑蓝图' : '新建蓝图'}
        </h3>
      </div>

      <div className="card p-6 space-y-6">
        <div>
          <label className="label">蓝图名称</label>
          <input
            type="text"
            className="input input-bordered w-full"
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            placeholder="请输入蓝图名称"
          />
        </div>

        <div>
          <label className="label">描述</label>
          <textarea
            className="textarea textarea-bordered w-full"
            value={formData.description}
            onChange={(e) => setFormData({ ...formData, description: e.target.value })}
            placeholder="蓝图描述（可选）"
            rows={2}
          />
        </div>

        <div className="grid grid-cols-3 gap-4">
          <div>
            <label className="label">总分</label>
            <input
              type="number"
              className="input input-bordered w-full"
              value={formData.totalScore}
              onChange={(e) => setFormData({ ...formData, totalScore: e.target.value })}
              min={1}
            />
          </div>
          <div>
            <label className="label">总题数</label>
            <input
              type="number"
              className="input input-bordered w-full"
              value={formData.totalQuestions}
              onChange={(e) => setFormData({ ...formData, totalQuestions: e.target.value })}
              min={1}
            />
          </div>
          <div>
            <label className="label">每题分值</label>
            <input
              type="number"
              className="input input-bordered w-full"
              value={formData.perQuestionScore}
              onChange={(e) => setFormData({ ...formData, perQuestionScore: e.target.value })}
              min={1}
            />
          </div>
        </div>

        <div>
          <label className="label">避免最近 N 天重复题</label>
          <input
            type="number"
            className="input input-bordered w-full max-w-xs"
            value={formData.avoidRepeatDays}
            onChange={(e) => setFormData({ ...formData, avoidRepeatDays: e.target.value })}
            min={0}
          />
          <span className="text-xs text-slate-500 ml-2">0 表示不限制</span>
        </div>

        <div>
          <label className="label">难度分布</label>
          <div className="grid grid-cols-3 gap-4">
            {formData.difficulty.map((d) => (
              <div key={d.level} className="form-control">
                <label className="label text-sm">
                  {getDifficultyLabel(d.level)}
                </label>
                <input
                  type="number"
                  className="input input-bordered"
                  value={d.count}
                  onChange={(e) => updateDifficultyCount(d.level, e.target.value)}
                  min={0}
                />
                <span className="text-xs text-slate-500 mt-1">
                  {formData.totalQuestions > 0
                    ? ((d.count / formData.totalQuestions) * 100).toFixed(0) + '%'
                    : '0%'}
                </span>
              </div>
            ))}
          </div>
          <div className="mt-2 text-sm">
            <span className={difficultyTotal !== formData.totalQuestions ? 'text-red-500' : 'text-green-500'}>
              已配置 {difficultyTotal} / {formData.totalQuestions} 题
            </span>
          </div>
        </div>

        <div>
          <label className="label">题型分布</label>
          <div className="grid grid-cols-3 gap-4">
            {formData.questionTypes.map((t) => (
              <div key={t.type} className="form-control">
                <label className="label text-sm">{getQuestionTypeLabel(t.type)}</label>
                <input
                  type="number"
                  className="input input-bordered"
                  value={t.count}
                  onChange={(e) => updateQuestionTypeCount(t.type, e.target.value)}
                  min={0}
                />
                <span className="text-xs text-slate-500 mt-1">
                  {formData.totalQuestions > 0
                    ? ((t.count / formData.totalQuestions) * 100).toFixed(0) + '%'
                    : '0%'}
                </span>
              </div>
            ))}
          </div>
          <div className="mt-2 text-sm">
            <span className={typeTotal !== formData.totalQuestions ? 'text-red-500' : 'text-green-500'}>
              已配置 {typeTotal} / {formData.totalQuestions} 题
            </span>
          </div>
        </div>

        <div>
          <div className="flex items-center justify-between mb-2">
            <label className="label mb-0">知识点分布</label>
            <button className="btn btn-ghost btn-sm" onClick={addKnowledgeTag}>
              + 添加知识点
            </button>
          </div>
          {formData.knowledgeTags.length === 0 ? (
            <p className="text-sm text-slate-500">暂未配置知识点约束</p>
          ) : (
            <div className="space-y-2">
              {formData.knowledgeTags.map((kt, index) => (
                <div key={index} className="flex items-center gap-2">
                  <select
                    className="select select-bordered flex-1"
                    value={kt.tag}
                    onChange={(e) => updateKnowledgeTag(index, 'tag', e.target.value)}
                  >
                    <option value="">请选择知识点</option>
                    {knowledgeTags.map((tagOpt) => (
                      <option key={tagOpt.tag} value={tagOpt.tag}>
                        {tagOpt.tag} ({tagOpt.count}题)
                      </option>
                    ))}
                  </select>
                  <input
                    type="number"
                    className="input input-bordered w-24"
                    value={kt.count}
                    onChange={(e) => updateKnowledgeTag(index, 'count', e.target.value)}
                    min={0}
                    placeholder="题数"
                  />
                  <button
                    className="btn btn-ghost btn-sm text-red-500"
                    onClick={() => removeKnowledgeTag(index)}
                  >
                    删除
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="flex justify-end gap-3 pt-4 border-t">
          <button className="btn btn-ghost" onClick={() => setViewMode('list')}>
            取消
          </button>
          <button className="btn btn-primary" onClick={handleSaveBlueprint} disabled={saving}>
            {saving ? '保存中...' : '保存蓝图'}
          </button>
        </div>
      </div>
    </div>
  );

  const renderPreview = () => {
    if (!generatedPaper) return null;

    return (
      <div>
        <div className="mb-4 flex items-center gap-2">
          <button className="btn btn-ghost btn-sm" onClick={() => setViewMode('list')}>
            ← 返回列表
          </button>
          <h3 className="text-lg font-semibold text-slate-800">试卷预览</h3>
          <div className="ml-auto flex gap-2">
            <button
              className="btn btn-primary btn-sm"
              onClick={() => setShowSaveModal(true)}
            >
              保存试卷
            </button>
          </div>
        </div>

        <div className="card p-4 mb-4">
          <div className="flex items-center gap-4 flex-wrap">
            <span className="font-medium">共 {generatedPaper.questions.length} 题</span>
            <span className="text-slate-500">总分: {generatedPaper.totalScore} 分</span>
            {generatedPaper.gapReport && generatedPaper.gapReport.messages && generatedPaper.gapReport.messages.length > 0 && (
              <div className="w-full mt-2 p-3 bg-amber-50 border border-amber-200 rounded-lg">
                <p className="text-amber-800 font-medium text-sm mb-1">提示：</p>
                <ul className="text-sm text-amber-700 list-disc list-inside">
                  {generatedPaper.gapReport.messages.map((msg, i) => (
                    <li key={i}>{msg}</li>
                  ))}
                </ul>
              </div>
            )}
          </div>
        </div>

        <div className="space-y-3">
          {generatedPaper.questions.map((q, index) => (
            <div key={q.questionId + '-' + index} className="card p-4">
              <div className="flex items-start justify-between gap-4">
                <div className="flex-1">
                  <div className="flex items-center gap-2 mb-2">
                    <span className="font-medium text-slate-700">
                      第 {index + 1} 题
                    </span>
                    <span className={`badge ${getDifficultyBadge(q.difficulty)} badge-sm`}>
                      {getDifficultyLabel(q.difficulty)}
                    </span>
                    <span className="badge badge-outline badge-sm">
                      {getQuestionTypeLabel(q.questionType)}
                    </span>
                    <span className="text-sm text-slate-500">{q.score} 分</span>
                  </div>
                  <p className="text-slate-800 font-medium mb-2">{q.title}</p>
                  {q.description && (
                    <p className="text-sm text-slate-500 mb-2">{q.description}</p>
                  )}
                  {q.knowledgeTags && (
                    <div className="flex flex-wrap gap-1 mb-2">
                      {q.knowledgeTags.split(',').map((tag, i) => (
                        <span key={i} className="badge badge-xs badge-ghost">
                          {tag.trim()}
                        </span>
                      ))}
                    </div>
                  )}
                  <div className="space-y-1 ml-4">
                    {q.options.map((opt, optIdx) => (
                      <div key={opt.id} className="text-sm text-slate-600">
                        {String.fromCharCode(65 + optIdx)}. {opt.content}
                      </div>
                    ))}
                  </div>
                </div>
                <button
                  className="btn btn-ghost btn-sm shrink-0"
                  onClick={() => handleReplaceQuestion(index)}
                >
                  换一题
                </button>
              </div>
            </div>
          ))}
        </div>

        {showSaveModal && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
            <div className="card p-6 w-full max-w-md bg-white">
              <h3 className="text-lg font-semibold mb-4">保存试卷</h3>
              <div className="space-y-4">
                <div>
                  <label className="label">试卷名称</label>
                  <input
                    type="text"
                    className="input input-bordered w-full"
                    value={savePaperName}
                    onChange={(e) => setSavePaperName(e.target.value)}
                    placeholder="请输入试卷名称"
                  />
                </div>
                <div>
                  <label className="label">描述</label>
                  <textarea
                    className="textarea textarea-bordered w-full"
                    value={savePaperDesc}
                    onChange={(e) => setSavePaperDesc(e.target.value)}
                    placeholder="试卷描述（可选）"
                    rows={3}
                  />
                </div>
              </div>
              <div className="flex justify-end gap-3 mt-6">
                <button
                  className="btn btn-ghost"
                  onClick={() => setShowSaveModal(false)}
                >
                  取消
                </button>
                <button
                  className="btn btn-primary"
                  onClick={handleSavePaper}
                  disabled={saving}
                >
                  {saving ? '保存中...' : '保存'}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    );
  };

  const renderSnapshotList = () => (
    <div>
      <div className="mb-4">
        <h3 className="text-lg font-semibold text-slate-800">已生成试卷</h3>
      </div>

      {loading ? (
        <div className="py-8 text-center text-slate-500">加载中...</div>
      ) : snapshots.length === 0 ? (
        <div className="py-12 text-center text-slate-500">
          <div className="text-4xl mb-2">📄</div>
          <p>暂无已保存的试卷</p>
          <p className="text-sm mt-1">
            通过蓝图生成试卷后可保存到这里
          </p>
        </div>
      ) : (
        <div className="grid gap-3">
          {snapshots.map((snap) => (
            <div key={snap.id} className="card p-4">
              <div className="flex items-start justify-between">
                <div>
                  <h4 className="font-medium text-slate-800">{snap.name}</h4>
                  <p className="text-sm text-slate-500 mt-1">
                    {snap.description || '暂无描述'}
                  </p>
                  <div className="flex flex-wrap gap-2 mt-3">
                    <span className="badge badge-outline badge-info">
                      {snap.totalQuestions} 题
                    </span>
                    <span className="badge badge-outline badge-success">
                      {snap.totalScore} 分
                    </span>
                    <span
                      className={`badge ${
                        snap.status === 'published'
                          ? 'badge-success'
                          : 'badge-ghost'
                      } badge-outline`}
                    >
                      {snap.status === 'published' ? '已发布' : '草稿'}
                    </span>
                    {snap.blueprint && (
                      <span className="badge badge-ghost badge-outline">
                        来自蓝图: {snap.blueprint.name}
                      </span>
                    )}
                  </div>
                </div>
                <div className="flex gap-2">
                  <button
                    className="btn btn-ghost btn-sm text-red-500"
                    onClick={() => handleDeleteSnapshot(snap)}
                  >
                    删除
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );

  return (
    <div className="min-h-screen bg-board">
      <div className="mx-auto max-w-5xl px-4 py-6">
        <div className="mb-6 flex items-center justify-between">
          <div>
            <h2 className="text-2xl font-bold text-slate-800">自动组卷</h2>
            <p className="text-sm text-slate-500 mt-1">
              配置组卷规则，智能生成试卷
            </p>
          </div>
          <button className="btn btn-ghost" onClick={onLogout}>
            退出登录
          </button>
        </div>

        <div className="tabs mb-6">
          <button
            className={`tab ${activeTab === 'blueprints' ? 'tab-active' : ''}`}
            onClick={() => setActiveTab('blueprints')}
          >
            组卷蓝图
          </button>
          <button
            className={`tab ${activeTab === 'snapshots' ? 'tab-active' : ''}`}
            onClick={() => setActiveTab('snapshots')}
          >
            已生成试卷
          </button>
        </div>

        {activeTab === 'blueprints' && viewMode === 'list' && renderBlueprintList()}
        {activeTab === 'blueprints' && viewMode === 'editor' && renderEditor()}
        {activeTab === 'blueprints' && viewMode === 'preview' && renderPreview()}
        {activeTab === 'snapshots' && renderSnapshotList()}
      </div>
    </div>
  );
}
