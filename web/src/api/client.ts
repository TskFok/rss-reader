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
    expire_days?: number
  ) =>
    client.put<Feed>(`/feeds/${id}`, {
      update_interval_minutes,
      proxy_id: proxy_id ?? null,
      ...(expire_days !== undefined && { expire_days }),
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
};

export const proxiesApi = {
  list: () => client.get<Proxy[]>('/proxies'),
  create: (name: string, url: string) =>
    client.post<Proxy>('/proxies', { name, url }),
  update: (id: number, name: string, url: string) =>
    client.put<Proxy>(`/proxies/${id}`, { name, url }),
  delete: (id: number) => client.delete(`/proxies/${id}`),
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
};

export const adminApi = {
  listUsers: () => client.get<User[]>('/admin/users'),
  unlockUser: (id: number) => client.put(`/admin/users/${id}/unlock`),
  getFeishuBindUrl: (id: number) =>
    client.get<{ url: string; goto?: string }>(`/admin/users/${id}/feishu/bind-url`),
};

export default client;
