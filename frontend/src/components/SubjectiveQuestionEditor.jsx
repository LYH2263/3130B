import { useEffect, useState } from 'react';

function buildInitialState(data) {
  if (!data) {
    return {
      title: '',
      referenceAnswer: '',
      fullScore: 10,
      status: 'active',
    };
  }

  return {
    title: data.title || '',
    referenceAnswer: data.referenceAnswer || '',
    fullScore: data.fullScore || 10,
    status: data.status || 'active',
  };
}

export function SubjectiveQuestionEditor({ open, initialData, onClose, onSubmit, loading }) {
  const [form, setForm] = useState(buildInitialState(initialData));

  useEffect(() => {
    if (open) {
      setForm(buildInitialState(initialData));
    }
  }, [open, initialData]);

  if (!open) {
    return null;
  }

  const handleSubmit = (event) => {
    event.preventDefault();
    const fullScore = parseFloat(form.fullScore);
    if (isNaN(fullScore) || fullScore <= 0) {
      return;
    }
    onSubmit({
      title: form.title,
      referenceAnswer: form.referenceAnswer,
      fullScore: fullScore,
      status: form.status,
    });
  };

  return (
    <div className="fixed inset-0 z-40 flex items-center justify-center bg-slate-900/50 px-4 py-8">
      <div className="w-full max-w-2xl rounded-2xl bg-base-100 p-6 shadow-2xl max-h-[90vh] overflow-y-auto">
        <div className="mb-4 flex items-center justify-between">
          <h3 className="text-xl font-semibold text-slate-800">{initialData ? '编辑主观题' : '新增主观题'}</h3>
          <button type="button" className="btn btn-sm btn-ghost" onClick={onClose}>
            关闭
          </button>
        </div>

        <form className="space-y-4" onSubmit={handleSubmit}>
          <label className="form-control">
            <span className="label-text mb-1 text-sm font-medium">题干（支持富文本格式）</span>
            <textarea
              className="textarea textarea-bordered min-h-32"
              value={form.title}
              onChange={(event) => setForm((prev) => ({ ...prev, title: event.target.value }))}
              placeholder="请输入题干内容..."
              required
            />
          </label>

          <label className="form-control">
            <span className="label-text mb-1 text-sm font-medium">参考答案</span>
            <textarea
              className="textarea textarea-bordered min-h-24"
              value={form.referenceAnswer}
              onChange={(event) => setForm((prev) => ({ ...prev, referenceAnswer: event.target.value }))}
              placeholder="请输入参考答案..."
            />
          </label>

          <div className="grid grid-cols-2 gap-4">
            <label className="form-control">
              <span className="label-text mb-1 text-sm font-medium">满分值</span>
              <input
                type="number"
                step="0.5"
                min="0.5"
                className="input input-bordered"
                value={form.fullScore}
                onChange={(event) => setForm((prev) => ({ ...prev, fullScore: event.target.value }))}
                required
              />
            </label>

            <label className="form-control">
              <span className="label-text mb-1 text-sm font-medium">状态</span>
              <select
                className="select select-bordered"
                value={form.status}
                onChange={(event) => setForm((prev) => ({ ...prev, status: event.target.value }))}
              >
                <option value="active">启用</option>
                <option value="inactive">停用</option>
              </select>
            </label>
          </div>

          <div className="flex justify-end gap-2 pt-2">
            <button type="button" className="btn btn-ghost" onClick={onClose}>
              取消
            </button>
            <button type="submit" className="btn btn-primary" disabled={loading}>
              {loading ? '保存中...' : '保存题目'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
