import { useEffect, useState } from 'react';
import { toast } from 'react-hot-toast';

import {
  getQuestionVersions,
  diffQuestionVersions,
  rollbackQuestionVersion,
} from '../api/client';

export function QuestionVersionHistory({ questionId, token, onClose, onRollback }) {
  const [versions, setVersions] = useState([]);
  const [loading, setLoading] = useState(true);
  const [selectedVersion, setSelectedVersion] = useState(null);
  const [compareMode, setCompareMode] = useState(false);
  const [compareFrom, setCompareFrom] = useState(null);
  const [compareTo, setCompareTo] = useState(null);
  const [diffData, setDiffData] = useState(null);
  const [loadingDiff, setLoadingDiff] = useState(false);
  const [rollbackConfirm, setRollbackConfirm] = useState(null);
  const [rollingBack, setRollingBack] = useState(false);

  const loadVersions = async () => {
    if (!questionId) return;
    setLoading(true);
    try {
      const data = await getQuestionVersions(questionId, token);
      setVersions(data);
      if (data.length > 0) {
        setSelectedVersion(data[0]);
      }
    } catch (error) {
      toast.error(error.message || '加载版本列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadVersions();
  }, [questionId, token]);

  const handleRollback = async (version) => {
    setRollingBack(true);
    try {
      await rollbackQuestionVersion(questionId, version.id, token);
      toast.success(`已回滚到版本 v${version.versionNumber}`);
      setRollbackConfirm(null);
      await loadVersions();
      if (onRollback) {
        onRollback();
      }
    } catch (error) {
      toast.error(error.message || '回滚失败');
    } finally {
      setRollingBack(false);
    }
  };

  const handleCompare = async () => {
    if (!compareFrom || !compareTo) return;
    setLoadingDiff(true);
    try {
      const diff = await diffQuestionVersions(
        questionId,
        compareFrom.id,
        compareTo.id,
        token
      );
      setDiffData(diff);
    } catch (error) {
      toast.error(error.message || '对比失败');
    } finally {
      setLoadingDiff(false);
    }
  };

  useEffect(() => {
    if (compareFrom && compareTo && compareMode) {
      handleCompare();
    }
  }, [compareFrom, compareTo, compareMode]);

  const formatTime = (timeStr) => {
    if (!timeStr) return '';
    const d = new Date(timeStr);
    return d.toLocaleString('zh-CN', {
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  if (!questionId) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/50 px-4 py-8">
      <div className="flex h-[90vh] w-full max-w-5xl flex-col rounded-2xl bg-base-100 shadow-2xl">
        <div className="flex items-center justify-between border-b border-slate-200 px-6 py-4">
          <div>
            <h3 className="text-xl font-semibold text-slate-800">历史版本</h3>
            <p className="text-sm text-slate-500">题目 ID: {questionId}</p>
          </div>
          <div className="flex items-center gap-2">
            <button
              className={`btn btn-sm ${compareMode ? 'btn-primary' : 'btn-outline'}`}
              onClick={() => {
                setCompareMode(!compareMode);
                setCompareFrom(null);
                setCompareTo(null);
                setDiffData(null);
              }}
            >
              {compareMode ? '退出对比' : '版本对比'}
            </button>
            <button type="button" className="btn btn-sm btn-ghost" onClick={onClose}>
              关闭
            </button>
          </div>
        </div>

        <div className="flex flex-1 overflow-hidden">
          <div className="w-72 shrink-0 overflow-auto border-r border-slate-200">
            {loading ? (
              <div className="p-4 text-center text-slate-500">加载中...</div>
            ) : versions.length === 0 ? (
              <div className="p-4 text-center text-slate-500">暂无历史版本</div>
            ) : (
              <div className="relative py-4">
                <div className="absolute left-6 top-0 h-full w-px bg-slate-200" />
                {versions.map((version, index) => (
                  <div
                    key={version.id}
                    className={`relative mb-2 cursor-pointer pl-12 pr-4 ${
                      selectedVersion?.id === version.id ? 'bg-sky-50' : ''
                    } ${compareMode ? 'hover:bg-slate-50' : 'hover:bg-slate-50'}`}
                    onClick={() => {
                      if (compareMode) {
                        if (!compareFrom) {
                          setCompareFrom(version);
                        } else if (!compareTo && compareFrom.id !== version.id) {
                          setCompareTo(version);
                        } else {
                          setCompareFrom(version);
                          setCompareTo(null);
                        }
                      } else {
                        setSelectedVersion(version);
                      }
                    }}
                  >
                    <div
                      className={`absolute left-5 top-3 h-3 w-3 -translate-x-1/2 rounded-full border-2 ${
                        selectedVersion?.id === version.id
                          ? 'border-sky-500 bg-sky-500'
                          : compareFrom?.id === version.id
                          ? 'border-emerald-500 bg-emerald-500'
                          : compareTo?.id === version.id
                          ? 'border-amber-500 bg-amber-500'
                          : 'border-slate-300 bg-white'
                      }`}
                    />
                    <div className="py-2">
                      <div className="flex items-center justify-between">
                        <span className="font-medium text-slate-800">
                          v{version.versionNumber}
                        </span>
                        {index === 0 && (
                          <span className="badge badge-xs badge-primary">最新</span>
                        )}
                      </div>
                      <div className="text-xs text-slate-500">
                        {formatTime(version.createdAt)}
                      </div>
                      {version.modifier && (
                        <div className="text-xs text-slate-600">
                          {version.modifier.username}
                        </div>
                      )}
                      {version.changeNote && (
                        <div className="mt-1 text-xs text-slate-600 line-clamp-2">
                          {version.changeNote}
                        </div>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          <div className="flex flex-1 flex-col overflow-auto">
            {compareMode ? (
              <div className="p-4">
                {!compareFrom || !compareTo ? (
                  <div className="flex h-full items-center justify-center text-slate-500">
                    <div className="text-center">
                      <p className="mb-2">请选择两个版本进行对比</p>
                      <p className="text-xs">
                        {compareFrom
                          ? `已选: v${compareFrom.versionNumber} (旧版本)`
                          : '点击左侧版本作为旧版本'}
                      </p>
                      <p className="text-xs">
                        {compareTo
                          ? `已选: v${compareTo.versionNumber} (新版本)`
                          : '点击左侧版本作为新版本'}
                      </p>
                    </div>
                  </div>
                ) : loadingDiff ? (
                  <div className="flex h-full items-center justify-center text-slate-500">
                    对比中...
                  </div>
                ) : diffData ? (
                  <div className="space-y-4">
                    <div className="mb-4 flex items-center justify-between rounded-lg bg-slate-50 p-3">
                      <span className="text-sm text-slate-600">
                        对比: <span className="font-medium text-emerald-600">v{compareFrom.versionNumber}</span>
                        {' → '}
                        <span className="font-medium text-amber-600">v{compareTo.versionNumber}</span>
                      </span>
                    </div>

                    <div className="space-y-3">
                      <div
                        className={`rounded-lg border p-3 ${
                          diffData.title.changed ? 'border-amber-300 bg-amber-50' : 'border-slate-200'
                        }`}
                      >
                        <div className="mb-2 text-sm font-medium text-slate-700">题干</div>
                        {diffData.title.changed ? (
                          <div className="space-y-2 text-sm">
                            <div className="rounded bg-red-100 p-2 line-through text-red-700">
                              {diffData.title.oldValue}
                            </div>
                            <div className="rounded bg-green-100 p-2 text-green-700">
                              {diffData.title.newValue}
                            </div>
                          </div>
                        ) : (
                          <div className="text-sm text-slate-600">{diffData.title.oldValue}</div>
                        )}
                      </div>

                      <div
                        className={`rounded-lg border p-3 ${
                          diffData.description.changed
                            ? 'border-amber-300 bg-amber-50'
                            : 'border-slate-200'
                        }`}
                      >
                        <div className="mb-2 text-sm font-medium text-slate-700">知识点说明</div>
                        {diffData.description.changed ? (
                          <div className="space-y-2 text-sm">
                            <div className="rounded bg-red-100 p-2 line-through text-red-700">
                              {diffData.description.oldValue || '(空)'}
                            </div>
                            <div className="rounded bg-green-100 p-2 text-green-700">
                              {diffData.description.newValue || '(空)'}
                            </div>
                          </div>
                        ) : (
                          <div className="text-sm text-slate-600">
                            {diffData.description.oldValue || '(空)'}
                          </div>
                        )}
                      </div>

                      <div className="rounded-lg border border-slate-200 p-3">
                        <div className="mb-3 text-sm font-medium text-slate-700">选项</div>
                        <div className="space-y-2">
                          {diffData.options.map((opt, idx) => (
                            <div
                              key={idx}
                              className={`rounded-lg border p-3 ${
                                opt.status === 'added'
                                  ? 'border-green-300 bg-green-50'
                                  : opt.status === 'deleted'
                                  ? 'border-red-300 bg-red-50'
                                  : opt.status === 'modified'
                                  ? 'border-amber-300 bg-amber-50'
                                  : 'border-slate-200'
                              }`}
                            >
                              <div className="mb-1 flex items-center gap-2">
                                <span className="text-xs font-medium uppercase">
                                  {opt.status === 'added' && (
                                    <span className="text-green-600">新增</span>
                                  )}
                                  {opt.status === 'deleted' && (
                                    <span className="text-red-600">删除</span>
                                  )}
                                  {opt.status === 'modified' && (
                                    <span className="text-amber-600">修改</span>
                                  )}
                                  {opt.status === 'unchanged' && (
                                    <span className="text-slate-500">未变</span>
                                  )}
                                </span>
                                <span className="text-xs text-slate-500">
                                  选项 {opt.newIndex !== undefined ? opt.newIndex + 1 : opt.oldIndex + 1}
                                </span>
                              </div>
                              {opt.content.changed ? (
                                <div className="space-y-1 text-sm">
                                  {opt.content.oldValue !== undefined && (
                                    <div className="rounded bg-red-100 p-1.5 line-through text-red-700">
                                      {opt.content.oldValue}
                                    </div>
                                  )}
                                  {opt.content.newValue !== undefined && (
                                    <div className="rounded bg-green-100 p-1.5 text-green-700">
                                      {opt.content.newValue}
                                    </div>
                                  )}
                                </div>
                              ) : (
                                <div className="text-sm text-slate-600">
                                  {opt.content.oldValue || opt.content.newValue}
                                </div>
                              )}
                              {opt.isCorrect.changed && (
                                <div className="mt-1 text-xs text-slate-500">
                                  正确答案: {opt.isCorrect.oldValue} → {opt.isCorrect.newValue}
                                </div>
                              )}
                            </div>
                          ))}
                        </div>
                      </div>
                    </div>
                  </div>
                ) : null}
              </div>
            ) : selectedVersion ? (
              <div className="p-4">
                <div className="mb-4 flex items-center justify-between">
                  <div>
                    <span className="text-lg font-semibold text-slate-800">
                      版本 v{selectedVersion.versionNumber}
                    </span>
                    {selectedVersion.changeNote && (
                      <p className="text-sm text-slate-500">{selectedVersion.changeNote}</p>
                    )}
                  </div>
                  <button
                    className="btn btn-sm btn-outline btn-warning"
                    onClick={() => setRollbackConfirm(selectedVersion)}
                  >
                    回滚到此版本
                  </button>
                </div>

                {selectedVersion.snapshot && (
                  <div className="space-y-4">
                    <div className="rounded-lg border border-slate-200 p-4">
                      <div className="mb-2 text-sm font-medium text-slate-700">题干</div>
                      <div className="text-slate-800">{selectedVersion.snapshot.title}</div>
                    </div>

                    {selectedVersion.snapshot.description && (
                      <div className="rounded-lg border border-slate-200 p-4">
                        <div className="mb-2 text-sm font-medium text-slate-700">知识点说明</div>
                        <div className="text-slate-700">{selectedVersion.snapshot.description}</div>
                      </div>
                    )}

                    <div className="rounded-lg border border-slate-200 p-4">
                      <div className="mb-3 text-sm font-medium text-slate-700">选项</div>
                      <div className="space-y-2">
                        {selectedVersion.snapshot.options.map((opt, idx) => (
                          <div
                            key={idx}
                            className={`flex items-center gap-3 rounded-lg border p-3 ${
                              opt.isCorrect
                                ? 'border-green-300 bg-green-50'
                                : 'border-slate-200'
                            }`}
                          >
                            <div
                              className={`flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-sm font-medium ${
                                opt.isCorrect
                                  ? 'bg-green-500 text-white'
                                  : 'bg-slate-200 text-slate-600'
                              }`}
                            >
                              {String.fromCharCode(65 + idx)}
                            </div>
                            <div className="flex-1 text-slate-700">{opt.content}</div>
                            {opt.isCorrect && (
                              <span className="badge badge-success badge-xs">正确答案</span>
                            )}
                          </div>
                        ))}
                      </div>
                    </div>
                  </div>
                )}
              </div>
            ) : (
              <div className="flex h-full items-center justify-center text-slate-500">
                选择左侧版本查看详情
              </div>
            )}
          </div>
        </div>

        {rollbackConfirm && (
          <div className="fixed inset-0 z-60 flex items-center justify-center bg-slate-900/50">
            <div className="w-full max-w-md rounded-2xl bg-base-100 p-6 shadow-2xl">
              <h3 className="mb-2 text-lg font-semibold text-slate-800">确认回滚</h3>
              <p className="mb-4 text-sm text-slate-600">
                确定要回滚到版本 <span className="font-medium">v{rollbackConfirm.versionNumber}</span> 吗？
              </p>
              <p className="mb-6 text-xs text-slate-500">
                回滚操作会将题目恢复到该版本的状态，并生成一个新的版本记录。
              </p>
              <div className="flex justify-end gap-2">
                <button
                  className="btn btn-ghost"
                  onClick={() => setRollbackConfirm(null)}
                  disabled={rollingBack}
                >
                  取消
                </button>
                <button
                  className="btn btn-warning"
                  onClick={() => handleRollback(rollbackConfirm)}
                  disabled={rollingBack}
                >
                  {rollingBack ? '回滚中...' : '确认回滚'}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
