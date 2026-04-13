import type { Article } from '../api/client';

/** 可尝试手动分类（无分类且非处理中）；是否具备模型由界面根据模型列表判断 */
export function articleNeedsClassifySlot(a: Article): boolean {
  if (a.ai_process_status === 'pending') return false;
  return (a.ai_category ?? '').trim() === '';
}

/** 可尝试手动翻译（无译文且非处理中） */
export function articleNeedsTranslateSlot(a: Article): boolean {
  if (a.ai_process_status === 'pending') return false;
  return (
    !(a.title_translated ?? '').trim() && !(a.content_translated ?? '').trim()
  );
}
