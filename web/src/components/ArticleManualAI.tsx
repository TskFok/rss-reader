import { useEffect, useState } from 'react';
import { createPortal } from 'react-dom';
import type { Article, AIModel } from '../api/client';
import { articlesApi, aiModelsApi } from '../api/client';
import { AI_TARGET_LANGUAGES, isKnownTargetLanguageCode } from '../constants/aiTargetLanguages';
import { useToast } from '../contexts/ToastContext';
import { articleNeedsClassifySlot, articleNeedsTranslateSlot } from '../utils/articleManualAi';

function parseApiError(e: unknown): { message: string; aiLastError?: string } {
  const ax = e as { response?: { data?: { error?: string; ai_last_error?: string } } };
  const d = ax.response?.data;
  return {
    message: d?.error ?? '请求失败',
    aiLastError: typeof d?.ai_last_error === 'string' ? d.ai_last_error : undefined,
  };
}

function langLabel(code: string): string {
  return AI_TARGET_LANGUAGES.find((x) => x.code === code)?.label ?? code;
}

export default function ArticleManualAI({
  article,
  onArticlePatched,
}: {
  article: Article;
  onArticlePatched: (a: Article) => void;
}) {
  const toast = useToast();
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [models, setModels] = useState<AIModel[]>([]);
  const [modelsLoading, setModelsLoading] = useState(true);

  const feedHasModel = (article.feed_ai_model_id ?? 0) > 0;
  const feedHasLang = !!(article.feed_ai_target_language ?? '').trim();

  const [useFeedForClassify, setUseFeedForClassify] = useState(feedHasModel);
  const [classifyModelId, setClassifyModelId] = useState(article.feed_ai_model_id ?? 0);

  const [useFeedForTranslate, setUseFeedForTranslate] = useState(feedHasModel && feedHasLang);
  const [trModelId, setTrModelId] = useState(article.feed_ai_model_id ?? 0);
  const [trLang, setTrLang] = useState(() => {
    const fl = (article.feed_ai_target_language ?? '').trim();
    return fl && isKnownTargetLanguageCode(fl) ? fl : 'zh-CN';
  });

  const [uiErr, setUiErr] = useState<string | null>(null);
  const [lastAiErr, setLastAiErr] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setModelsLoading(true);
    aiModelsApi
      .list()
      .then((r) => {
        if (!cancelled) setModels(r.data);
      })
      .catch(() => {
        if (!cancelled) setModels([]);
      })
      .finally(() => {
        if (!cancelled) setModelsLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    setUseFeedForClassify((article.feed_ai_model_id ?? 0) > 0);
    setClassifyModelId(article.feed_ai_model_id ?? 0);
    const hf = (article.feed_ai_model_id ?? 0) > 0 && !!(article.feed_ai_target_language ?? '').trim();
    setUseFeedForTranslate(hf);
    setTrModelId(article.feed_ai_model_id ?? 0);
    const fl = (article.feed_ai_target_language ?? '').trim();
    setTrLang(fl && isKnownTargetLanguageCode(fl) ? fl : 'zh-CN');
    setUiErr(null);
    setLastAiErr(null);
    setDrawerOpen(false);
  }, [article.id, article.feed_ai_model_id, article.feed_ai_target_language]);

  useEffect(() => {
    if (!models.length) return;
    setClassifyModelId((prev) => (prev > 0 ? prev : models[0].id));
    setTrModelId((prev) => (prev > 0 ? prev : models[0].id));
  }, [models]);

  useEffect(() => {
    if (!drawerOpen) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setDrawerOpen(false);
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [drawerOpen]);

  const showClassify = articleNeedsClassifySlot(article);
  const showTranslate = articleNeedsTranslateSlot(article);

  if (!showClassify && !showTranslate) return null;

  const runClassify = async () => {
    setUiErr(null);
    setLastAiErr(null);
    if (!useFeedForClassify) {
      if (!classifyModelId) {
        toast.showToast({ message: '请选择用于分类的模型', variant: 'error' });
        return;
      }
    }
    const payload = useFeedForClassify ? {} : { ai_model_id: classifyModelId };
    const tid = toast.showToast({ message: 'AI 分类中…', variant: 'loading' });
    try {
      const { data } = await articlesApi.manualAIClassify(article.id, payload);
      toast.dismiss(tid);
      toast.showToast({ message: '分类已保存', variant: 'success' });
      onArticlePatched(data.article);
      setDrawerOpen(false);
    } catch (e) {
      toast.dismiss(tid);
      const { message, aiLastError } = parseApiError(e);
      setUiErr(message);
      setLastAiErr(aiLastError ?? null);
      toast.showToast({ message: '分类失败', variant: 'error', duration: 5000 });
    }
  };

  const runTranslate = async () => {
    setUiErr(null);
    setLastAiErr(null);
    if (!useFeedForTranslate) {
      if (!trModelId) {
        toast.showToast({ message: '请选择用于翻译的模型', variant: 'error' });
        return;
      }
      if (!trLang) {
        toast.showToast({ message: '请选择目标语言', variant: 'error' });
        return;
      }
    }
    const payload = useFeedForTranslate
      ? {}
      : { ai_model_id: trModelId, ai_target_language: trLang };
    const tid = toast.showToast({ message: 'AI 翻译中…', variant: 'loading' });
    try {
      const { data } = await articlesApi.manualAITranslate(article.id, payload);
      toast.dismiss(tid);
      toast.showToast({ message: '译文已保存', variant: 'success' });
      onArticlePatched(data.article);
      setDrawerOpen(false);
    } catch (e) {
      toast.dismiss(tid);
      const { message, aiLastError } = parseApiError(e);
      setUiErr(message);
      setLastAiErr(aiLastError ?? null);
      toast.showToast({ message: '翻译失败', variant: 'error', duration: 5000 });
    }
  };

  const drawerBody =
    modelsLoading ? (
      <p className="article-manual-ai-hint">加载 AI 模型…</p>
    ) : models.length === 0 ? (
      <p className="article-manual-ai-hint">请先在「订阅」→「AI 模型」中添加至少一个模型。</p>
    ) : (
      <>
        {(uiErr || (lastAiErr && lastAiErr !== uiErr)) && (
          <div className="article-manual-ai-errors">
            {uiErr ? <p className="article-manual-ai-error">{uiErr}</p> : null}
            {lastAiErr && lastAiErr !== uiErr ? (
              <p className="article-manual-ai-error-secondary" title={lastAiErr}>
                服务端记录的 ai_last_error：{lastAiErr}
              </p>
            ) : null}
          </div>
        )}

        {showClassify && (
          <div className="article-manual-ai-block">
            <div className="article-manual-ai-block-title">AI 分类</div>
            <label className="article-manual-ai-check">
              <input
                type="checkbox"
                checked={useFeedForClassify}
                disabled={!feedHasModel}
                onChange={(e) => setUseFeedForClassify(e.target.checked)}
              />
              使用订阅默认模型
              {feedHasModel && article.feed_ai_model_id ? (
                <span className="article-manual-ai-muted">
                  （ID {article.feed_ai_model_id}）
                </span>
              ) : (
                <span className="article-manual-ai-muted">（订阅未配置模型时请取消勾选并选择）</span>
              )}
            </label>
            {!useFeedForClassify && (
              <div className="article-manual-ai-row">
                <label>模型</label>
                <select
                  value={classifyModelId || ''}
                  onChange={(e) => setClassifyModelId(Number(e.target.value))}
                >
                  {models.map((m) => (
                    <option key={m.id} value={m.id}>
                      {m.name}
                    </option>
                  ))}
                </select>
              </div>
            )}
            <button type="button" className="article-manual-ai-btn primary" onClick={runClassify}>
              执行分类
            </button>
          </div>
        )}

        {showTranslate && (
          <div className="article-manual-ai-block">
            <div className="article-manual-ai-block-title">AI 翻译</div>
            <label className="article-manual-ai-check">
              <input
                type="checkbox"
                checked={useFeedForTranslate}
                disabled={!feedHasModel || !feedHasLang}
                onChange={(e) => setUseFeedForTranslate(e.target.checked)}
              />
              使用订阅默认（模型与目标语言）
              {feedHasModel && feedHasLang ? (
                <span className="article-manual-ai-muted">
                  （模型 {article.feed_ai_model_id}，{langLabel(article.feed_ai_target_language ?? '')}）
                </span>
              ) : (
                <span className="article-manual-ai-muted">（请取消勾选并自选模型与语言）</span>
              )}
            </label>
            {!useFeedForTranslate && (
              <>
                <div className="article-manual-ai-row">
                  <label>模型</label>
                  <select
                    value={trModelId || ''}
                    onChange={(e) => setTrModelId(Number(e.target.value))}
                  >
                    {models.map((m) => (
                      <option key={m.id} value={m.id}>
                        {m.name}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="article-manual-ai-row">
                  <label>目标语言</label>
                  <select value={trLang} onChange={(e) => setTrLang(e.target.value)}>
                    {AI_TARGET_LANGUAGES.map((o) => (
                      <option key={o.code} value={o.code}>
                        {o.label}
                      </option>
                    ))}
                  </select>
                </div>
              </>
            )}
            <button type="button" className="article-manual-ai-btn primary" onClick={runTranslate}>
              执行翻译
            </button>
          </div>
        )}
      </>
    );

  const drawer =
    drawerOpen &&
    typeof document !== 'undefined' &&
    createPortal(
      <div className="article-ai-drawer-root" role="dialog" aria-modal="true" aria-labelledby="article-ai-drawer-title">
        <div
          className="article-ai-drawer-backdrop"
          role="presentation"
          onClick={() => setDrawerOpen(false)}
        />
        <aside className="article-ai-drawer-panel">
          <div className="article-ai-drawer-header">
            <h2 id="article-ai-drawer-title">手动 AI</h2>
            <button
              type="button"
              className="article-ai-drawer-close"
              onClick={() => setDrawerOpen(false)}
              aria-label="关闭"
            >
              ×
            </button>
          </div>
          <div className="article-ai-drawer-body">
            <p className="article-ai-drawer-article-title" title={article.title}>
              {article.title}
            </p>
            <div className="article-manual-ai article-manual-ai--drawer">{drawerBody}</div>
          </div>
        </aside>
      </div>,
      document.body
    );

  return (
    <>
      <button
        type="button"
        className="article-detail-ai-trigger"
        onClick={() => setDrawerOpen(true)}
        title="手动 AI 分类与翻译"
      >
        手动 AI
      </button>
      {drawer}
    </>
  );
}
