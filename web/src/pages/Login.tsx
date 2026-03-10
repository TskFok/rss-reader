import { useState, useEffect, useRef } from 'react';
import { Link, useNavigate, useLocation } from 'react-router-dom';
import { authApi } from '../api/client';
import { useAuth } from '../contexts/AuthContext';

const FEISHU_QR_VALID_ORIGINS = [
  'https://accounts.feishu.cn',
  'https://open.feishu.cn',
  'https://passport.feishu.cn',
  'https://www.feishu.cn',
  'https://login.feishu.cn',
  'https://sf3-cn.feishucdn.com',
];

declare global {
  interface Window {
    QRLogin?: new (opt: {
      id: string;
      goto: string;
      width: number;
      height: number;
      style?: string;
    }) => { matchOrigin?: (origin: string) => boolean };
  }
}

export default function Login() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [message, setMessage] = useState('');
  const [loading, setLoading] = useState(false);
  const [mode, setMode] = useState<'password' | 'feishu'>('password');
  const [feishuGoto, setFeishuGoto] = useState<string | null>(null);
  const navigate = useNavigate();
  const location = useLocation();
  const { setUser } = useAuth();
  const feishuQRInstanceRef = useRef<{ matchOrigin?: (origin: string) => boolean } | null>(null);
  const feishuMessageHandlerRef = useRef<((e: MessageEvent) => void) | null>(null);

  useEffect(() => {
    if (localStorage.getItem('token')) {
      navigate('/', { replace: true });
      return;
    }
    const msg = (location.state as { message?: string })?.message;
    if (msg) setMessage(msg);
  }, [location, navigate]);

  // 选择飞书登录时自动获取并展示二维码
  useEffect(() => {
    if (mode === 'feishu') {
      setError('');
      authApi
        .getFeishuLoginUrl()
        .then(({ data }) => setFeishuGoto(data.goto ?? null))
        .catch(() => setError('获取飞书登录地址失败'));
    } else {
      clearFeishuQR();
    }
  }, [mode]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (mode !== 'password') return;
    setError('');
    setLoading(true);
    try {
      const { data } = await authApi.login(username, password);
      localStorage.setItem('token', data.token);
      setUser(data.user);
      navigate('/');
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
      setError(msg || '登录失败');
    } finally {
      setLoading(false);
    }
  };

  const clearFeishuQR = () => {
    setFeishuGoto(null);
    const el = document.getElementById('feishuLoginQRContainer');
    if (el) el.innerHTML = '';
    const iframeContainer = document.getElementById('feishuLoginIframeContainer');
    if (iframeContainer) iframeContainer.innerHTML = '';
    if (feishuMessageHandlerRef.current) {
      window.removeEventListener('message', feishuMessageHandlerRef.current);
      feishuMessageHandlerRef.current = null;
    }
    feishuQRInstanceRef.current = null;
  };

  useEffect(() => {
    const authUrl = feishuGoto;
    const QRLogin = window.QRLogin;
    if (!authUrl || !QRLogin) return;
    const container = document.getElementById('feishuLoginQRContainer');
    if (!container) return;
    container.innerHTML = '';
    if (feishuMessageHandlerRef.current) {
      window.removeEventListener('message', feishuMessageHandlerRef.current);
      feishuMessageHandlerRef.current = null;
    }
    try {
      feishuQRInstanceRef.current = new QRLogin({
        id: 'feishuLoginQRContainer',
        goto: authUrl,
        width: 280,
        height: 280,
        style: 'width:280px;height:280px;',
      });
      const handler = (event: MessageEvent) => {
        const t = event.data?.type;
        if (t === 'feishu_login_success') {
          window.removeEventListener('message', handler);
          feishuMessageHandlerRef.current = null;
          clearFeishuQR();
          try {
            const token = event.data?.token;
            const user = event.data?.user;
            if (token && user) {
              localStorage.setItem('token', token);
              localStorage.setItem('user', JSON.stringify(user));
              setUser(user);
              navigate('/');
            } else {
              setError('登录结果异常');
            }
          } catch {
            setError('登录结果处理失败');
          }
          return;
        }
        if (t === 'feishu_login_error') {
          window.removeEventListener('message', handler);
          feishuMessageHandlerRef.current = null;
          setError(event.data?.message ?? '飞书登录失败');
          return;
        }
        const instance = feishuQRInstanceRef.current;
        const validOrigin =
          instance && typeof instance.matchOrigin === 'function'
            ? instance.matchOrigin(event.origin)
            : FEISHU_QR_VALID_ORIGINS.some((o) => event.origin === o);
        if (!validOrigin) return;
        const raw = event.data;
        const tmpCode =
          typeof raw === 'string' ? raw : (raw && (raw as { tmp_code?: string }).tmp_code ? (raw as { tmp_code: string }).tmp_code : null);
        if (tmpCode && /^[a-zA-Z0-9_-]+$/.test(tmpCode)) {
          const sep = authUrl.indexOf('?') >= 0 ? '&' : '?';
          const iframeSrc = authUrl + sep + 'tmp_code=' + encodeURIComponent(tmpCode);
          const iframeContainer = document.getElementById('feishuLoginIframeContainer');
          if (iframeContainer) {
            const iframe = document.createElement('iframe');
            iframe.setAttribute('src', iframeSrc);
            iframe.setAttribute('title', '飞书登录');
            iframe.style.cssText = 'position:absolute;width:0;height:0;border:0;visibility:hidden';
            iframeContainer.appendChild(iframe);
          }
        }
      };
      feishuMessageHandlerRef.current = handler;
      window.addEventListener('message', handler);
    } catch {
      setError('初始化飞书扫码失败');
    }
    return () => {
      const el = document.getElementById('feishuLoginQRContainer');
      if (el) el.innerHTML = '';
      if (feishuMessageHandlerRef.current) {
        window.removeEventListener('message', feishuMessageHandlerRef.current);
        feishuMessageHandlerRef.current = null;
      }
      feishuQRInstanceRef.current = null;
    };
  }, [feishuGoto, navigate, setUser]);

  return (
    <div className="auth-page">
      <h1>RSS 阅读器</h1>
      <div className="login-tabs">
        <button
          type="button"
          className={mode === 'password' ? 'active' : ''}
          onClick={() => setMode('password')}
        >
          账号密码登录
        </button>
        <button
          type="button"
          className={mode === 'feishu' ? 'active' : ''}
          onClick={() => setMode('feishu')}
        >
          飞书登录
        </button>
      </div>
      {mode === 'password' ? (
        <form onSubmit={handleSubmit} className="auth-form">
          <input
            type="text"
            placeholder="用户名"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            required
            autoComplete="username"
          />
          <input
            type="password"
            placeholder="密码"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            autoComplete="current-password"
          />
          {message && <p className="message">{message}</p>}
          {error && <p className="error">{error}</p>}
          <button type="submit" disabled={loading}>
            {loading ? '登录中...' : '登录'}
          </button>
        </form>
      ) : (
        <div className="auth-form auth-form-feishu">
          {message && <p className="message">{message}</p>}
          {error && <p className="error">{error}</p>}
          {feishuGoto == null && !error && (
            <p className="feishu-qr-hint" style={{ marginTop: 8 }}>
              正在加载飞书扫码...
            </p>
          )}
          {feishuGoto != null && (
            <>
              <div id="feishuLoginIframeContainer" style={{ position: 'absolute', width: 0, height: 0, overflow: 'hidden' }} aria-hidden />
              <div
                id="feishuLoginQRContainer"
                className="feishu-qr-inline"
                style={{
                  width: 280,
                  height: 280,
                  margin: '16px auto',
                  background: '#1a1a1a',
                  border: '1px solid var(--border, #2a2a4a)',
                  borderRadius: 8,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  overflow: 'hidden',
                }}
              />
              <p className="feishu-qr-hint">使用飞书 App 扫码即可登录</p>
            </>
          )}
        </div>
      )}
      <p>
        还没有账号？ <Link to="/register">注册</Link>
      </p>
    </div>
  );
}
