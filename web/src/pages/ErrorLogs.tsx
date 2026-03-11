import { useEffect, useState } from 'react';
import { errorLogsApi } from '../api/client';
import type { ErrorLogItem } from '../api/client';

const PAGE_SIZE = 20;

function formatDate(s: string) {
  const d = new Date(s);
  if (Number.isNaN(d.getTime())) return s;
  return d.toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

export default function ErrorLogs() {
  const [items, setItems] = useState<ErrorLogItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [expanded, setExpanded] = useState<Set<number>>(new Set());

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError('');
    (async () => {
      try {
        const r = await errorLogsApi.list({ page, page_size: PAGE_SIZE });
        if (cancelled) return;
        setItems(r.data.items);
        setTotal(r.data.total);
      } catch (err: unknown) {
        if (cancelled) return;
        const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
        setError(msg || '获取错误日志失败');
        setItems([]);
        setTotal(0);
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, [page]);

  const toggleExpanded = (id: number) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const handleDelete = async (id: number) => {
    if (!confirm('确定删除这条错误日志？')) return;
    try {
      await errorLogsApi.delete(id);
      setItems((prev) => prev.filter((x) => x.id !== id));
      setTotal((t) => Math.max(0, t - 1));
    } catch {}
  };

  return (
    <div className="feeds-page">
      <section className="feeds-card">
        <div className="feeds-card-header">
          <div>
            <h2>错误日志</h2>
            <p>记录程序执行过程中的错误（信息 / 位置 / 时间）</p>
          </div>
          <div className="feeds-card-header-right">
            <span className="feeds-card-sub">{total} 条</span>
          </div>
        </div>

        {error && <p className="error">{error}</p>}

        {loading ? (
          <div className="feeds-empty-card">加载中...</div>
        ) : items.length === 0 ? (
          <div className="feeds-empty-card">暂无错误日志</div>
        ) : (
          <>
            <div className="feeds-list-scroll">
              <ul className="feeds-category-list">
                {items.map((it) => {
                  const isOpen = expanded.has(it.id);
                  const headline = `${formatDate(it.created_at)} · ${it.level.toUpperCase()} · ${it.method} ${it.path} · ${it.status}`;
                  return (
                    <li key={it.id} style={{ alignItems: 'flex-start', gap: '12px' }}>
                      <div className="feeds-category-main" style={{ gap: '6px' }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '10px', flexWrap: 'wrap' }}>
                          <span className="feeds-category-name">{headline}</span>
                          <span className="feeds-proxy-url">{it.location}</span>
                        </div>
                        <div style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>{it.message}</div>
                        {it.stack && (
                          <>
                            <button type="button" className="feeds-summary-toggle-btn" onClick={() => toggleExpanded(it.id)}>
                              {isOpen ? '收起堆栈' : '展开堆栈'}
                            </button>
                            {isOpen && (
                              <pre style={{ margin: 0, padding: '10px 12px', borderRadius: '8px', overflow: 'auto' }}>
                                {it.stack}
                              </pre>
                            )}
                          </>
                        )}
                      </div>
                      <div className="feeds-category-actions">
                        <button type="button" className="danger" onClick={() => handleDelete(it.id)}>删除</button>
                      </div>
                    </li>
                  );
                })}
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

