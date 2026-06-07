import { useEffect, useMemo, useState } from 'react';
import { toast } from 'react-hot-toast';

import {
  getDiscussions,
  getDiscussionReplies,
  createDiscussion,
  toggleDiscussionLike,
  deleteDiscussion,
} from '../api/client';

const MAX_CONTENT_LENGTH = 1000;

function formatTime(dateStr) {
  const date = new Date(dateStr);
  return date.toLocaleString();
}

function CommentItem(props) {
  const {
    discussion,
    onDelete,
    onReply,
    isReply = false,
    userRole,
    currentUserId,
    token,
    role,
  } = props;

  const [liked, setLiked] = useState(discussion.isLiked);
  const [likeCount, setLikeCount] = useState(discussion.likeCount);
  const [likeLoading, setLikeLoading] = useState(false);
  const [replyOpen, setReplyOpen] = useState(false);
  const [replyContent, setReplyContent] = useState('');
  const [replySubmitting, setReplySubmitting] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [showReplies, setShowReplies] = useState(false);
  const [replies, setReplies] = useState([]);
  const [replyTotal, setReplyTotal] = useState(0);
  const [replyPage, setReplyPage] = useState(1);
  const [replyLoading, setReplyLoading] = useState(false);
  const [hasMoreReplies, setHasMoreReplies] = useState(false);

  const canDelete = useMemo(() => {
    return currentUserId === discussion.authorId || userRole === 'teacher';
  }, [currentUserId, discussion.authorId, userRole]);

  const handleLike = async () => {
    if (likeLoading) return;
    setLikeLoading(true);
    try {
      const result = await toggleDiscussionLike(discussion.id, token, role);
      setLiked(result.isLiked);
      setLikeCount(result.likeCount);
    } catch (error) {
      toast.error(error.message || '操作失败');
    } finally {
      setLikeLoading(false);
    }
  };

  const loadReplies = async (page = 1) => {
    if (replyLoading) return;
    setReplyLoading(true);
    try {
      const result = await getDiscussionReplies(
        { parentId: discussion.id, page, pageSize: 5 },
        token,
        role
      );
      const newReplies = result.items || [];
      const total = result.total || 0;
      const pageSize = result.pageSize || 5;

      if (page === 1) {
        setReplies(newReplies);
      } else {
        setReplies((prev) => [...prev, ...newReplies]);
      }
      setReplyTotal(total);
      setReplyPage(page);
      setHasMoreReplies(newReplies.length > 0 && total > page * pageSize);
    } catch (error) {
      toast.error(error.message || '加载回复失败');
    } finally {
      setReplyLoading(false);
    }
  };

  const toggleReplies = () => {
    if (!showReplies) {
      setShowReplies(true);
      loadReplies(1);
    } else {
      setShowReplies(false);
    }
  };

  const handleSubmitReply = async () => {
    const content = replyContent.trim();
    if (!content) {
      toast.error('请输入回复内容');
      return;
    }
    if (content.length > MAX_CONTENT_LENGTH) {
      toast.error('内容不能超过 ' + MAX_CONTENT_LENGTH + ' 字');
      return;
    }
    setReplySubmitting(true);
    try {
      const newReply = await createDiscussion(
        {
          questionId: discussion.questionId,
          content,
          parentId: discussion.id,
        },
        token,
        role
      );
      toast.success('回复成功');
      setReplyContent('');
      setReplyOpen(false);
      if (showReplies) {
        setReplies((prev) => [{ ...newReply, isLiked: false }, ...prev]);
        setReplyTotal((prev) => prev + 1);
      } else {
        setShowReplies(true);
        setReplies([{ ...newReply, isLiked: false }]);
        setReplyTotal(1);
      }
      if (onReply) onReply(discussion.id, newReply);
    } catch (error) {
      toast.error(error.message || '回复失败');
    } finally {
      setReplySubmitting(false);
    }
  };

  const handleLoadMoreReplies = () => {
    loadReplies(replyPage + 1);
  };

  const handleReplyDeleted = (replyId) => {
    setReplies((prev) => prev.filter((r) => r.id !== replyId));
    setReplyTotal((prev) => prev - 1);
  };

  const handleDelete = async () => {
    if (!window.confirm('确定要删除这条评论吗？')) return;
    setDeleting(true);
    try {
      await deleteDiscussion(discussion.id, token, role);
      toast.success('删除成功');
      if (onDelete) onDelete(discussion.id);
    } catch (error) {
      toast.error(error.message || '删除失败');
    } finally {
      setDeleting(false);
    }
  };

  const replyClass = isReply ? 'pl-4 border-l-2 border-slate-200 ml-2' : '';
  const likeClass = liked ? 'text-rose-500' : 'text-slate-400 hover:text-rose-400';
  const authorInitial = discussion.author?.username?.charAt(0)?.toUpperCase() || '?';
  const authorName = discussion.author?.username || '匿名用户';

  return (
    <div className={replyClass}>
      <div className="py-3">
        <div className="flex items-start justify-between gap-3">
          <div className="flex items-center gap-2 flex-1 min-w-0">
            <div className="w-8 h-8 rounded-full bg-gradient-to-br from-sky-400 to-emerald-400 flex items-center justify-center text-white text-sm font-medium shrink-0">
              {authorInitial}
            </div>
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2 flex-wrap">
                <span className="font-medium text-sm text-slate-800">
                  {authorName}
                </span>
                {discussion.author?.role === 'teacher' && (
                  <span className="badge badge-xs badge-primary">教师</span>
                )}
                {!isReply && discussion.floor > 0 && (
                  <span className="text-xs text-slate-400">
                    #{discussion.floor}楼
                  </span>
                )}
                <span className="text-xs text-slate-400">
                  {formatTime(discussion.createdAt)}
                </span>
              </div>
            </div>
          </div>
          {canDelete && (
            <button
              className="btn btn-ghost btn-xs text-slate-400 hover:text-red-500"
              onClick={handleDelete}
              disabled={deleting}
            >
              {deleting ? '删除中...' : '删除'}
            </button>
          )}
        </div>

        <div className="mt-2 ml-10 text-sm text-slate-700 whitespace-pre-wrap break-words">
          {discussion.content}
        </div>

        <div className="mt-2 ml-10 flex items-center gap-4 text-xs">
          <button
            className={'flex items-center gap-1 transition-colors ' + likeClass}
            onClick={handleLike}
            disabled={likeLoading}
          >
            <span>{liked ? '❤️' : '🤍'}</span>
            <span>{likeCount}</span>
          </button>

          {!isReply && (
            <button
              className="text-slate-400 hover:text-sky-500 transition-colors"
              onClick={() => setReplyOpen(!replyOpen)}
            >
              💬 回复
            </button>
          )}

          {!isReply && replyTotal > 0 && (
            <button
              className="text-sky-500 hover:text-sky-600 transition-colors"
              onClick={toggleReplies}
            >
              {showReplies ? '收起回复' : '查看 ' + replyTotal + ' 条回复'}
            </button>
          )}
        </div>

        {replyOpen && (
          <div className="mt-3 ml-10">
            <textarea
              className="textarea textarea-bordered w-full min-h-20 text-sm"
              placeholder="写下你的回复..."
              value={replyContent}
              onChange={(e) => setReplyContent(e.target.value)}
              maxLength={MAX_CONTENT_LENGTH}
            />
            <div className="flex items-center justify-between mt-2">
              <span className="text-xs text-slate-400">
                {replyContent.length}/{MAX_CONTENT_LENGTH}
              </span>
              <div className="flex gap-2">
                <button
                  className="btn btn-ghost btn-sm"
                  onClick={() => {
                    setReplyOpen(false);
                    setReplyContent('');
                  }}
                >
                  取消
                </button>
                <button
                  className="btn btn-primary btn-sm"
                  onClick={handleSubmitReply}
                  disabled={replySubmitting || !replyContent.trim()}
                >
                  {replySubmitting ? '发送中...' : '发送'}
                </button>
              </div>
            </div>
          </div>
        )}

        {showReplies && replies.length > 0 && (
          <div className="mt-2 ml-10 space-y-1">
            {replies.map((reply) => (
              <CommentItem
                key={reply.id}
                discussion={reply}
                isReply={true}
                userRole={userRole}
                currentUserId={currentUserId}
                token={token}
                role={role}
                onDelete={handleReplyDeleted}
              />
            ))}
            {hasMoreReplies && (
              <button
                className="btn btn-ghost btn-sm w-full text-sky-500"
                onClick={handleLoadMoreReplies}
                disabled={replyLoading}
              >
                {replyLoading ? '加载中...' : '加载更多回复'}
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

export function DiscussionSection({ questionId, token, user, defaultOpen = false }) {
  const [isOpen, setIsOpen] = useState(defaultOpen);
  const [discussions, setDiscussions] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(10);
  const [loading, setLoading] = useState(false);
  const [sort, setSort] = useState('hot');
  const [hasMore, setHasMore] = useState(false);

  const [newContent, setNewContent] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const role = user?.role || 'student';

  const loadDiscussions = async (pageNum = 1, sortType = sort) => {
    if (loading) return;
    setLoading(true);
    try {
      const result = await getDiscussions(
        { questionId, sort: sortType, page: pageNum, pageSize },
        token,
        role
      );
      const items = result.items || [];
      const totalCount = result.total || 0;

      if (pageNum === 1) {
        setDiscussions(items);
      } else {
        setDiscussions((prev) => [...prev, ...items]);
      }
      setTotal(totalCount);
      setPage(pageNum);
      setHasMore(items.length > 0 && totalCount > pageNum * pageSize);
    } catch (error) {
      toast.error(error.message || '加载评论失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (isOpen && questionId) {
      loadDiscussions(1, sort);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen, questionId, sort]);

  const handleSortChange = (newSort) => {
    setSort(newSort);
    setDiscussions([]);
    setPage(1);
    setHasMore(false);
  };

  const handleLoadMore = () => {
    loadDiscussions(page + 1, sort);
  };

  const handleSubmit = async () => {
    const content = newContent.trim();
    if (!content) {
      toast.error('请输入评论内容');
      return;
    }
    if (content.length > MAX_CONTENT_LENGTH) {
      toast.error('内容不能超过 ' + MAX_CONTENT_LENGTH + ' 字');
      return;
    }
    setSubmitting(true);
    try {
      const newDiscussion = await createDiscussion(
        { questionId, content },
        token,
        role
      );
      toast.success('评论成功');
      setNewContent('');
      setDiscussions((prev) => [{ ...newDiscussion, isLiked: false }, ...prev]);
      setTotal((prev) => prev + 1);
    } catch (error) {
      toast.error(error.message || '评论失败');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = (id) => {
    setDiscussions((prev) => prev.filter((d) => d.id !== id));
    setTotal((prev) => prev - 1);
  };

  const toggleSection = () => {
    setIsOpen(!isOpen);
  };

  const hotBtnClass = 'btn btn-xs ' + (sort === 'hot' ? 'btn-primary' : 'btn-ghost');
  const timeBtnClass = 'btn btn-xs ' + (sort === 'time' ? 'btn-primary' : 'btn-ghost');

  return (
    <div className="border border-slate-200 rounded-2xl bg-white shadow-sm overflow-hidden">
      <button
        className="w-full px-5 py-4 flex items-center justify-between bg-slate-50 hover:bg-slate-100 transition-colors"
        onClick={toggleSection}
      >
        <div className="flex items-center gap-2">
          <span className="text-lg">💬</span>
          <span className="font-semibold text-slate-800">题目讨论</span>
          <span className="text-sm text-slate-500">({total})</span>
        </div>
        <span className="text-slate-400">
          {isOpen ? '▲' : '▼'}
        </span>
      </button>

      {isOpen && (
        <div className="p-5 space-y-4">
          <div>
            <textarea
              className="textarea textarea-bordered w-full min-h-24"
              placeholder="分享你的想法或问题..."
              value={newContent}
              onChange={(e) => setNewContent(e.target.value)}
              maxLength={MAX_CONTENT_LENGTH}
            />
            <div className="flex items-center justify-between mt-2">
              <span className="text-xs text-slate-400">
                {newContent.length}/{MAX_CONTENT_LENGTH}
              </span>
              <button
                className="btn btn-primary btn-sm"
                onClick={handleSubmit}
                disabled={submitting || !newContent.trim()}
              >
                {submitting ? '发表中...' : '发表评论'}
              </button>
            </div>
          </div>

          <div className="flex items-center gap-2 border-b border-slate-200 pb-2">
            <button
              className={hotBtnClass}
              onClick={() => handleSortChange('hot')}
            >
              🔥 热度排序
            </button>
            <button
              className={timeBtnClass}
              onClick={() => handleSortChange('time')}
            >
              🕐 最新发布
            </button>
          </div>

          <div className="space-y-1">
            {loading && page === 1 ? (
              <div className="py-8 text-center text-sm text-slate-400">
                加载中...
              </div>
            ) : discussions.length === 0 ? (
              <div className="py-8 text-center text-sm text-slate-400">
                暂无评论，来发表第一条吧~
              </div>
            ) : (
              <>
                {discussions.map((discussion) => (
                  <CommentItem
                    key={discussion.id}
                    discussion={discussion}
                    userRole={user?.role}
                    currentUserId={user?.id}
                    token={token}
                    role={role}
                    onDelete={handleDelete}
                  />
                ))}
                {hasMore && (
                  <div className="pt-2">
                    <button
                      className="btn btn-ghost btn-sm w-full"
                      onClick={handleLoadMore}
                      disabled={loading}
                    >
                      {loading ? '加载中...' : '加载更多'}
                    </button>
                  </div>
                )}
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
