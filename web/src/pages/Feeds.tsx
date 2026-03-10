import { useState, useEffect, useRef } from 'react';
import { useSearchParams } from 'react-router-dom';
import { feedsApi, categoriesApi, opmlApi, proxiesApi } from '../api/client';
import type { Feed, FeedCategory, Proxy } from '../api/client';
import { useAuth } from '../contexts/AuthContext';
import Admin from './Admin';

const PAGE_SIZE = 8;

export default function Feeds() {
  const { user } = useAuth();
  const isSuperAdmin = user?.is_super_admin ?? false;

  const [feeds, setFeeds] = useState<Feed[]>([]);
  const [categories, setCategories] = useState<FeedCategory[]>([]);
  const [url, setUrl] = useState('');
  const [categoryId, setCategoryId] = useState<number | ''>('');
  const [interval, setInterval] = useState(60);
  const [expireDays, setExpireDays] = useState(90);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [editing, setEditing] = useState<number | null>(null);
  const [editInterval, setEditInterval] = useState(60);
  const [editExpireDays, setEditExpireDays] = useState(90);
  const [catName, setCatName] = useState('');
  const [catError, setCatError] = useState('');
  const [catLoading, setCatLoading] = useState(false);
  const [editingCat, setEditingCat] = useState<number | null>(null);
  const [editCatName, setEditCatName] = useState('');
  const [categoryPage, setCategoryPage] = useState(1);
  const [feedsPage, setFeedsPage] = useState(1);
  const urlInputRef = useRef<HTMLInputElement | null>(null);
  const [proxyId, setProxyId] = useState<number | ''>('');
  const [editProxyId, setEditProxyId] = useState<number | ''>('');
  const [searchParams, setSearchParams] = useSearchParams();
  const initialTabParam = (searchParams.get('tab') || 'feeds') as 'categories' | 'feeds' | 'proxies' | 'users';
  const [activeTab, setActiveTabState] = useState<'categories' | 'feeds' | 'proxies' | 'users'>(() => {
    if (initialTabParam === 'users' && !isSuperAdmin) {
      return 'feeds';
    }
    if (initialTabParam === 'categories' || initialTabParam === 'feeds' || initialTabParam === 'proxies' || initialTabParam === 'users') {
      return initialTabParam;
    }
    return 'feeds';
  });

  // URL 变化时同步 activeTab（如刷新、浏览器前进/后退）
  useEffect(() => {
    const tab = (searchParams.get('tab') || 'feeds') as 'categories' | 'feeds' | 'proxies' | 'users';
    const next = tab === 'users' && !isSuperAdmin ? 'feeds' : tab;
    if (['categories', 'feeds', 'proxies', 'users'].includes(next)) {
      setActiveTabState(next);
    }
  }, [searchParams, isSuperAdmin]);
  const opmlInputRef = useRef<HTMLInputElement | null>(null);
  const [opmlMsg, setOpmlMsg] = useState('');
  const [opmlLoading, setOpmlLoading] = useState(false);
  const [proxies, setProxies] = useState<Proxy[]>([]);
  const [proxyName, setProxyName] = useState('');
  const [proxyUrl, setProxyUrl] = useState('');
  const [proxyError, setProxyError] = useState('');
  const [proxyLoading, setProxyLoading] = useState(false);
  const [editingProxy, setEditingProxy] = useState<number | null>(null);
  const [editProxyName, setEditProxyName] = useState('');
  const [editProxyUrl, setEditProxyUrl] = useState('');

  const totalCategoryPages = Math.max(1, Math.ceil(categories.length / PAGE_SIZE));
  const totalFeedsPages = Math.max(1, Math.ceil(feeds.length / PAGE_SIZE));
  const paginatedCategories = categories.slice((categoryPage - 1) * PAGE_SIZE, categoryPage * PAGE_SIZE);
  const paginatedFeeds = feeds.slice((feedsPage - 1) * PAGE_SIZE, feedsPage * PAGE_SIZE);

  // 仅在当前 tab 下请求对应接口；订阅列表页串行请求，避免同时请求导致数据库 unexpected EOF
  useEffect(() => {
    if (activeTab === 'categories') {
      loadCategories();
      return;
    }
    if (activeTab === 'proxies') {
      loadProxies();
      return;
    }
    if (activeTab === 'feeds') {
      let cancelled = false;
      (async () => {
        try {
          const fr = await feedsApi.list();
          if (!cancelled) setFeeds(fr.data);
        } catch {
          if (!cancelled) setFeeds([]);
        }
        try {
          const cr = await categoriesApi.list();
          if (!cancelled) setCategories(cr.data);
        } catch {
          if (!cancelled) setCategories([]);
        }
        try {
          const pr = await proxiesApi.list();
          if (!cancelled) setProxies(pr.data);
        } catch {
          if (!cancelled) setProxies([]);
        }
      })();
      return () => {
        cancelled = true;
      };
    }
  }, [activeTab]);

  // 数据变少时若当前页超出范围则回到第 1 页
  useEffect(() => {
    if (categoryPage > totalCategoryPages) setCategoryPage(1);
  }, [categories.length, categoryPage, totalCategoryPages]);
  useEffect(() => {
    if (feedsPage > totalFeedsPages) setFeedsPage(1);
  }, [feeds.length, feedsPage, totalFeedsPages]);

  const loadFeeds = () => {
    feedsApi.list().then((r) => setFeeds(r.data)).catch(() => setFeeds([]));
  };

  const loadCategories = () => {
    categoriesApi.list().then((r) => setCategories(r.data)).catch(() => setCategories([]));
  };

  const loadProxies = () => {
    proxiesApi.list().then((r) => setProxies(r.data)).catch(() => setProxies([]));
  };

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      if (categoryId === '') {
        setError('请选择分类');
        return;
      }
      await feedsApi.create(
        url,
        categoryId,
        interval,
        proxyId === '' ? null : proxyId,
        expireDays
      );
      setUrl('');
      setCategoryId('');
      setProxyId('');
      loadFeeds();
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
      setError(msg || '添加失败');
    } finally {
      setLoading(false);
    }
  };

  const handleAddCategory = async (e: React.FormEvent) => {
    e.preventDefault();
    setCatError('');
    setCatLoading(true);
    try {
      const { data } = await categoriesApi.create(catName);
      setCatName('');
      setCategories((prev) => [data, ...prev]);
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
      setCatError(msg || '创建失败');
    } finally {
      setCatLoading(false);
    }
  };

  const handleUpdateCategory = async (id: number) => {
    try {
      const { data } = await categoriesApi.update(id, editCatName);
      setCategories((prev) => prev.map((c) => (c.id === id ? data : c)));
      setEditingCat(null);
    } catch {}
  };

  const handleDeleteCategory = async (id: number) => {
    if (!confirm('确定删除此分类？')) return;
    try {
      await categoriesApi.delete(id);
      setCategories((prev) => prev.filter((c) => c.id !== id));
      // 如果当前选择的分类被删了，清空选择
      if (categoryId === id) setCategoryId('');
    } catch {}
  };

  const handleUpdate = async (id: number) => {
    try {
      await feedsApi.update(
        id,
        editInterval,
        editProxyId === '' ? null : editProxyId,
        editExpireDays
      );
      setEditing(null);
      loadFeeds();
    } catch {}
  };

  const handleDelete = async (id: number) => {
    if (!confirm('确定删除此订阅？')) return;
    try {
      await feedsApi.delete(id);
      loadFeeds();
    } catch {}
  };

  const handleAddProxy = async (e: React.FormEvent) => {
    e.preventDefault();
    setProxyError('');
    setProxyLoading(true);
    try {
      await proxiesApi.create(proxyName, proxyUrl);
      setProxyName('');
      setProxyUrl('');
      loadProxies();
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
      setProxyError(msg || '添加失败');
    } finally {
      setProxyLoading(false);
    }
  };

  const handleUpdateProxy = async (id: number) => {
    try {
      await proxiesApi.update(id, editProxyName, editProxyUrl);
      setEditingProxy(null);
      loadProxies();
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
      setProxyError(msg || '更新失败');
    }
  };

  const handleDeleteProxy = async (id: number) => {
    if (!confirm('确定删除此代理？')) return;
    try {
      await proxiesApi.delete(id);
      loadProxies();
    } catch {}
  };

  const formatDate = (s: string | null) => {
    if (!s) return '从未';
    return new Date(s).toLocaleString('zh-CN');
  };

  const handleExportOPML = async () => {
    try {
      const res = await opmlApi.export();
      const blob = new Blob([res.data], { type: 'text/xml' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'subscriptions.opml';
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch {
      setOpmlMsg('导出失败');
    }
  };

  const handleImportOPMLChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setOpmlLoading(true);
    setOpmlMsg('');
    try {
      const res = await opmlApi.import(file);
      const data = res.data as { imported?: number; skipped?: number; failed?: number };
      const imported = data.imported ?? 0;
      const skipped = data.skipped ?? 0;
      const failed = data.failed ?? 0;
      setOpmlMsg(`导入完成：成功 ${imported} 条，跳过 ${skipped} 条，失败 ${failed} 条`);
      loadFeeds();
      loadCategories();
    } catch {
      setOpmlMsg('导入失败');
    } finally {
      setOpmlLoading(false);
      e.target.value = '';
    }
  };

  const tabTitle =
    activeTab === 'categories'
      ? '订阅分类'
      : activeTab === 'feeds'
      ? '订阅列表'
      : activeTab === 'proxies'
      ? '代理'
      : '用户管理';
  const tabDesc =
    activeTab === 'categories'
      ? '用于对订阅进行分组管理'
      : activeTab === 'feeds'
      ? '当前账号下的所有订阅源'
      : activeTab === 'proxies'
      ? '配置 RSS 抓取时使用的代理服务器'
      : '管理系统用户与账号状态';

  return (
    <div className="feeds-page settings-page">
      <aside className="settings-sidebar">
        <div className="settings-sidebar-title">系统设置</div>
        <button
          type="button"
          className={`settings-sidebar-item ${activeTab === 'categories' ? 'active' : ''}`}
          onClick={() => {
            setActiveTabState('categories');
            setSearchParams((prev) => {
              const p = new URLSearchParams(prev);
              p.set('tab', 'categories');
              return p;
            });
          }}
        >
          订阅分类
        </button>
        <button
          type="button"
          className={`settings-sidebar-item ${activeTab === 'feeds' ? 'active' : ''}`}
          onClick={() => {
            setActiveTabState('feeds');
            setSearchParams((prev) => {
              const p = new URLSearchParams(prev);
              p.set('tab', 'feeds');
              return p;
            });
          }}
        >
          订阅列表
        </button>
        <button
          type="button"
          className={`settings-sidebar-item ${activeTab === 'proxies' ? 'active' : ''}`}
          onClick={() => {
            setActiveTabState('proxies');
            setSearchParams((prev) => {
              const p = new URLSearchParams(prev);
              p.set('tab', 'proxies');
              return p;
            });
          }}
        >
          代理
        </button>
        {isSuperAdmin && (
          <button
            type="button"
            className={`settings-sidebar-item ${activeTab === 'users' ? 'active' : ''}`}
            onClick={() => {
              setActiveTabState('users');
              setSearchParams((prev) => {
                const p = new URLSearchParams(prev);
                p.set('tab', 'users');
                return p;
              });
            }}
          >
            用户管理
          </button>
        )}
      </aside>

      <section className="settings-main">
        <div className="feeds-header-card">
          <div className="feeds-header-main">
            <h1>{tabTitle}</h1>
            <p>{tabDesc}</p>
          </div>
          {activeTab === 'feeds' && (
            <div className="feeds-header-side">
              <span className="feeds-header-pill">订阅总数 {feeds.length}</span>
            </div>
          )}
          {activeTab === 'categories' && (
            <div className="feeds-header-side">
              <span className="feeds-header-pill">{categories.length} 个分类</span>
            </div>
          )}
          {activeTab === 'proxies' && (
            <div className="feeds-header-side">
              <span className="feeds-header-pill">{proxies.length} 个代理</span>
            </div>
          )}
        </div>

        {activeTab === 'categories' && (
          <section className="feeds-card feeds-card-categories">
            <div className="feeds-card-header">
              <div>
                <h2>订阅分类</h2>
                <p>用于对订阅进行分组管理</p>
              </div>
              <span className="feeds-card-sub">{categories.length} 个分类</span>
            </div>

            <form onSubmit={handleAddCategory} className="feeds-inline-form">
              <input
                type="text"
                placeholder="输入分类名称"
                value={catName}
                onChange={(e) => setCatName(e.target.value)}
                required
              />
              <button type="submit" disabled={catLoading}>
                {catLoading ? '创建中...' : '新建分类'}
              </button>
            </form>
            {catError && <p className="error">{catError}</p>}

            {categories.length === 0 ? (
              <div className="feeds-empty-card">暂无分类，请先创建分类。</div>
            ) : (
              <>
                <div className="feeds-list-scroll">
                  <ul className="feeds-category-list">
                    {paginatedCategories.map((c) => (
                    <li key={c.id}>
                      <div className="feeds-category-main">
                        <span className="feeds-category-name">{c.name}</span>
                      </div>
                      <div className="feeds-category-actions">
                        {editingCat === c.id ? (
                          <>
                            <input
                              type="text"
                              value={editCatName}
                              onChange={(e) => setEditCatName(e.target.value)}
                            />
                            <button type="button" onClick={() => handleUpdateCategory(c.id)}>
                              保存
                            </button>
                            <button type="button" onClick={() => setEditingCat(null)}>
                              取消
                            </button>
                          </>
                        ) : (
                          <>
                            <button
                              type="button"
                              onClick={() => {
                                setEditingCat(c.id);
                                setEditCatName(c.name);
                              }}
                            >
                              编辑
                            </button>
                            <button
                              type="button"
                              className="danger"
                              onClick={() => handleDeleteCategory(c.id)}
                            >
                              删除
                            </button>
                          </>
                        )}
                      </div>
                    </li>
                  ))}
                  </ul>
                </div>
                <div className="feeds-pagination feeds-pagination-bottom">
                  <button
                    type="button"
                    disabled={categoryPage <= 1}
                    onClick={() => setCategoryPage((p) => Math.max(1, p - 1))}
                  >
                    上一页
                  </button>
                  <span className="feeds-pagination-info">
                    第 {categoryPage} / {totalCategoryPages} 页（共 {categories.length} 条）
                  </span>
                  <button
                    type="button"
                    disabled={categoryPage >= totalCategoryPages}
                    onClick={() => setCategoryPage((p) => Math.min(totalCategoryPages, p + 1))}
                  >
                    下一页
                  </button>
                </div>
              </>
            )}
          </section>
        )}

        {activeTab === 'feeds' && (
          <section className="feeds-card feeds-card-feeds">
            <div className="feeds-card-header">
              <div>
                <h2>订阅列表</h2>
                <p>当前账号下的所有订阅源</p>
              </div>
            </div>

            <form onSubmit={handleAdd} className="feeds-inline-form feeds-inline-main-form">
              <input
                ref={urlInputRef}
                type="url"
                placeholder="RSS 地址，例如 https://example.com/feed"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                required
              />
              <select
                value={categoryId}
                onChange={(e) => setCategoryId(e.target.value === '' ? '' : Number(e.target.value))}
                required
                title="分类"
              >
                <option value="">选择分类</option>
                {categories.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name}
                  </option>
                ))}
              </select>
              <select
                value={interval}
                onChange={(e) => setInterval(Number(e.target.value))}
              >
                <option value={30}>30 分钟</option>
                <option value={60}>1 小时</option>
                <option value={120}>2 小时</option>
                <option value={360}>6 小时</option>
                <option value={720}>12 小时</option>
                <option value={1440}>24 小时</option>
              </select>
              <select
                value={expireDays}
                onChange={(e) => setExpireDays(Number(e.target.value))}
                title="内容保留"
              >
                <option value={0}>永不过期</option>
                <option value={30}>30 天</option>
                <option value={90}>3 个月</option>
                <option value={180}>6 个月</option>
                <option value={365}>1 年</option>
              </select>
              <select
                value={proxyId}
                onChange={(e) => setProxyId(e.target.value === '' ? '' : Number(e.target.value))}
                title="代理"
              >
                <option value="">无代理</option>
                {proxies.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name || p.url}
                  </option>
                ))}
              </select>
              <button type="submit" disabled={loading}>
                {loading ? '添加中...' : '添加订阅'}
              </button>
            </form>
            {error && <p className="error">{error}</p>}

            <div className="feeds-opml-row">
              <div className="feeds-opml-text">OPML 导入 / 导出</div>
              <div className="feeds-opml-actions">
                <button type="button" onClick={handleExportOPML}>
                  导出 OPML
                </button>
                <button
                  type="button"
                  onClick={() => opmlInputRef.current?.click()}
                  disabled={opmlLoading}
                >
                  {opmlLoading ? '导入中...' : '导入 OPML'}
                </button>
                <input
                  ref={opmlInputRef}
                  type="file"
                  accept=".opml,.xml"
                  style={{ display: 'none' }}
                  onChange={handleImportOPMLChange}
                />
              </div>
            </div>
            {opmlMsg && <p className="feeds-opml-message">{opmlMsg}</p>}

            {feeds.length === 0 ? (
              <div className="feeds-empty-card">暂无订阅，请先添加订阅。</div>
            ) : (
              <>
                <div className="feeds-list-scroll">
                  <div className="feeds-table-wrapper">
                  <table className="feeds-table">
                    <thead>
                      <tr>
                        <th>名称 / 地址</th>
                        <th>分类</th>
                        <th>代理</th>
                        <th>更新间隔</th>
                        <th>内容保留</th>
                        <th>上次更新</th>
                        <th style={{ width: '160px' }}>操作</th>
                      </tr>
                    </thead>
                    <tbody>
                      {paginatedFeeds.map((f) => {
                      const isEditing = editing === f.id;
                      return (
                        <tr key={f.id}>
                          <td>
                            <div className="feeds-table-title">
                              <div className="feeds-table-main">{f.title || f.url}</div>
                              <div className="feeds-table-sub">{f.url}</div>
                            </div>
                          </td>
                          <td>{f.category?.name || '未分类'}</td>
                          <td>
                            {isEditing ? (
                              <select
                                value={editProxyId}
                                onChange={(e) =>
                                  setEditProxyId(e.target.value === '' ? '' : Number(e.target.value))
                                }
                                title="代理"
                              >
                                <option value="">无代理</option>
                                {proxies.map((p) => (
                                  <option key={p.id} value={p.id}>
                                    {p.name || p.url}
                                  </option>
                                ))}
                              </select>
                            ) : (
                              f.proxy ? (f.proxy.name || f.proxy.url) : '无'
                            )}
                          </td>
                          <td>
                            {isEditing ? (
                              <select
                                value={editInterval}
                                onChange={(e) => setEditInterval(Number(e.target.value))}
                              >
                                <option value={30}>30 分钟</option>
                                <option value={60}>1 小时</option>
                                <option value={120}>2 小时</option>
                                <option value={360}>6 小时</option>
                                <option value={720}>12 小时</option>
                                <option value={1440}>24 小时</option>
                              </select>
                            ) : (
                              `${f.update_interval_minutes} 分钟`
                            )}
                          </td>
                          <td>
                            {isEditing ? (
                              <select
                                value={editExpireDays}
                                onChange={(e) => setEditExpireDays(Number(e.target.value))}
                                title="内容保留"
                              >
                                <option value={0}>永不过期</option>
                                <option value={30}>30 天</option>
                                <option value={90}>3 个月</option>
                                <option value={180}>6 个月</option>
                                <option value={365}>1 年</option>
                              </select>
                            ) : (
                              f.expire_days === 0 ? '永不过期' : `${f.expire_days} 天`
                            )}
                          </td>
                          <td>{formatDate(f.last_fetched_at)}</td>
                          <td>
                            <div className="feeds-row-actions">
                              {isEditing ? (
                                <>
                                  <button type="button" onClick={() => handleUpdate(f.id)}>
                                    保存
                                  </button>
                                  <button type="button" onClick={() => setEditing(null)}>
                                    取消
                                  </button>
                                </>
                              ) : (
                                <>
                                  <button
                                    type="button"
                              onClick={() => {
                                setEditing(f.id);
                                setEditInterval(f.update_interval_minutes);
                                setEditExpireDays(f.expire_days ?? 90);
                                setEditProxyId(f.proxy_id ?? '');
                              }}
                                  >
                                    编辑
                                  </button>
                                  <button
                                    type="button"
                                    className="danger"
                                    onClick={() => handleDelete(f.id)}
                                  >
                                    删除
                                  </button>
                                </>
                              )}
                            </div>
                          </td>
                        </tr>
                      );
                      })}
                    </tbody>
                  </table>
                  </div>
                </div>
                <div className="feeds-pagination feeds-pagination-bottom">
                  <button
                    type="button"
                    disabled={feedsPage <= 1}
                    onClick={() => setFeedsPage((p) => Math.max(1, p - 1))}
                  >
                    上一页
                  </button>
                  <span className="feeds-pagination-info">
                    第 {feedsPage} / {totalFeedsPages} 页（共 {feeds.length} 条）
                  </span>
                  <button
                    type="button"
                    disabled={feedsPage >= totalFeedsPages}
                    onClick={() => setFeedsPage((p) => Math.min(totalFeedsPages, p + 1))}
                  >
                    下一页
                  </button>
                </div>
              </>
            )}
          </section>
        )}

        {activeTab === 'proxies' && (
          <section className="feeds-card feeds-card-proxies">
            <div className="feeds-card-header">
              <div>
                <h2>代理列表</h2>
                <p>配置 RSS 抓取时使用的代理服务器，支持 http、https、socks5 协议</p>
              </div>
              <span className="feeds-card-sub">{proxies.length} 个代理</span>
            </div>

            <form onSubmit={handleAddProxy} className="feeds-inline-form">
              <input
                type="text"
                placeholder="名称（可选）"
                value={proxyName}
                onChange={(e) => setProxyName(e.target.value)}
              />
              <input
                type="text"
                placeholder="代理地址，如 http://127.0.0.1:7890"
                value={proxyUrl}
                onChange={(e) => setProxyUrl(e.target.value)}
                required
              />
              <button type="submit" disabled={proxyLoading}>
                {proxyLoading ? '添加中...' : '添加代理'}
              </button>
            </form>
            {proxyError && <p className="error">{proxyError}</p>}

            {proxies.length === 0 ? (
              <div className="feeds-empty-card">暂无代理，请先添加代理。</div>
            ) : (
              <div className="feeds-list-scroll">
                <ul className="feeds-category-list">
                  {proxies.map((p) => (
                    <li key={p.id}>
                      <div className="feeds-category-main">
                        <span className="feeds-category-name">{p.name || p.url}</span>
                        {p.name && <span className="feeds-proxy-url">{p.url}</span>}
                      </div>
                      <div className="feeds-category-actions">
                        {editingProxy === p.id ? (
                          <>
                            <input
                              type="text"
                              placeholder="名称"
                              value={editProxyName}
                              onChange={(e) => setEditProxyName(e.target.value)}
                            />
                            <input
                              type="text"
                              placeholder="代理地址"
                              value={editProxyUrl}
                              onChange={(e) => setEditProxyUrl(e.target.value)}
                            />
                            <button type="button" onClick={() => handleUpdateProxy(p.id)}>
                              保存
                            </button>
                            <button type="button" onClick={() => setEditingProxy(null)}>
                              取消
                            </button>
                          </>
                        ) : (
                          <>
                            <button
                              type="button"
                              onClick={() => {
                                setEditingProxy(p.id);
                                setEditProxyName(p.name);
                                setEditProxyUrl(p.url);
                                setProxyError('');
                              }}
                            >
                              编辑
                            </button>
                            <button
                              type="button"
                              className="danger"
                              onClick={() => handleDeleteProxy(p.id)}
                            >
                              删除
                            </button>
                          </>
                        )}
                      </div>
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </section>
        )}

        {activeTab === 'users' && isSuperAdmin && (
          <section className="feeds-card">
            <Admin />
          </section>
        )}
      </section>
    </div>
  );
}
