import { useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { summaryHistoriesApi } from '../api/client';
import type { SummaryHistoryItem } from '../api/client';

const PAGE_SIZE = 20;

function formatDate(s: string) {
  const d = new Date(s);
  if (Number.isNaN(d.getTime())) return s;
  return d.toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' });
}

export default function SummaryHistory() {
  const [items, setItems] = useState<SummaryHistoryItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [retryingId, setRetryingId] = useState<number | null>(null);
  const [archivingId, setArchivingId] = useState<number | null>(null);
  const [batchArchiving, setBatchArchiving] = useState(false);
  const [selectedIds, setSelectedIds] = useState<number[]>([]);
  const [reloadKey, setReloadKey] = useState(0);
  const [searchParams, setSearchParams] = useSearchParams();
  const fixedID = useMemo(() => {
    const raw = (searchParams.get('id') || '').trim();
    if (!raw) return undefined;
    const n = Number(raw);
    if (!Number.isInteger(n) || n <= 0) return undefined;
    return n;
  }, [searchParams]);

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError('');
    (async () => {
      try {
        const params = {
          page,
          page_size: PAGE_SIZE,
          id: fixedID,
        };
        const r = await summaryHistoriesApi.list(params);
        if (cancelled) return;
        setItems(r.data.items);
        setTotal(r.data.total);
        setSelectedIds([]);
      } catch (err: unknown) {
        if (cancelled) return;
        const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
        setError(msg || '获取总结历史失败');
        setItems([]);
        setTotal(0);
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, [fixedID, page, reloadKey]);

  const handleDelete = async (id: number) => {
    if (!confirm('确定删除这条总结记录？')) return;
    try {
      await summaryHistoriesApi.delete(id);
      setItems((prev) => prev.filter((x) => x.id !== id));
      setTotal((t) => Math.max(0, t - 1));
    } catch {}
  };

  const handleRetry = async (id: number) => {
    setRetryingId(id);
    setError('');
    try {
      await summaryHistoriesApi.retry(id);
      // 重新拉取当前页，确保展示最新的重试结果
      setReloadKey((k) => k + 1);
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
      setError(msg || '重试失败');
    } finally {
      setRetryingId((cur) => (cur === id ? null : cur));
    }
  };

  const handleArchive = async (id: number) => {
    setArchivingId(id);
    setError('');
    try {
      await summaryHistoriesApi.archiveToKnowledge(id);
      setError('');
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
      setError(msg || '归档失败');
    } finally {
      setArchivingId((cur) => (cur === id ? null : cur));
    }
  };

  const handleArchiveSelected = async () => {
    if (selectedIds.length === 0) return;
    setBatchArchiving(true);
    setError('');
    try {
      const r = await summaryHistoriesApi.archiveBatchToKnowledge(selectedIds);
      const failed = r.data.failed_ids?.length ?? 0;
      if (failed > 0) {
        setError(`批量归档部分失败：${failed}/${selectedIds.length}`);
      } else {
        setError('');
      }
      setReloadKey((k) => k + 1);
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
      setError(msg || '批量归档失败');
    } finally {
      setBatchArchiving(false);
    }
  };

  return (
    <div className="feeds-page">
      <section className="feeds-card">
        <div className="feeds-card-header">
          <div>
            <h2>总结历史</h2>
            <p>查看与管理 AI 总结的历史记录</p>
          </div>
          <div className="feeds-card-header-right">
            <span className="feeds-card-sub">{total} 条</span>
            <button
              type="button"
              onClick={handleArchiveSelected}
              disabled={batchArchiving || selectedIds.length === 0}
            >
              {batchArchiving ? '归档中...' : `批量沉淀（${selectedIds.length}）`}
            </button>
            {fixedID ? (
              <button
                type="button"
                onClick={() => {
                  const next = new URLSearchParams(searchParams);
                  next.delete('id');
                  setSearchParams(next, { replace: true });
                  setPage(1);
                }}
              >
                清除ID筛选
              </button>
            ) : null}
          </div>
        </div>

        {error && <p className="error">{error}</p>}

        {loading ? (
          <div className="feeds-empty-card">加载中...</div>
        ) : items.length === 0 ? (
          <div className="feeds-empty-card">暂无总结历史</div>
        ) : (
          <>
            <div className="feeds-list-scroll">
              <ul className="feeds-category-list">
                {items.map((it) => (
                  <li key={it.id} style={{ alignItems: 'flex-start', gap: '12px' }}>
                    <input
                      type="checkbox"
                      aria-label={`选择记录-${it.id}`}
                      checked={selectedIds.includes(it.id)}
                      onChange={(e) => {
                        setSelectedIds((prev) => {
                          if (e.target.checked) return [...prev, it.id];
                          return prev.filter((id) => id !== it.id);
                        });
                      }}
                    />
                    <div className="feeds-category-main" style={{ gap: '6px' }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '10px', flexWrap: 'wrap' }}>
                        <span className="feeds-category-name">{formatDate(it.created_at)}</span>
                        <span className="feeds-proxy-url">模型：{it.ai_model_name || it.ai_model_id}</span>
                        {it.template_name && <span className="feeds-proxy-url">模板：{it.template_name}</span>}
                        <span className="feeds-proxy-url">文章：{it.article_count} / {it.total}</span>
                        <span className="feeds-proxy-url">页码：{it.page}（每页 {it.page_size}）</span>
                        <span className="feeds-proxy-url">排序：{it.order === 'asc' ? '从旧到新' : '从新到旧'}</span>
                        <span className="feeds-proxy-url">状态：{it.status || (it.error ? 'failed' : 'success')}</span>
                        <span className="feeds-proxy-url">触发：{it.trigger_source || 'manual'}</span>
                      </div>
                      {(it.start_time || it.end_time) && (
                        <div className="feeds-proxy-url">
                          时间：{it.start_time || '不限'} ~ {it.end_time || '不限'}
                        </div>
                      )}
                      {it.error && <div className="error">错误：{it.error}</div>}
                      <div className="feeds-summary-result-content" style={{ padding: 0, background: 'transparent', border: 'none' }}>
                        {it.content || (it.error ? '(无内容)' : '')}
                      </div>
                    </div>
                    <div className="feeds-category-actions">
                      {it.error && (
                        <button
                          type="button"
                          onClick={() => handleRetry(it.id)}
                          disabled={retryingId === it.id}
                        >
                          {retryingId === it.id ? '重试中...' : '重试'}
                        </button>
                      )}
                      <button
                        type="button"
                        onClick={() => handleArchive(it.id)}
                        disabled={archivingId === it.id}
                      >
                        {archivingId === it.id ? '归档中...' : '沉淀到知识库'}
                      </button>
                      <button type="button" className="danger" onClick={() => handleDelete(it.id)}>删除</button>
                    </div>
                  </li>
                ))}
              </ul>
            </div>
            <div className="feeds-pagination feeds-pagination-bottom">
              <button type="button" disabled={page <= 1} onClick={() => setPage((p) => Math.max(1, p - 1))}>上一页</button>
              <span className="feeds-pagination-info">第 {page} / {totalPages} 页（共 {total} 条）</span>
              <button type="button" disabled={page >= totalPages} onClick={() => setPage((p) => Math.min(totalPages, p + 1))}>下一页</button>
            </div>
          </>
        )}
      </section>
    </div>
  );
}
