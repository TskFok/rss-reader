import type { Article } from '../api/client';

const STALE_PENDING_MS = 30 * 60 * 1000;

function isActivePending(a: Article): boolean {
  if (a.ai_process_status !== 'pending') return false;
  if (!a.updated_at) return true;
  const updatedAt = Date.parse(a.updated_at);
  if (!Number.isFinite(updatedAt)) return true;
  return Date.now() - updatedAt < STALE_PENDING_MS;
}

function hasCompletedTranslation(a: Article): boolean {
  if (!(a.content_translated ?? '').trim()) return false;
  return a.ai_process_status !== 'failed' && a.ai_process_status !== 'pending';
}

/** 可尝试手动分类（无分类且非处理中）；是否具备模型由界面根据模型列表判断 */
export function articleNeedsClassifySlot(a: Article): boolean {
  if (isActivePending(a)) return false;
  return (a.ai_category ?? '').trim() === '';
}

/** 可尝试手动翻译（无可用正文译文且非处理中） */
export function articleNeedsTranslateSlot(a: Article): boolean {
  if (isActivePending(a)) return false;
  return !hasCompletedTranslation(a);
}
