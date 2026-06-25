import axios from 'axios';

const client = axios.create({
  baseURL: '/api',
  headers: { 'Content-Type': 'application/json' },
});

client.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

client.interceptors.response.use(
  (r) => r,
  (err) => {
    if (err.response?.status === 401) {
      const url = err.config?.url ?? '';
      // 登录/注册接口返回 401 时不重定向，由页面处理错误
      if (url.includes('/auth/login') || url.includes('/auth/register')) {
        return Promise.reject(err);
      }
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      window.location.href = '/login';
    }
    return Promise.reject(err);
  }
);

export interface User {
  id: number;
  username: string;
  status: string;
  is_super_admin: boolean;
  created_at: string;
  feishu_id?: string | null;
  feishu_name?: string | null;
}

export interface Feed {
  id: number;
  user_id: number;
  category_id: number | null;
  proxy_id: number | null;
  ai_model_id: number | null;
  ai_classify_enabled: boolean;
  ai_translate_enabled: boolean;
  ai_target_language: string;
  category?: FeedCategory;
  proxy?: Proxy | null;
  url: string;
  title: string;
  update_interval_minutes: number;
  expire_days: number; // 0=永不过期
  last_fetched_at: string | null;
  created_at: string;
}

export interface FeedCategory {
  id: number;
  user_id: number;
  name: string;
  sort_order?: number;
  created_at: string;
  updated_at: string;
}

export interface Proxy {
  id: number;
  user_id: number;
  name: string;
  url: string;
  created_at: string;
  updated_at: string;
}

export interface AIModel {
  id: number;
  user_id: number;
  name: string;
  base_url: string;
  backup_model_id?: number | null;
  sort_order?: number;
  top_p?: number | null;
  n?: number | null;
  presence_penalty?: number | null;
  frequency_penalty?: number | null;
  created_at: string;
  updated_at: string;
}

export const KIMI_DEFAULT_SAMPLING = {
  top_p: 0.95,
  n: 1,
  presence_penalty: 0,
  frequency_penalty: 0,
} as const;

export type AIModelSamplingParams = {
  top_p?: number | null;
  n?: number | null;
  presence_penalty?: number | null;
  frequency_penalty?: number | null;
};

export interface Article {
  id: number;
  feed_id: number;
  /** SHA256 十六进制 64 字符，对应 RSS 原始标识见 guid_raw */
  guid: string;
  /** RSS 条目的原始 guid 或用于去重的 link */
  guid_raw?: string;
  title: string;
  link: string;
  content: string;
  ai_process_status?: string;
  ai_last_error?: string;
  ai_category?: string;
  ai_category_translated?: string;
  title_translated?: string;
  content_translated?: string;
  published_at: string | null;
  created_at: string;
  updated_at?: string;
  read: boolean;
  favorite?: boolean;
  feed_title?: string;
  feed_ai_translate_enabled?: boolean;
  feed_ai_classify_enabled?: boolean;
  /** 订阅上配置的模型，用于手动 AI */
  feed_ai_model_id?: number | null;
  feed_ai_target_language?: string;
}

export interface SummaryHistoryItem {
  id: number;
  ai_model_id: number;
  ai_model_name: string;
  summary_template_id?: number | null;
  summary_template_name?: string;
  start_time: string;
  end_time: string;
  page: number;
  page_size: number;
  order: 'desc' | 'asc' | string;
  article_count: number;
  total: number;
  content: string;
  error: string;
  created_at: string;
}

