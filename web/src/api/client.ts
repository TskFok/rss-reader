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
  sort_order?: number;
  created_at: string;
  updated_at: string;
}

export interface Article {
  id: number;
  feed_id: number;
  guid: string;
  title: string;
  link: string;
  content: string;
  published_at: string | null;
  created_at: string;
  read: boolean;
  favorite?: boolean;
  feed_title?: string;
}

export interface SummaryHistoryItem {
  id: number;
  ai_model_id: number;
  ai_model_name: string;
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

export interface SummarySchedule {
  id: number;
  user_id: number;
  ai_model_id: number;
  feed_ids_json: string;
  run_at: string; // HH:MM
  page_size: number;
  order: 'desc' | 'asc' | string;
  enabled: boolean;
  last_run_at?: string | null;
  created_at: string;
  updated_at: string;
}

export const authApi = {
  register: (username: string, password: string) =>
    client.post('/auth/register', { username, password }),
  login: (username: string, password: string) =>
    client.post<{ token: string; user: User }>('/auth/login', { username, password }),
  getFeishuLoginUrl: () =>
    client.get<{ url: string; goto: string }>('/auth/feishu/login-url'),
};

export const feedsApi = {
  list: () => client.get<Feed[]>('/feeds'),
  create: (
    url: string,
    category_id: number,
    update_interval_minutes: number,
    proxy_id?: number | null,
    expire_days?: number
  ) =>
    client.post<Feed>('/feeds', {
      url,
      category_id,
      update_interval_minutes,
      proxy_id: proxy_id ?? null,
      expire_days: expire_days ?? 90,
    }),
  update: (
    id: number,
    update_interval_minutes: number,
    proxy_id?: number | null,
    expire_days?: number,
    category_id?: number | null
  ) =>
    client.put<Feed>(`/feeds/${id}`, {
      update_interval_minutes,
      proxy_id: proxy_id ?? null,
      ...(expire_days !== undefined && { expire_days }),
      ...(category_id !== undefined && { category_id }),
    }),
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
  create: (name: string, base_url: string, api_key?: string) =>
    client.post<AIModel>('/ai-models', { name, base_url, api_key: api_key ?? '' }),
  update: (id: number, name: string, base_url: string, api_key?: string | null) =>
    client.put<AIModel>(`/ai-models/${id}`, {
      name,
      base_url,
      ...(api_key !== undefined && { api_key: api_key ?? '' }),
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
  /** 流式总结：通过 onChunk 逐段接收内容，onMeta 接收 article_count（onMetaAll 可选接收更多 meta） */
  summarizeStream: async (
    params: {
      ai_model_id: number;
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
  delete: (id: number) => client.delete(`/summary-histories/${id}`),
};

export const summarySchedulesApi = {
  list: () => client.get<SummarySchedule[]>('/summary-schedules'),
  create: (params: { ai_model_id: number; feed_ids?: number[]; run_at: string; page_size?: number; order?: 'desc' | 'asc'; enabled?: boolean }) =>
    client.post<SummarySchedule>('/summary-schedules', params),
  update: (id: number, params: { ai_model_id: number; feed_ids?: number[]; run_at: string; page_size?: number; order?: 'desc' | 'asc'; enabled?: boolean }) =>
    client.put<SummarySchedule>(`/summary-schedules/${id}`, params),
  delete: (id: number) => client.delete(`/summary-schedules/${id}`),
};

export const adminApi = {
  listUsers: () => client.get<User[]>('/admin/users'),
  unlockUser: (id: number) => client.put(`/admin/users/${id}/unlock`),
  getFeishuBindUrl: (id: number) =>
    client.get<{ url: string; goto?: string }>(`/admin/users/${id}/feishu/bind-url`),
};

export default client;
