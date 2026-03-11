import { useState, useEffect, useRef } from 'react';
import { useSearchParams } from 'react-router-dom';
import { feedsApi, categoriesApi, opmlApi, proxiesApi, aiModelsApi, articlesApi, summarySchedulesApi, summaryHistoriesApi } from '../api/client';
import type { Feed, FeedCategory, Proxy, AIModel, SummarySchedule } from '../api/client';
import { useAuth } from '../contexts/AuthContext';
import Modal from '../components/Modal';
import Admin from './Admin';

const PAGE_SIZE_OPTIONS = [5, 8, 10, 20, 50] as const;
const SUMMARY_PAGE_SIZE_OPTIONS = [10, 20, 50, 100] as const;
const TAB_OPTIONS = ['categories', 'feeds', 'proxies', 'ai-models', 'ai-summary', 'ai-summary-schedule', 'users'] as const;
type TabType = (typeof TAB_OPTIONS)[number];

/** 上海时区当日的 YYYY-MM-DD */
function getTodayShanghai(): string {
  return new Date().toLocaleDateString('sv-SE', { timeZone: 'Asia/Shanghai' });
}

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
  const [feedAddOpen, setFeedAddOpen] = useState(false);
  const [editing, setEditing] = useState<number | null>(null);
  const [editError, setEditError] = useState('');
  const [editInterval, setEditInterval] = useState(60);
  const [editExpireDays, setEditExpireDays] = useState(90);
  const [editCategoryId, setEditCategoryId] = useState<number | ''>('');
  const [catName, setCatName] = useState('');
  const [catError, setCatError] = useState('');
  const [catLoading, setCatLoading] = useState(false);
  const [catAddOpen, setCatAddOpen] = useState(false);
  const [editingCat, setEditingCat] = useState<number | null>(null);
  const [editCatName, setEditCatName] = useState('');
  const [categoryPage, setCategoryPage] = useState(1);
  const [categoryPageSize, setCategoryPageSize] = useState(8);
  const [feedsPage, setFeedsPage] = useState(1);
  const [feedsPageSize, setFeedsPageSize] = useState(8);
  const [draggedCategoryId, setDraggedCategoryId] = useState<number | null>(null);
  const [dragOverCategoryId, setDragOverCategoryId] = useState<number | null>(null);
  const urlInputRef = useRef<HTMLInputElement | null>(null);
  const [proxyId, setProxyId] = useState<number | ''>('');
  const [editProxyId, setEditProxyId] = useState<number | ''>('');
  const [searchParams, setSearchParams] = useSearchParams();
  const initialTabParam = (searchParams.get('tab') || 'feeds') as TabType;
  const [activeTab, setActiveTabState] = useState<TabType>(() => {
    if (initialTabParam === 'users' && !isSuperAdmin) {
      return 'feeds';
    }
    if (TAB_OPTIONS.includes(initialTabParam)) {
      return initialTabParam;
    }
    return 'feeds';
  });

  // URL 变化时同步 activeTab（如刷新、浏览器前进/后退）；非超管访问 tab=users 时修正 URL
  useEffect(() => {
    const tab = (searchParams.get('tab') || 'feeds') as TabType;
    const next = tab === 'users' && !isSuperAdmin ? 'feeds' : tab;
    if (TAB_OPTIONS.includes(next)) {
      setActiveTabState(next);
      if (tab === 'users' && !isSuperAdmin) {
        setSearchParams((prev) => {
          const p = new URLSearchParams(prev);
          p.set('tab', 'feeds');
          return p;
        });
      }
    }
  }, [searchParams, isSuperAdmin, setSearchParams]);
  const opmlInputRef = useRef<HTMLInputElement | null>(null);
  const [opmlMsg, setOpmlMsg] = useState('');
  const [opmlLoading, setOpmlLoading] = useState(false);
  const [proxies, setProxies] = useState<Proxy[]>([]);
  const [proxyName, setProxyName] = useState('');
  const [proxyUrl, setProxyUrl] = useState('');
  const [proxyError, setProxyError] = useState('');
  const [proxyLoading, setProxyLoading] = useState(false);
  const [proxyAddOpen, setProxyAddOpen] = useState(false);
  const [editingProxy, setEditingProxy] = useState<number | null>(null);
  const [editProxyName, setEditProxyName] = useState('');
  const [editProxyUrl, setEditProxyUrl] = useState('');
  const [aiModels, setAiModels] = useState<AIModel[]>([]);
  const [aiModelName, setAiModelName] = useState('');
  const [aiModelBaseUrl, setAiModelBaseUrl] = useState('');
  const [aiModelApiKey, setAiModelApiKey] = useState('');
  const [aiModelBackupId, setAiModelBackupId] = useState<number | ''>('');
  const [aiModelError, setAiModelError] = useState('');
  const [aiModelSuccess, setAiModelSuccess] = useState('');
  const [aiModelLoading, setAiModelLoading] = useState(false);
  const [aiModelAddOpen, setAiModelAddOpen] = useState(false);
  const [editingAiModel, setEditingAiModel] = useState<number | null>(null);
  const [editAiModelName, setEditAiModelName] = useState('');
  const [editAiModelBaseUrl, setEditAiModelBaseUrl] = useState('');
  const [editAiModelApiKey, setEditAiModelApiKey] = useState('');
  const [editAiModelBackupId, setEditAiModelBackupId] = useState<number | ''>('');
  const [testingAiModel, setTestingAiModel] = useState<number | null>(null);
  const [draggedAiModelId, setDraggedAiModelId] = useState<number | null>(null);
  const [dragOverAiModelId, setDragOverAiModelId] = useState<number | null>(null);

  // AI 总结（默认时间范围：上海时区当日）
  const [summaryAiModelId, setSummaryAiModelId] = useState<number | ''>('');
  const [summaryFeedIds, setSummaryFeedIds] = useState<Set<number>>(new Set());
  const [summaryStartDate, setSummaryStartDate] = useState(getTodayShanghai);
  const [summaryEndDate, setSummaryEndDate] = useState(getTodayShanghai);
  const [summaryLoading, setSummaryLoading] = useState(false);
  const [summaryResult, setSummaryResult] = useState('');
  const [summaryError, setSummaryError] = useState('');
  const [summaryArticleCount, setSummaryArticleCount] = useState(0);
  const [summaryTotal, setSummaryTotal] = useState<number | null>(null);
  const [summaryPanelOpen, setSummaryPanelOpen] = useState(true);
  const [summaryPage, setSummaryPage] = useState(1);
  const [summaryPageSize, setSummaryPageSize] = useState(20);
  const [summaryOrder, setSummaryOrder] = useState<'desc' | 'asc'>('desc');
  const [summarySavedMsg, setSummarySavedMsg] = useState('');
  const [summarySaving, setSummarySaving] = useState(false);

  // 定时总结配置
  const [scheduleItems, setScheduleItems] = useState<SummarySchedule[]>([]);
  const [scheduleLoading, setScheduleLoading] = useState(false);
  const [scheduleError, setScheduleError] = useState('');
  const [scheduleAiModelId, setScheduleAiModelId] = useState<number | ''>('');
  const [scheduleFeedIds, setScheduleFeedIds] = useState<Set<number>>(new Set());
  const [scheduleRunAt, setScheduleRunAt] = useState('08:30');
  const [schedulePageSize, setSchedulePageSize] = useState(20);
  const [scheduleOrder, setScheduleOrder] = useState<'desc' | 'asc'>('desc');
  const [scheduleModalOpen, setScheduleModalOpen] = useState(false);
  const [editingScheduleId, setEditingScheduleId] = useState<number | null>(null);

  const totalCategoryPages = Math.max(1, Math.ceil(categories.length / categoryPageSize));
  const totalFeedsPages = Math.max(1, Math.ceil(feeds.length / feedsPageSize));
  const paginatedCategories = categories.slice((categoryPage - 1) * categoryPageSize, categoryPage * categoryPageSize);
  const paginatedFeeds = feeds.slice((feedsPage - 1) * feedsPageSize, feedsPage * feedsPageSize);

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
    if (activeTab === 'ai-models') {
      loadAiModels();
      return;
    }
    if (activeTab === 'ai-summary') {
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
          const mr = await aiModelsApi.list();
          if (!cancelled) setAiModels(mr.data);
        } catch {
          if (!cancelled) setAiModels([]);
        }
      })();
      return () => {
        cancelled = true;
      };
    }
    if (activeTab === 'ai-summary-schedule') {
      let cancelled = false;
      (async () => {
        try {
          const fr = await feedsApi.list();
          if (!cancelled) setFeeds(fr.data);
        } catch {
          if (!cancelled) setFeeds([]);
        }
        try {
          const mr = await aiModelsApi.list();
          if (!cancelled) setAiModels(mr.data);
        } catch {
          if (!cancelled) setAiModels([]);
        }
        try {
          const sr = await summarySchedulesApi.list();
          if (!cancelled) setScheduleItems(sr.data);
        } catch {
          if (!cancelled) setScheduleItems([]);
        }
        if (!cancelled) {
          // 默认选择模型
          if (scheduleAiModelId === '' && aiModels.length > 0) setScheduleAiModelId(aiModels[0].id);
        }
      })();
      return () => { cancelled = true; };
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

  // 数据或每页条数变化时，若当前页超出范围则回到第 1 页
  useEffect(() => {
    if (categoryPage > totalCategoryPages) setCategoryPage(1);
  }, [categories.length, categoryPage, categoryPageSize, totalCategoryPages]);
  useEffect(() => {
    if (feedsPage > totalFeedsPages) setFeedsPage(1);
  }, [feeds.length, feedsPage, feedsPageSize, totalFeedsPages]);

  // AI 总结：有模型列表且未选择时，默认选第一个
  useEffect(() => {
    if (activeTab === 'ai-summary' && aiModels.length > 0 && summaryAiModelId === '') {
      setSummaryAiModelId(aiModels[0].id);
    }
  }, [activeTab, aiModels, summaryAiModelId]);

  useEffect(() => {
    if (activeTab === 'ai-summary-schedule' && aiModels.length > 0 && scheduleAiModelId === '') {
      setScheduleAiModelId(aiModels[0].id);
    }
  }, [activeTab, aiModels, scheduleAiModelId]);

  const toggleScheduleFeed = (feedId: number) => {
    setScheduleFeedIds((prev) => {
      const next = new Set(prev);
      if (next.has(feedId)) next.delete(feedId);
      else next.add(feedId);
      return next;
    });
  };

  const loadSchedules = async () => {
    try {
      const r = await summarySchedulesApi.list();
      setScheduleItems(r.data);
    } catch {
      setScheduleItems([]);
    }
  };

  const handleCreateSchedule = async (e: React.FormEvent) => {
    e.preventDefault();
    setScheduleError('');
    if (scheduleAiModelId === '') {
      setScheduleError('请选择 AI 模型');
      return;
    }
    setScheduleLoading(true);
    try {
      const payload = {
        ai_model_id: scheduleAiModelId,
        feed_ids: scheduleFeedIds.size > 0 ? [...scheduleFeedIds] : [],
        run_at: scheduleRunAt,
        page_size: schedulePageSize,
        order: scheduleOrder,
      };
      if (editingScheduleId !== null) {
        await summarySchedulesApi.update(editingScheduleId, payload);
      } else {
        await summarySchedulesApi.create(payload);
      }
      await loadSchedules();
      setScheduleModalOpen(false);
      setEditingScheduleId(null);
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
      setScheduleError(msg || '保存失败');
    } finally {
      setScheduleLoading(false);
    }
  };

  const handleDeleteSchedule = async (id: number) => {
    if (!confirm('确定删除这条定时总结配置？')) return;
    try {
      await summarySchedulesApi.delete(id);
      setScheduleItems((prev) => prev.filter((x) => x.id !== id));
    } catch {}
  };

  const handleToggleScheduleEnabled = async (s: SummarySchedule) => {
    setScheduleError('');
    try {
      let ids: number[] = [];
      try {
        ids = JSON.parse(s.feed_ids_json || '[]') as number[];
      } catch {
        ids = [];
      }
      await summarySchedulesApi.update(s.id, {
        ai_model_id: s.ai_model_id,
        feed_ids: ids,
        run_at: s.run_at,
        page_size: s.page_size,
        order: (s.order === 'asc' ? 'asc' : 'desc') as 'asc' | 'desc',
        enabled: !s.enabled,
      });
      setScheduleItems((prev) => prev.map((x) => (x.id === s.id ? { ...x, enabled: !x.enabled } : x)));
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
      setScheduleError(msg || '更新失败');
    }
  };

  const openCreateScheduleModal = () => {
    setScheduleError('');
    setEditingScheduleId(null);
    setScheduleRunAt('08:30');
    setSchedulePageSize(20);
    setScheduleOrder('desc');
    setScheduleFeedIds(new Set());
    if (aiModels.length > 0 && scheduleAiModelId === '') {
      setScheduleAiModelId(aiModels[0].id);
    }
    setScheduleModalOpen(true);
  };

  const openEditScheduleModal = (s: SummarySchedule) => {
    setScheduleError('');
    setEditingScheduleId(s.id);
    setScheduleAiModelId(s.ai_model_id);
    setScheduleRunAt(s.run_at || '08:30');
    setSchedulePageSize(s.page_size || 20);
    setScheduleOrder((s.order === 'asc' ? 'asc' : 'desc') as 'asc' | 'desc');
    try {
      const ids = JSON.parse(s.feed_ids_json || '[]') as number[];
      setScheduleFeedIds(new Set(ids));
    } catch {
      setScheduleFeedIds(new Set());
    }
    setScheduleModalOpen(true);
  };

  const loadFeeds = () => {
    feedsApi.list().then((r) => setFeeds(r.data)).catch(() => setFeeds([]));
  };

  const loadCategories = () => {
    categoriesApi.list().then((r) => setCategories(r.data)).catch(() => setCategories([]));
  };

  const loadProxies = () => {
    proxiesApi.list().then((r) => setProxies(r.data)).catch(() => setProxies([]));
  };

  const loadAiModels = () => {
    aiModelsApi.list().then((r) => setAiModels(r.data)).catch(() => setAiModels([]));
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
      setFeedAddOpen(false);
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
      setCatAddOpen(false);
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
      setCatError(msg || '创建失败');
    } finally {
      setCatLoading(false);
    }
  };

  const handleUpdateCategory = async (id: number) => {
    setCatError('');
    try {
      const { data } = await categoriesApi.update(id, editCatName);
      setCategories((prev) => prev.map((c) => (c.id === id ? data : c)));
      setEditingCat(null);
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
      setCatError(msg || '更新失败');
    }
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

  const handleCategoryDragStart = (e: React.DragEvent, id: number) => {
    setDraggedCategoryId(id);
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('text/plain', String(id));
  };

  const handleCategoryDragEnd = () => {
    setDraggedCategoryId(null);
    setDragOverCategoryId(null);
  };

  const handleCategoryDragOver = (e: React.DragEvent, id: number) => {
    e.preventDefault();
    if (draggedCategoryId === null || draggedCategoryId === id) return;
    setDragOverCategoryId(id);
  };

  const handleCategoryDrop = (e: React.DragEvent, targetId: number) => {
    e.preventDefault();
    setDraggedCategoryId(null);
    setDragOverCategoryId(null);
    if (draggedCategoryId === null || draggedCategoryId === targetId) return;
    const fromIdx = categories.findIndex((c) => c.id === draggedCategoryId);
    const toIdx = categories.findIndex((c) => c.id === targetId);
    if (fromIdx === -1 || toIdx === -1) return;
    const next = [...categories];
    const [removed] = next.splice(fromIdx, 1);
    next.splice(toIdx, 0, removed);
    const idList = next.map((c) => c.id);
    setCategories(next);
    categoriesApi.reorder(idList).catch(() => loadCategories());
  };

  const handleUpdate = async (id: number) => {
    setEditError('');
    try {
      await feedsApi.update(
        id,
        editInterval,
        editProxyId === '' ? null : editProxyId,
        editExpireDays,
        editCategoryId === '' ? 0 : editCategoryId
      );
      setEditing(null);
      loadFeeds();
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
      setEditError(msg || '更新失败');
    }
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
      setProxyAddOpen(false);
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

  const handleAddAiModel = async (e: React.FormEvent) => {
    e.preventDefault();
    setAiModelError('');
    setAiModelSuccess('');
    setAiModelLoading(true);
    try {
      await aiModelsApi.create(
        aiModelName,
        aiModelBaseUrl,
        aiModelApiKey || undefined,
        aiModelBackupId === '' ? undefined : aiModelBackupId
      );
      setAiModelName('');
      setAiModelBaseUrl('');
      setAiModelApiKey('');
      setAiModelBackupId('');
      setAiModelAddOpen(false);
      loadAiModels();
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
      setAiModelError(msg || '添加失败');
    } finally {
      setAiModelLoading(false);
    }
  };

  const handleUpdateAiModel = async (id: number) => {
    setAiModelError('');
    setAiModelSuccess('');
    try {
      await aiModelsApi.update(
        id,
        editAiModelName,
        editAiModelBaseUrl,
        editAiModelApiKey === '' ? undefined : editAiModelApiKey,
        editAiModelBackupId === '' ? undefined : editAiModelBackupId
      );
      setEditingAiModel(null);
      setEditAiModelApiKey('');
      setEditAiModelBackupId('');
      loadAiModels();
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
      setAiModelError(msg || '更新失败');
    }
  };

  const handleDeleteAiModel = async (id: number) => {
    if (!confirm('确定删除此 AI 模型？')) return;
    try {
      await aiModelsApi.delete(id);
      loadAiModels();
    } catch {}
  };

  const handleTestAiModel = async (id: number) => {
    setTestingAiModel(id);
    setAiModelError('');
    setAiModelSuccess('');
    try {
      await aiModelsApi.test(id);
      const modelName = aiModels.find((m) => m.id === id)?.name ?? '模型';
      setAiModelSuccess(`${modelName} 可用`);
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
      setAiModelError(msg || '检测失败');
    } finally {
      setTestingAiModel(null);
    }
  };

  const handleAiModelDragStart = (e: React.DragEvent, id: number) => {
    setDraggedAiModelId(id);
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('text/plain', String(id));
  };

  const handleAiModelDragEnd = () => {
    setDraggedAiModelId(null);
    setDragOverAiModelId(null);
  };

  const handleAiModelDragOver = (e: React.DragEvent, id: number) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
    if (draggedAiModelId !== null && draggedAiModelId !== id) setDragOverAiModelId(id);
  };

  const handleAiModelDrop = (e: React.DragEvent, targetId: number) => {
    e.preventDefault();
    setDragOverAiModelId(null);
    if (draggedAiModelId === null || draggedAiModelId === targetId) {
      setDraggedAiModelId(null);
      return;
    }
    const fromIdx = aiModels.findIndex((m) => m.id === draggedAiModelId);
    const toIdx = aiModels.findIndex((m) => m.id === targetId);
    if (fromIdx === -1 || toIdx === -1) {
      setDraggedAiModelId(null);
      return;
    }
    const next = [...aiModels];
    const [item] = next.splice(fromIdx, 1);
    next.splice(toIdx, 0, item);
    setAiModels(next);
    setDraggedAiModelId(null);
    aiModelsApi.reorder(next.map((m) => m.id)).catch(() => setAiModelError('排序保存失败'));
  };

  const handleSummary = async (e?: React.FormEvent, overridePage?: number) => {
    e?.preventDefault();
    setSummaryError('');
    setSummaryResult('');
    setSummarySavedMsg('');
    if (summaryAiModelId === '') {
      setSummaryError('请选择 AI 模型');
      return;
    }
    setSummaryLoading(true);
    try {
      const pageToUse = overridePage ?? Math.max(1, summaryPage);
      const params: {
        ai_model_id: number;
        feed_ids?: number[];
        start_time?: string;
        end_time?: string;
        page?: number;
        page_size?: number;
        order?: 'desc' | 'asc';
      } = { ai_model_id: summaryAiModelId };
      if (summaryFeedIds.size > 0) {
        params.feed_ids = [...summaryFeedIds];
      }
      if (summaryStartDate) params.start_time = summaryStartDate;
      if (summaryEndDate) params.end_time = summaryEndDate;
      params.page = pageToUse;
      params.page_size = summaryPageSize;
      params.order = summaryOrder;
      await articlesApi.summarizeStream(params, {
        onMeta: (count) => setSummaryArticleCount(count),
        onMetaAll: (meta) => {
          if (typeof meta.total === 'number') setSummaryTotal(meta.total);
        },
        onChunk: (delta) =>
          setSummaryResult((prev) => prev + delta),
        onError: (msg) => setSummaryError(msg),
      });
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { error?: string } } })?.response?.data?.error ??
        (err as Error)?.message;
      setSummaryError(msg || '生成总结失败');
    } finally {
      setSummaryLoading(false);
    }
  };

  const handleSummaryNextPage = async () => {
    const next = Math.max(1, summaryPage) + 1;
    setSummaryPage(next);
    await handleSummary(undefined, next);
  };

  const handleSaveSummary = async () => {
    if (summaryLoading || summarySaving) return;
    if (!summaryResult.trim()) return;
    if (summaryAiModelId === '') return;
    setSummarySaving(true);
    setSummarySavedMsg('');
    setSummaryError('');
    try {
      await summaryHistoriesApi.create({
        ai_model_id: summaryAiModelId,
        feed_ids: summaryFeedIds.size > 0 ? [...summaryFeedIds] : [],
        start_time: summaryStartDate || undefined,
        end_time: summaryEndDate || undefined,
        page: Math.max(1, summaryPage),
        page_size: summaryPageSize,
        order: summaryOrder,
        article_count: summaryArticleCount,
        total: summaryTotal ?? undefined,
        content: summaryResult,
      });
      setSummarySavedMsg('已保存到总结历史');
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
      setSummaryError(msg || '保存失败');
    } finally {
      setSummarySaving(false);
    }
  };

  const toggleSummaryFeed = (feedId: number) => {
    setSummaryFeedIds((prev) => {
      const next = new Set(prev);
      if (next.has(feedId)) {
        next.delete(feedId);
      } else {
        next.add(feedId);
      }
      return next;
    });
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

  return (
    <div className="feeds-page">
      <section className="settings-main">
        {activeTab === 'categories' && (
          <section className="feeds-card feeds-card-categories">
            <div className="feeds-card-header">
              <div>
                <h2>订阅分类</h2>
                <p>用于对订阅进行分组管理</p>
              </div>
              <div className="feeds-card-header-right">
                <span className="feeds-card-sub">{categories.length} 个分类</span>
                <button type="button" className="feeds-primary-btn" onClick={() => { setCatAddOpen(true); setCatError(''); }}>
                  新建分类
                </button>
              </div>
            </div>

            <Modal open={catAddOpen} onClose={() => { setCatAddOpen(false); setCatError(''); }} title="新建分类">
              <form onSubmit={handleAddCategory} className="feeds-modal-form">
                {catError && <p className="error">{catError}</p>}
                <div className="feeds-modal-row">
                  <label>分类名称</label>
                  <input
                    type="text"
                    placeholder="输入分类名称"
                    value={catName}
                    onChange={(e) => setCatName(e.target.value)}
                    required
                  />
                </div>
                <div className="feeds-modal-actions">
                  <button type="button" onClick={() => { setCatAddOpen(false); setCatError(''); }}>取消</button>
                  <button type="submit" disabled={catLoading}>{catLoading ? '创建中...' : '确定'}</button>
                </div>
              </form>
            </Modal>

            <Modal open={editingCat !== null} onClose={() => { setEditingCat(null); setCatError(''); }} title="编辑分类">
              <form onSubmit={(e) => { e.preventDefault(); if (editingCat !== null) handleUpdateCategory(editingCat); }} className="feeds-modal-form">
                {catError && <p className="error">{catError}</p>}
                <div className="feeds-modal-row">
                  <label>分类名称</label>
                  <input
                    type="text"
                    placeholder="输入分类名称"
                    value={editCatName}
                    onChange={(e) => setEditCatName(e.target.value)}
                    required
                  />
                </div>
                <div className="feeds-modal-actions">
                  <button type="button" onClick={() => setEditingCat(null)}>取消</button>
                  <button type="submit">保存</button>
                </div>
              </form>
            </Modal>

            {categories.length === 0 ? (
              <div className="feeds-empty-card">暂无分类，请先创建分类。</div>
            ) : (
              <>
                <div className="feeds-list-scroll">
                  <ul className="feeds-category-list feeds-category-list-draggable">
                    {paginatedCategories.map((c) => (
                    <li
                      key={c.id}
                      draggable
                      onDragStart={(e) => handleCategoryDragStart(e, c.id)}
                      onDragEnd={handleCategoryDragEnd}
                      onDragOver={(e) => handleCategoryDragOver(e, c.id)}
                      onDrop={(e) => handleCategoryDrop(e, c.id)}
                      className={
                        draggedCategoryId === c.id
                          ? 'feeds-category-dragging'
                          : dragOverCategoryId === c.id
                            ? 'feeds-category-drag-over'
                            : ''
                      }
                    >
                      <span className="feeds-drag-handle" title="拖动排序">⋮⋮</span>
                      <div className="feeds-category-main">
                        <span className="feeds-category-name">{c.name}</span>
                      </div>
                      <div className="feeds-category-actions">
                        <button
                          type="button"
                          onClick={() => {
                            setEditingCat(c.id);
                            setEditCatName(c.name);
                            setCatError('');
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
                  <label className="feeds-pagination-size">
                    每页
                    <select
                      value={categoryPageSize}
                      onChange={(e) => {
                        const v = Number(e.target.value);
                        setCategoryPageSize(v);
                        setCategoryPage(1);
                      }}
                    >
                      {PAGE_SIZE_OPTIONS.map((n) => (
                        <option key={n} value={n}>{n}</option>
                      ))}
                    </select>
                    条
                  </label>
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
              <div className="feeds-card-header-right">
                <button type="button" className="feeds-primary-btn" onClick={() => { setFeedAddOpen(true); setError(''); }}>
                  添加订阅
                </button>
              </div>
            </div>

            <Modal open={feedAddOpen} onClose={() => { setFeedAddOpen(false); setError(''); }} title="添加订阅">
              <form onSubmit={handleAdd} className="feeds-modal-form">
                {error && <p className="error">{error}</p>}
                <div className="feeds-modal-row">
                  <label>RSS 地址</label>
                  <input
                    ref={urlInputRef}
                    type="url"
                    placeholder="https://example.com/feed"
                    value={url}
                    onChange={(e) => setUrl(e.target.value)}
                    required
                  />
                </div>
                <div className="feeds-modal-row">
                  <label>分类</label>
                  <select
                    value={categoryId}
                    onChange={(e) => setCategoryId(e.target.value === '' ? '' : Number(e.target.value))}
                    required
                  >
                    <option value="">选择分类</option>
                    {categories.map((c) => (
                      <option key={c.id} value={c.id}>
                        {c.name}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="feeds-modal-row">
                  <label>更新间隔</label>
                  <select value={interval} onChange={(e) => setInterval(Number(e.target.value))}>
                    <option value={30}>30 分钟</option>
                    <option value={60}>1 小时</option>
                    <option value={120}>2 小时</option>
                    <option value={360}>6 小时</option>
                    <option value={720}>12 小时</option>
                    <option value={1440}>24 小时</option>
                  </select>
                </div>
                <div className="feeds-modal-row">
                  <label>内容保留</label>
                  <select value={expireDays} onChange={(e) => setExpireDays(Number(e.target.value))}>
                    <option value={0}>永不过期</option>
                    <option value={30}>30 天</option>
                    <option value={90}>3 个月</option>
                    <option value={180}>6 个月</option>
                    <option value={365}>1 年</option>
                  </select>
                </div>
                <div className="feeds-modal-row">
                  <label>代理</label>
                  <select
                    value={proxyId}
                    onChange={(e) => setProxyId(e.target.value === '' ? '' : Number(e.target.value))}
                  >
                    <option value="">无代理</option>
                    {proxies.map((p) => (
                      <option key={p.id} value={p.id}>
                        {p.name || p.url}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="feeds-modal-actions">
                  <button type="button" onClick={() => { setFeedAddOpen(false); setError(''); }}>取消</button>
                  <button type="submit" disabled={loading}>{loading ? '添加中...' : '确定'}</button>
                </div>
              </form>
            </Modal>

            <Modal open={editing !== null} onClose={() => { setEditing(null); setEditError(''); }} title="编辑订阅">
              <form onSubmit={(e) => { e.preventDefault(); if (editing !== null) handleUpdate(editing); }} className="feeds-modal-form">
                {editError && <p className="error">{editError}</p>}
                <div className="feeds-modal-row">
                  <label>分类</label>
                  <select
                    value={editCategoryId}
                    onChange={(e) => setEditCategoryId(e.target.value === '' ? '' : Number(e.target.value))}
                  >
                    <option value="">未分类</option>
                    {categories.map((c) => (
                      <option key={c.id} value={c.id}>
                        {c.name}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="feeds-modal-row">
                  <label>代理</label>
                  <select
                    value={editProxyId}
                    onChange={(e) => setEditProxyId(e.target.value === '' ? '' : Number(e.target.value))}
                  >
                    <option value="">无代理</option>
                    {proxies.map((p) => (
                      <option key={p.id} value={p.id}>
                        {p.name || p.url}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="feeds-modal-row">
                  <label>更新间隔</label>
                  <select value={editInterval} onChange={(e) => setEditInterval(Number(e.target.value))}>
                    <option value={30}>30 分钟</option>
                    <option value={60}>1 小时</option>
                    <option value={120}>2 小时</option>
                    <option value={360}>6 小时</option>
                    <option value={720}>12 小时</option>
                    <option value={1440}>24 小时</option>
                  </select>
                </div>
                <div className="feeds-modal-row">
                  <label>内容保留</label>
                  <select value={editExpireDays} onChange={(e) => setEditExpireDays(Number(e.target.value))}>
                    <option value={0}>永不过期</option>
                    <option value={30}>30 天</option>
                    <option value={90}>3 个月</option>
                    <option value={180}>6 个月</option>
                    <option value={365}>1 年</option>
                  </select>
                </div>
                <div className="feeds-modal-actions">
                  <button type="button" onClick={() => { setEditing(null); setEditError(''); }}>取消</button>
                  <button type="submit">保存</button>
                </div>
              </form>
            </Modal>

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
                      {paginatedFeeds.map((f) => (
                        <tr key={f.id}>
                          <td>
                            <div className="feeds-table-title">
                              <div className="feeds-table-main">{f.title || f.url}</div>
                              <div className="feeds-table-sub">{f.url}</div>
                            </div>
                          </td>
                          <td>{f.category?.name || '未分类'}</td>
                          <td>{f.proxy ? (f.proxy.name || f.proxy.url) : '无'}</td>
                          <td>{f.update_interval_minutes} 分钟</td>
                          <td>{f.expire_days === 0 ? '永不过期' : `${f.expire_days} 天`}</td>
                          <td>{formatDate(f.last_fetched_at)}</td>
                          <td>
                            <div className="feeds-row-actions">
                              <button
                                type="button"
                                onClick={() => {
                                  setEditing(f.id);
                                  setEditInterval(f.update_interval_minutes);
                                  setEditExpireDays(f.expire_days ?? 90);
                                  setEditProxyId(f.proxy_id ?? '');
                                  setEditCategoryId(f.category_id ?? '');
                                }}
                              >
                                编辑
                              </button>
                              <button type="button" className="danger" onClick={() => handleDelete(f.id)}>
                                删除
                              </button>
                            </div>
                          </td>
                        </tr>
                      ))}
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
                  <label className="feeds-pagination-size">
                    每页
                    <select
                      value={feedsPageSize}
                      onChange={(e) => {
                        const v = Number(e.target.value);
                        setFeedsPageSize(v);
                        setFeedsPage(1);
                      }}
                    >
                      {PAGE_SIZE_OPTIONS.map((n) => (
                        <option key={n} value={n}>{n}</option>
                      ))}
                    </select>
                    条
                  </label>
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
              <div className="feeds-card-header-right">
                <span className="feeds-card-sub">{proxies.length} 个代理</span>
                <button type="button" className="feeds-primary-btn" onClick={() => { setProxyAddOpen(true); setProxyError(''); }}>
                  添加代理
                </button>
              </div>
            </div>

            <Modal open={proxyAddOpen} onClose={() => { setProxyAddOpen(false); setProxyError(''); }} title="添加代理">
              <form onSubmit={handleAddProxy} className="feeds-modal-form">
                {proxyError && <p className="error">{proxyError}</p>}
                <div className="feeds-modal-row">
                  <label>名称（可选）</label>
                  <input
                    type="text"
                    placeholder="名称"
                    value={proxyName}
                    onChange={(e) => setProxyName(e.target.value)}
                  />
                </div>
                <div className="feeds-modal-row">
                  <label>代理地址</label>
                  <input
                    type="text"
                    placeholder="如 http://127.0.0.1:7890"
                    value={proxyUrl}
                    onChange={(e) => setProxyUrl(e.target.value)}
                    required
                  />
                </div>
                <div className="feeds-modal-actions">
                  <button type="button" onClick={() => { setProxyAddOpen(false); setProxyError(''); }}>取消</button>
                  <button type="submit" disabled={proxyLoading}>{proxyLoading ? '添加中...' : '确定'}</button>
                </div>
              </form>
            </Modal>

            <Modal open={editingProxy !== null} onClose={() => { setEditingProxy(null); setProxyError(''); }} title="编辑代理">
              <form onSubmit={(e) => { e.preventDefault(); if (editingProxy !== null) handleUpdateProxy(editingProxy); }} className="feeds-modal-form">
                {proxyError && <p className="error">{proxyError}</p>}
                <div className="feeds-modal-row">
                  <label>名称（可选）</label>
                  <input
                    type="text"
                    placeholder="名称"
                    value={editProxyName}
                    onChange={(e) => setEditProxyName(e.target.value)}
                  />
                </div>
                <div className="feeds-modal-row">
                  <label>代理地址</label>
                  <input
                    type="text"
                    placeholder="如 http://127.0.0.1:7890"
                    value={editProxyUrl}
                    onChange={(e) => setEditProxyUrl(e.target.value)}
                    required
                  />
                </div>
                <div className="feeds-modal-actions">
                  <button type="button" onClick={() => { setEditingProxy(null); setProxyError(''); }}>取消</button>
                  <button type="submit">保存</button>
                </div>
              </form>
            </Modal>

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
                        <button type="button" className="danger" onClick={() => handleDeleteProxy(p.id)}>
                          删除
                        </button>
                      </div>
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </section>
        )}

        {activeTab === 'ai-models' && (
          <section className="feeds-card feeds-card-proxies">
            <div className="feeds-card-header">
              <div>
                <h2>AI 模型列表</h2>
                <p>配置 AI 模型名称、调用地址与 API 密钥，支持 OpenAI 兼容接口（如 OpenAI、Azure、Ollama 等）</p>
              </div>
              <div className="feeds-card-header-right">
                <span className="feeds-card-sub">{aiModels.length} 个模型</span>
                <button type="button" className="feeds-primary-btn" onClick={() => { setAiModelAddOpen(true); setAiModelError(''); setAiModelSuccess(''); }}>
                  添加模型
                </button>
              </div>
            </div>
            {aiModelError && <p className="error">{aiModelError}</p>}
            {aiModelSuccess && <p className="bind-msg-success">{aiModelSuccess}</p>}

            <Modal open={aiModelAddOpen} onClose={() => { setAiModelAddOpen(false); setAiModelError(''); setAiModelSuccess(''); }} title="添加 AI 模型">
              <form onSubmit={handleAddAiModel} className="feeds-modal-form">
                {aiModelError && <p className="error">{aiModelError}</p>}
                <div className="feeds-modal-row">
                  <label>模型名称</label>
                  <input
                    type="text"
                    placeholder="如 gpt-4o-mini"
                    value={aiModelName}
                    onChange={(e) => setAiModelName(e.target.value)}
                    required
                  />
                </div>
                <div className="feeds-modal-row">
                  <label>调用地址</label>
                  <input
                    type="text"
                    placeholder="如 https://api.openai.com/v1"
                    value={aiModelBaseUrl}
                    onChange={(e) => setAiModelBaseUrl(e.target.value)}
                    required
                  />
                </div>
                <div className="feeds-modal-row">
                  <label>API 密钥（可选）</label>
                  <input
                    type="password"
                    placeholder="留空则不设置"
                    value={aiModelApiKey}
                    onChange={(e) => setAiModelApiKey(e.target.value)}
                  />
                </div>
                <div className="feeds-modal-row">
                  <label>备用模型（可选）</label>
                  <select
                    value={aiModelBackupId}
                    onChange={(e) =>
                      setAiModelBackupId(e.target.value === '' ? '' : Number(e.target.value))
                    }
                  >
                    <option value="">不设置</option>
                    {aiModels.map((m) => (
                      <option key={m.id} value={m.id}>
                        {m.name}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="feeds-modal-actions">
                  <button type="button" onClick={() => { setAiModelAddOpen(false); setAiModelError(''); setAiModelSuccess(''); }}>取消</button>
                  <button type="submit" disabled={aiModelLoading}>{aiModelLoading ? '添加中...' : '确定'}</button>
                </div>
              </form>
            </Modal>

            <Modal open={editingAiModel !== null} onClose={() => { setEditingAiModel(null); setEditAiModelApiKey(''); setEditAiModelBackupId(''); setAiModelError(''); setAiModelSuccess(''); }} title="编辑 AI 模型">
              <form onSubmit={(e) => { e.preventDefault(); if (editingAiModel !== null) handleUpdateAiModel(editingAiModel); }} className="feeds-modal-form">
                {aiModelError && <p className="error">{aiModelError}</p>}
                <div className="feeds-modal-row">
                  <label>模型名称</label>
                  <input
                    type="text"
                    placeholder="如 gpt-4o-mini"
                    value={editAiModelName}
                    onChange={(e) => setEditAiModelName(e.target.value)}
                    required
                  />
                </div>
                <div className="feeds-modal-row">
                  <label>调用地址</label>
                  <input
                    type="text"
                    placeholder="如 https://api.openai.com/v1"
                    value={editAiModelBaseUrl}
                    onChange={(e) => setEditAiModelBaseUrl(e.target.value)}
                    required
                  />
                </div>
                <div className="feeds-modal-row">
                  <label>API 密钥（留空保持不变）</label>
                  <input
                    type="password"
                    placeholder="留空则不修改"
                    value={editAiModelApiKey}
                    onChange={(e) => setEditAiModelApiKey(e.target.value)}
                  />
                </div>
                <div className="feeds-modal-row">
                  <label>备用模型（可选）</label>
                  <select
                    value={editAiModelBackupId}
                    onChange={(e) =>
                      setEditAiModelBackupId(
                        e.target.value === '' ? '' : Number(e.target.value)
                      )
                    }
                  >
                    <option value="">不设置</option>
                    {aiModels
                      .filter((item) => editingAiModel === null || item.id !== editingAiModel)
                      .map((m) => (
                        <option key={m.id} value={m.id}>
                          {m.name}
                        </option>
                      ))}
                  </select>
                </div>
                <div className="feeds-modal-actions">
                  <button type="button" onClick={() => { setEditingAiModel(null); setEditAiModelApiKey(''); setAiModelError(''); setAiModelSuccess(''); }}>取消</button>
                  <button type="submit">保存</button>
                </div>
              </form>
            </Modal>

            {aiModels.length === 0 ? (
              <div className="feeds-empty-card">暂无 AI 模型，请先添加模型。</div>
            ) : (
              <div className="feeds-list-scroll">
                <ul className="feeds-category-list feeds-category-list-draggable">
                  {aiModels.map((m) => (
                    <li
                      key={m.id}
                      draggable
                      onDragStart={(e) => handleAiModelDragStart(e, m.id)}
                      onDragEnd={handleAiModelDragEnd}
                      onDragOver={(e) => handleAiModelDragOver(e, m.id)}
                      onDrop={(e) => handleAiModelDrop(e, m.id)}
                      className={
                        draggedAiModelId === m.id
                          ? 'feeds-ai-model-dragging'
                          : dragOverAiModelId === m.id
                            ? 'feeds-ai-model-drag-over'
                            : ''
                      }
                    >
                      <span className="feeds-drag-handle" title="拖动排序">⋮⋮</span>
                      <div className="feeds-category-main">
                        <span className="feeds-category-name">{m.name}</span>
                        <span className="feeds-proxy-url">{m.base_url}</span>
                        {m.backup_model_id != null && m.backup_model_id !== 0 && (
                          <span className="feeds-proxy-url">
                            备用：{aiModels.find((x) => x.id === m.backup_model_id)?.name ?? `ID ${m.backup_model_id}`}
                          </span>
                        )}
                      </div>
                      <div className="feeds-category-actions">
                        <button
                          type="button"
                          onClick={() => handleTestAiModel(m.id)}
                          disabled={testingAiModel === m.id}
                        >
                          {testingAiModel === m.id ? '检测中...' : '检测'}
                        </button>
                        <button
                          type="button"
                          onClick={() => {
                            setEditingAiModel(m.id);
                            setEditAiModelName(m.name);
                            setEditAiModelBaseUrl(m.base_url);
                            setEditAiModelApiKey('');
                            setEditAiModelBackupId(m.backup_model_id ?? '');
                            setAiModelError('');
                            setAiModelSuccess('');
                          }}
                        >
                          编辑
                        </button>
                        <button
                          type="button"
                          className="danger"
                          onClick={() => handleDeleteAiModel(m.id)}
                        >
                          删除
                        </button>
                      </div>
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </section>
        )}

        {activeTab === 'ai-summary' && (
          <section className="feeds-card feeds-card-ai-summary">
            <div className="feeds-summary-layout">
              <div className="feeds-summary-main">
                <div className="feeds-summary-main-header">
                  <h2>AI 总结</h2>
                  <div className="feeds-summary-header-actions">
                    <button
                      type="button"
                      className="feeds-primary-btn"
                      onClick={() => handleSummary()}
                      disabled={summaryLoading || aiModels.length === 0}
                      title={aiModels.length === 0 ? '请先添加 AI 模型' : undefined}
                    >
                      {summaryLoading ? '生成中...' : '生成总结'}
                    </button>
                    <button
                      type="button"
                      className="feeds-summary-toggle-btn"
                      onClick={handleSummaryNextPage}
                      disabled={summaryLoading || aiModels.length === 0}
                      title="页码 +1 并生成下一页总结"
                    >
                      总结下一页
                    </button>
                    <button
                      type="button"
                      className="feeds-summary-toggle-btn"
                      onClick={handleSaveSummary}
                      disabled={summaryLoading || summarySaving || !summaryResult.trim()}
                      title={summaryLoading ? '生成中，无法保存' : '保存当前总结到历史记录'}
                    >
                      {summarySaving ? '保存中...' : '保存'}
                    </button>
                    <button
                      type="button"
                      className="feeds-summary-toggle-btn"
                      onClick={() => setSummaryPanelOpen((v) => !v)}
                      title={summaryPanelOpen ? '收起选项' : '显示选项'}
                    >
                      {summaryPanelOpen ? '收起选项' : '选项'}
                    </button>
                  </div>
                </div>
                {summarySavedMsg && <p className="bind-msg-success">{summarySavedMsg}</p>}
                {summaryError && <p className="error">{summaryError}</p>}
                <div className="feeds-summary-content">
                  {(summaryResult || summaryArticleCount > 0) ? (
                    <>
                      <div className="feeds-summary-result-header">
                        总结结果（共 {summaryArticleCount} 篇文章）
                        {summaryLoading && !summaryResult && (
                          <span className="feeds-summary-streaming"> 生成中...</span>
                        )}
                      </div>
                      <div className="feeds-summary-result-content">
                        {summaryResult || (summaryLoading ? '等待内容...' : '')}
                      </div>
                    </>
                  ) : (
                    <div className="feeds-summary-empty-state">
                      在右侧选择模型、时间范围与订阅源后点击「生成总结」
                    </div>
                  )}
                </div>
              </div>
              <aside
                className={`feeds-summary-panel ${summaryPanelOpen ? 'feeds-summary-panel-open' : ''}`}
              >
                <div className="feeds-summary-panel-inner">
                  <form onSubmit={handleSummary} className="feeds-summary-form">
                    <div className="feeds-summary-row">
                      <label>AI 模型</label>
                      <select
                        value={summaryAiModelId}
                        onChange={(e) =>
                          setSummaryAiModelId(e.target.value === '' ? '' : Number(e.target.value))
                        }
                        required
                      >
                        <option value="">选择模型</option>
                        {aiModels.map((m) => (
                          <option key={m.id} value={m.id}>
                            {m.name}
                          </option>
                        ))}
                      </select>
                    </div>
                    {aiModels.length === 0 && (
                      <p className="feeds-summary-hint">请先在「AI 模型」中添加模型</p>
                    )}

                    <div className="feeds-summary-row">
                      <label>开始时间</label>
                      <input
                        type="date"
                        value={summaryStartDate}
                        onChange={(e) => setSummaryStartDate(e.target.value)}
                        className="feeds-summary-date-input"
                      />
                    </div>
                    <div className="feeds-summary-row">
                      <label>结束时间</label>
                      <input
                        type="date"
                        value={summaryEndDate}
                        onChange={(e) => setSummaryEndDate(e.target.value)}
                        className="feeds-summary-date-input"
                      />
                    </div>
                    <p className="feeds-summary-hint">不选则包含全部时间</p>

                    <div className="feeds-summary-row">
                      <label>订阅源</label>
                      <div className="feeds-summary-feeds">
                        {feeds.length === 0 ? (
                          <span className="feeds-summary-empty">暂无订阅</span>
                        ) : (
                          feeds.map((f) => (
                            <label key={f.id} className="feeds-summary-feed-check">
                              <input
                                type="checkbox"
                                checked={summaryFeedIds.has(f.id)}
                                onChange={() => toggleSummaryFeed(f.id)}
                              />
                              <span>{f.title || f.url}</span>
                            </label>
                          ))
                        )}
                      </div>
                    </div>
                    <p className="feeds-summary-hint">不选则包含全部订阅</p>

                    <div className="feeds-summary-row">
                      <label htmlFor="summary-page">页码</label>
                      <input
                        id="summary-page"
                        type="number"
                        min={1}
                        value={summaryPage}
                        onChange={(e) => setSummaryPage(Math.max(1, Number(e.target.value || 1)))}
                        className="feeds-summary-date-input"
                      />
                    </div>
                    <div className="feeds-summary-row">
                      <label>每页</label>
                      <select
                        value={summaryPageSize}
                        onChange={(e) => setSummaryPageSize(Number(e.target.value))}
                      >
                        {SUMMARY_PAGE_SIZE_OPTIONS.map((n) => (
                          <option key={n} value={n}>{n}</option>
                        ))}
                      </select>
                    </div>
                    <div className="feeds-summary-row">
                      <label>排序</label>
                      <select
                        value={summaryOrder}
                        onChange={(e) => setSummaryOrder(e.target.value as 'desc' | 'asc')}
                      >
                        <option value="desc">从新到旧</option>
                        <option value="asc">从旧到新</option>
                      </select>
                    </div>
                    <p className="feeds-summary-hint">通过分页控制每次参与总结的文章数量</p>
                  </form>
                </div>
              </aside>
            </div>
          </section>
        )}

        {activeTab === 'ai-summary-schedule' && (
          <section className="feeds-card">
            <div className="feeds-card-header">
              <div>
                <h2>定时总结</h2>
                <p>每天定时总结“昨天”的文章，按页生成并保存到总结历史</p>
              </div>
              <div className="feeds-card-header-right">
                <span className="feeds-card-sub">{scheduleItems.length} 条配置</span>
                <button type="button" className="feeds-primary-btn" onClick={openCreateScheduleModal}>
                  新增配置
                </button>
              </div>
            </div>

            {scheduleError && <p className="error">{scheduleError}</p>}

            <Modal
              open={scheduleModalOpen}
              onClose={() => { setScheduleModalOpen(false); setEditingScheduleId(null); setScheduleError(''); }}
              title={editingScheduleId === null ? '新增定时总结配置' : '编辑定时总结配置'}
            >
              <form onSubmit={handleCreateSchedule} className="feeds-modal-form">
                {scheduleError && <p className="error">{scheduleError}</p>}
                <div className="feeds-modal-row">
                  <label>AI 模型</label>
                  <select
                    value={scheduleAiModelId}
                    onChange={(e) => setScheduleAiModelId(e.target.value === '' ? '' : Number(e.target.value))}
                    required
                  >
                    <option value="">选择模型</option>
                    {aiModels.map((m) => (
                      <option key={m.id} value={m.id}>{m.name}</option>
                    ))}
                  </select>
                </div>
                <div className="feeds-modal-row">
                  <label>每天执行时间</label>
                  <input type="time" value={scheduleRunAt} onChange={(e) => setScheduleRunAt(e.target.value)} required />
                </div>
                <div className="feeds-modal-row">
                  <label>每页条数</label>
                  <select value={schedulePageSize} onChange={(e) => setSchedulePageSize(Number(e.target.value))}>
                    {SUMMARY_PAGE_SIZE_OPTIONS.map((n) => (
                      <option key={n} value={n}>{n}</option>
                    ))}
                  </select>
                </div>
                <div className="feeds-modal-row">
                  <label>排序</label>
                  <select value={scheduleOrder} onChange={(e) => setScheduleOrder(e.target.value as 'desc' | 'asc')}>
                    <option value="desc">从新到旧</option>
                    <option value="asc">从旧到新</option>
                  </select>
                </div>
                <div className="feeds-modal-row">
                  <label>订阅源（不选表示全部订阅）</label>
                  <div className="feeds-summary-feeds" style={{ maxHeight: '220px' }}>
                    {feeds.length === 0 ? (
                      <span className="feeds-summary-empty">暂无订阅</span>
                    ) : (
                      feeds.map((f) => (
                        <label key={f.id} className="feeds-summary-feed-check">
                          <input
                            type="checkbox"
                            checked={scheduleFeedIds.has(f.id)}
                            onChange={() => toggleScheduleFeed(f.id)}
                          />
                          <span>{f.title || f.url}</span>
                        </label>
                      ))
                    )}
                  </div>
                </div>
                <div className="feeds-modal-row">
                  <div className="feeds-modal-actions">
                  <button type="button" onClick={() => { setScheduleModalOpen(false); setEditingScheduleId(null); setScheduleError(''); }}>取消</button>
                  <button type="submit" disabled={scheduleLoading}>
                    {scheduleLoading ? '保存中...' : '保存'}
                  </button>
                  </div>
                </div>
              </form>
            </Modal>

            <div className="feeds-list-scroll">
              <ul className="feeds-category-list">
                {scheduleItems.length === 0 ? (
                  <li>
                    <div className="feeds-category-main">
                      <span className="feeds-category-name">暂无配置</span>
                    </div>
                  </li>
                ) : (
                  scheduleItems.map((s) => (
                    <li key={s.id}>
                      <div className="feeds-category-main">
                        <span className="feeds-category-name">
                          {s.enabled ? '已启用' : '未启用'} · {s.run_at} · 每页 {s.page_size} · {s.order === 'asc' ? '从旧到新' : '从新到旧'}
                        </span>
                        <span className="feeds-proxy-url">模型 ID：{s.ai_model_id}</span>
                        <span className="feeds-proxy-url">上次执行：{s.last_run_at ? formatDate(s.last_run_at) : '从未'}</span>
                      </div>
                      <div className="feeds-category-actions">
                        <button type="button" onClick={() => openEditScheduleModal(s)}>编辑</button>
                        <button type="button" onClick={() => handleToggleScheduleEnabled(s)}>
                          {s.enabled ? '停用' : '启用'}
                        </button>
                        <button type="button" className="danger" onClick={() => handleDeleteSchedule(s.id)}>删除</button>
                      </div>
                    </li>
                  ))
                )}
              </ul>
            </div>
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