export interface SummaryTemplate {
  id: number;
  user_id: number;
  name: string;
  system_prompt: string;
  user_prompt_prefix: string;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

export interface SummarySchedule {
  id: number;
  user_id: number;
  ai_model_id: number;
  summary_template_id?: number | null;
  feed_ids_json: string;
  run_at: string; // HH:MM
  page_size: number;
  order: 'desc' | 'asc' | string;
  enabled: boolean;
  last_run_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface ErrorLogItem {
  id: number;
  user_id?: number | null;
  level: string;
  message: string;
  location: string;
  method: string;
  path: string;
  status: number;
  stack: string;
  created_at: string;
}

export const authApi = {
  register: (username: string, password: string) =>
    client.post('/auth/register', { username, password }),
  login: (username: string, password: string) =>
    client.post<{ token: string; user: User }>('/auth/login', { username, password }),
  getFeishuLoginUrl: () =>
    client.get<{ url: string; goto: string }>('/auth/feishu/login-url'),
};

export interface FeedAIOptions {
  ai_model_id?: number | null;
  ai_classify_enabled?: boolean;
  ai_translate_enabled?: boolean;
  ai_target_language?: string;
}

export const feedsApi = {
  list: () => client.get<Feed[]>('/feeds'),
  create: (
    url: string,
    category_id: number,
    update_interval_minutes: number,
    proxy_id?: number | null,
    expire_days?: number,
    ai?: FeedAIOptions
  ) =>
    client.post<Feed>('/feeds', {
      url,
      category_id,
      update_interval_minutes,
      proxy_id: proxy_id ?? null,
      expire_days: expire_days ?? 90,
      ...(ai?.ai_model_id !== undefined && { ai_model_id: ai.ai_model_id }),
      ...(ai?.ai_classify_enabled !== undefined && { ai_classify_enabled: ai.ai_classify_enabled }),
      ...(ai?.ai_translate_enabled !== undefined && { ai_translate_enabled: ai.ai_translate_enabled }),
      ...(ai?.ai_target_language !== undefined && { ai_target_language: ai.ai_target_language }),
    }),
  update: (
    id: number,
    url: string,
    update_interval_minutes: number,
    proxy_id?: number | null,
    expire_days?: number,
    category_id?: number | null,
    ai?: FeedAIOptions
  ) =>
    client.put<Feed>(`/feeds/${id}`, {
      url,
      update_interval_minutes,
      proxy_id: proxy_id ?? null,
      ...(expire_days !== undefined && { expire_days }),
      ...(category_id !== undefined && { category_id }),
      ...(ai?.ai_model_id !== undefined && { ai_model_id: ai.ai_model_id }),
      ...(ai?.ai_classify_enabled !== undefined && { ai_classify_enabled: ai.ai_classify_enabled }),
      ...(ai?.ai_translate_enabled !== undefined && { ai_translate_enabled: ai.ai_translate_enabled }),
      ...(ai?.ai_target_language !== undefined && { ai_target_language: ai.ai_target_language }),
    }),
  refresh: (id: number) => client.post<Feed>(`/feeds/${id}/refresh`),
  delete: (id: number) => client.delete(`/feeds/${id}`),
};

export const opmlApi = {
  export: () =>
    client.get('/feeds/opml', {
      responseType: 'blob',
    }),
  import: (file: File) => {
    const form = new FormData();
    form.append('file', file);
    return client.post('/feeds/opml', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
  },
};

export const categoriesApi = {
  list: () => client.get<FeedCategory[]>('/categories'),
  create: (name: string) => client.post<FeedCategory>('/categories', { name }),
  update: (id: number, name: string) => client.put<FeedCategory>(`/categories/${id}`, { name }),
  delete: (id: number) => client.delete(`/categories/${id}`),
  reorder: (id_list: number[]) =>
    client.put<{ message: string }>('/categories/reorder', { id_list }),
};

export const proxiesApi = {
  list: () => client.get<Proxy[]>('/proxies'),
  create: (name: string, url: string) =>
    client.post<Proxy>('/proxies', { name, url }),
  update: (id: number, name: string, url: string) =>
    client.put<Proxy>(`/proxies/${id}`, { name, url }),
  delete: (id: number) => client.delete(`/proxies/${id}`),
};

export const aiModelsApi = {
  list: () => client.get<AIModel[]>('/ai-models'),
  create: (
    name: string,
    base_url: string,
    api_key?: string,
    backup_model_id?: number | null,
    sampling?: AIModelSamplingParams
  ) =>
    client.post<AIModel>('/ai-models', {
      name,
      base_url,
      api_key: api_key ?? '',
      ...(backup_model_id !== undefined && { backup_model_id: backup_model_id ?? 0 }),
      ...(sampling?.top_p !== undefined && { top_p: sampling.top_p }),
      ...(sampling?.n !== undefined && { n: sampling.n }),
      ...(sampling?.presence_penalty !== undefined && { presence_penalty: sampling.presence_penalty }),
      ...(sampling?.frequency_penalty !== undefined && { frequency_penalty: sampling.frequency_penalty }),
    }),
  update: (
    id: number,
    name: string,
    base_url: string,
    api_key?: string | null,
    backup_model_id?: number | null,
    sampling?: AIModelSamplingParams
  ) =>
    client.put<AIModel>(`/ai-models/${id}`, {
      name,
      base_url,
      ...(api_key !== undefined && { api_key: api_key ?? '' }),
      ...(backup_model_id !== undefined && { backup_model_id: backup_model_id ?? 0 }),
      ...(sampling?.top_p !== undefined && { top_p: sampling.top_p }),
      ...(sampling?.n !== undefined && { n: sampling.n }),
      ...(sampling?.presence_penalty !== undefined && { presence_penalty: sampling.presence_penalty }),
      ...(sampling?.frequency_penalty !== undefined && { frequency_penalty: sampling.frequency_penalty }),
    }),
  delete: (id: number) => client.delete(`/ai-models/${id}`),
  test: (id: number) =>
    client.post<{ message: string }>(`/ai-models/${id}/test`),
  reorder: (id_list: number[]) =>
    client.put<{ message: string }>('/ai-models/reorder', { id_list }),
};

export const articlesApi = {
  list: (params?: {
    feed_id?: number;
    read?: boolean;
    favorite?: boolean;
    page?: number;
    page_size?: number;
  }) => client.get<{ items: Article[]; total: number }>('/articles', { params }),
  markRead: (id: number) => client.put(`/articles/${id}/read`),
  toggleFavorite: (id: number) =>
    client.put<{ favorite: boolean }>(`/articles/${id}/favorite`),
  /** 手动 AI 分类（同步）；body 可传 { ai_model_id } 覆盖订阅默认模型 */
  manualAIClassify: (id: number, body?: { ai_model_id?: number }) =>
    client.post<{ article: Article }>(`/articles/${id}/ai/classify`, body ?? {}),
  /** 手动翻译；无分类时服务端会同次生成分类。body 可传 { ai_model_id, ai_target_language }，空对象表示使用订阅默认 */
  manualAITranslate: (
    id: number,
    body?: { ai_model_id?: number; ai_target_language?: string }
  ) => client.post<{ article: Article }>(`/articles/${id}/ai/translate`, body ?? {}),
  /** 手动翻译（流式）：无分类时服务端会同次生成分类；onChunk 逐段接收译文，最终 onDone 返回完整文章 */
  manualAITranslateStream: async (
    id: number,
    body: { ai_model_id?: number; ai_target_language?: string },
    callbacks: {
      onChunk: (delta: string) => void;
      onDone: (article: Article) => void;
      onError: (message: string, aiLastError?: string) => void;
    }
  ): Promise<void> => {
    const token = localStorage.getItem('token');
    const base = client.defaults.baseURL ?? '/api';
    const res = await fetch(`${base}/articles/${id}/ai/translate/stream`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...(token && { Authorization: `Bearer ${token}` }),
      },
      body: JSON.stringify(body ?? {}),
    });
    if (!res.ok) {
      const data = await res.json().catch(() => ({}));
      callbacks.onError(
        (data as { error?: string }).error || res.statusText,
        (data as { ai_last_error?: string }).ai_last_error
      );
      return;
    }
    const reader = res.body?.getReader();
    if (!reader) {
      callbacks.onError('无法读取响应流');
      return;
    }
    const dec = new TextDecoder();
    let buf = '';
    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buf += dec.decode(value, { stream: true });
        const lines = buf.split('\n');
        buf = lines.pop() ?? '';
        for (const line of lines) {
          const trimmed = line.trim();
          if (!trimmed.startsWith('data:')) continue;
          const data = trimmed.slice(5).trim();
          if (!data || data === '[DONE]') continue;
          try {
            let obj: unknown = JSON.parse(data);
            // Gin 传字符串时会二次 JSON 编码，需二次解析
            if (typeof obj === 'string') obj = JSON.parse(obj);
            const o = obj as Record<string, unknown>;
            if (typeof o.delta === 'string') {
              callbacks.onChunk(o.delta);
            } else if (typeof o.error === 'string') {
              callbacks.onError(
                o.error,
                typeof o.ai_last_error === 'string' ? o.ai_last_error : undefined
              );
              return;
            } else if (o.article && typeof o.article === 'object') {
              callbacks.onDone(o.article as Article);
              return;
            }
          } catch {
            // 忽略解析错误
          }
        }
      }
    } finally {
      reader.releaseLock();
    }
  },
  /** 流式总结：通过 onChunk 逐段接收内容，onMeta 接收 article_count（onMetaAll 可选接收更多 meta） */
  summarizeStream: async (
    params: {
      ai_model_id: number;
      summary_template_id?: number;
      feed_ids?: number[];
      start_time?: string;
      end_time?: string;
      page?: number;
      page_size?: number;
      order?: 'desc' | 'asc';
    },
    callbacks: {
      onMeta: (article_count: number) => void;
      onMetaAll?: (meta: { article_count: number; total?: number; page?: number; page_size?: number; order?: string }) => void;
      onChunk: (delta: string) => void;
      onError: (message: string) => void;
    }
  ): Promise<void> => {
    const token = localStorage.getItem('token');
    const base = client.defaults.baseURL ?? '/api';
    const res = await fetch(`${base}/articles/summarize`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...(token && { Authorization: `Bearer ${token}` }),
      },
      body: JSON.stringify(params),
    });
    if (!res.ok) {
      const data = await res.json().catch(() => ({}));
      callbacks.onError((data as { error?: string }).error || res.statusText);
      return;
    }
    const reader = res.body?.getReader();
    if (!reader) {
      callbacks.onError('无法读取响应流');
      return;
    }
    const dec = new TextDecoder();
    let buf = '';
    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buf += dec.decode(value, { stream: true });
        const lines = buf.split('\n');
        buf = lines.pop() ?? '';
        for (const line of lines) {
          const trimmed = line.trim();
          if (!trimmed.startsWith('data:')) continue;
          const data = trimmed.slice(5).trim();
          if (!data || data === '[DONE]') continue;
          try {
            let obj: unknown = JSON.parse(data);
            // Gin 传字符串时会二次 JSON 编码，需二次解析
            if (typeof obj === 'string') obj = JSON.parse(obj);
            const o = obj as Record<string, unknown>;
            if (typeof o.article_count === 'number') {
              callbacks.onMeta(o.article_count);
              callbacks.onMetaAll?.({
                article_count: o.article_count,
                total: typeof o.total === 'number' ? o.total : undefined,
                page: typeof o.page === 'number' ? o.page : undefined,
                page_size: typeof o.page_size === 'number' ? o.page_size : undefined,
                order: typeof o.order === 'string' ? o.order : undefined,
              });
            } else if (typeof o.delta === 'string') {
              callbacks.onChunk(o.delta);
            } else if (typeof o.error === 'string') {
              callbacks.onError(o.error);
            }
          } catch {
            // 忽略解析错误
          }
        }
      }
    } finally {
      reader.releaseLock();
    }
  },
};

export const summaryHistoriesApi = {
  list: (params?: { page?: number; page_size?: number }) =>
    client.get<{ items: SummaryHistoryItem[]; total: number }>('/summary-histories', { params }),
  create: (params: {
    ai_model_id: number;
    summary_template_id?: number;
    summary_template_name?: string;
    feed_ids?: number[];
    start_time?: string;
    end_time?: string;
    page?: number;
    page_size?: number;
    order?: 'desc' | 'asc' | string;
    article_count?: number;
    total?: number;
    content: string;
    error?: string;
  }) => client.post<{ id: number }>('/summary-histories', params),
  retry: (id: number) => client.post<{ id: number; content: string; error: string }>(`/summary-histories/${id}/retry`),
  delete: (id: number) => client.delete(`/summary-histories/${id}`),
};

export const summarySchedulesApi = {
  list: () => client.get<SummarySchedule[]>('/summary-schedules'),
  create: (params: {
    ai_model_id: number;
    summary_template_id?: number;
    feed_ids?: number[];
    run_at: string;
    page_size?: number;
    order?: 'desc' | 'asc';
    enabled?: boolean;
  }) => client.post<SummarySchedule>('/summary-schedules', params),
  update: (
    id: number,
    params: {
      ai_model_id: number;
      summary_template_id?: number;
      feed_ids?: number[];
      run_at: string;
      page_size?: number;
      order?: 'desc' | 'asc';
      enabled?: boolean;
    }
  ) => client.put<SummarySchedule>(`/summary-schedules/${id}`, params),
  delete: (id: number) => client.delete(`/summary-schedules/${id}`),
};

export const summaryTemplatesApi = {
  list: () => client.get<{ items: SummaryTemplate[] }>('/summary-templates'),
  create: (params: { name: string; system_prompt?: string; user_prompt_prefix?: string; sort_order?: number }) =>
    client.post<SummaryTemplate>('/summary-templates', params),
  update: (
    id: number,
    params: { name: string; system_prompt?: string; user_prompt_prefix?: string; sort_order?: number }
  ) => client.put<SummaryTemplate>(`/summary-templates/${id}`, params),
  delete: (id: number) => client.delete(`/summary-templates/${id}`),
};

export const errorLogsApi = {
  list: (params?: { page?: number; page_size?: number }) =>
    client.get<{ items: ErrorLogItem[]; total: number }>('/error-logs', { params }),
  delete: (id: number) => client.delete(`/error-logs/${id}`),
};

export const adminApi = {
  listUsers: () => client.get<User[]>('/admin/users'),
  unlockUser: (id: number) => client.put(`/admin/users/${id}/unlock`),
  getFeishuBindUrl: (id: number) =>
    client.get<{ url: string; goto?: string }>(`/admin/users/${id}/feishu/bind-url`),
};

export interface UserSettings {
  feishu_notify_type: string;
  feishu_bot_webhook: string;
  feishu_id: string;
}

export const userSettingsApi = {
  get: () => client.get<UserSettings>('/users/me/settings'),
  update: (data: Partial<UserSettings>) => client.put<{ message: string }>('/users/me/settings', data),
  testFeishuBot: () => client.post<{ message: string }>('/users/me/feishu-bot/test'),
};

export default client;
