import { useState, useEffect, useRef } from 'react';
import { QRCodeSVG } from 'qrcode.react';
import { adminApi } from '../api/client';
import type { User } from '../api/client';

/** 绑定飞书：与 finance 项目一致，使用 LarkSSOSDKWebQRCode 在弹框内嵌二维码，收到 tmp_code 后一次跳转完成 OAuth */

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

export default function Admin() {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [bindMsg, setBindMsg] = useState('');
  const [bindFeishuUrl, setBindFeishuUrl] = useState<string | null>(null);
  const [bindFeishuAuthUrl, setBindFeishuAuthUrl] = useState<string | null>(null);
  const feishuBindQRInstanceRef = useRef<{ matchOrigin?: (origin: string) => boolean } | null>(null);
  const feishuBindMessageHandlerRef = useRef<((e: MessageEvent) => void) | null>(null);

  const refreshUsers = () => {
    adminApi
      .listUsers()
      .then((r) => setUsers(r.data))
      .catch(() => setUsers([]));
  };

  useEffect(() => {
    adminApi
      .listUsers()
      .then((r) => setUsers(r.data))
      .catch(() => setUsers([]))
      .finally(() => setLoading(false));
  }, []);

  const unlock = async (id: number) => {
    try {
      await adminApi.unlockUser(id);
      setUsers((prev) =>
        prev.map((u) => (u.id === id ? { ...u, status: 'active' } : u))
      );
    } catch {}
  };

  const bindFeishu = async (e: React.MouseEvent, id: number) => {
    e.preventDefault();
    e.stopPropagation();
    try {
      setBindMsg('');
      const { data } = await adminApi.getFeishuBindUrl(id);
      const fullUrl = data.url.startsWith('http')
        ? data.url
        : `${window.location.origin}${data.url.startsWith('/') ? '' : '/'}${data.url}`;
      setBindFeishuUrl(fullUrl);
      setBindFeishuAuthUrl(data.goto ?? null);
    } catch {
      setBindMsg('获取飞书绑定地址失败');
    }
  };

  const closeBindFeishuModal = () => {
    setBindFeishuUrl(null);
    setBindFeishuAuthUrl(null);
    const el = document.getElementById('feishuBindQRContainer');
    if (el) el.innerHTML = '';
    const iframeContainer = document.getElementById('feishuBindIframeContainer');
    if (iframeContainer) iframeContainer.innerHTML = '';
    if (feishuBindMessageHandlerRef.current) {
      window.removeEventListener('message', feishuBindMessageHandlerRef.current);
      feishuBindMessageHandlerRef.current = null;
    }
    feishuBindQRInstanceRef.current = null;
    refreshUsers();
  };

  useEffect(() => {
    const authUrl = bindFeishuAuthUrl;
    const QRLogin = window.QRLogin;
    if (!authUrl || !QRLogin) return;
    const container = document.getElementById('feishuBindQRContainer');
    if (!container) return;
    container.innerHTML = '';
    if (feishuBindMessageHandlerRef.current) {
      window.removeEventListener('message', feishuBindMessageHandlerRef.current);
      feishuBindMessageHandlerRef.current = null;
    }
    try {
      feishuBindQRInstanceRef.current = new QRLogin({
        id: 'feishuBindQRContainer',
        goto: authUrl,
        width: 280,
        height: 280,
        style: 'width:280px;height:280px;',
      });
      const handler = (event: MessageEvent) => {
        // 绑定结果：回调页在 iframe 中可能与本页不同源（如前端 5173 / 后端 8080），仅根据 type 识别
        const t = event.data?.type;
        if (t === 'feishu_bind_success') {
          window.removeEventListener('message', handler);
          feishuBindMessageHandlerRef.current = null;
          closeBindFeishuModal();
          setBindMsg('绑定成功');
          setTimeout(() => setBindMsg(''), 2500);
          return;
        }
        if (t === 'feishu_bind_error') {
          window.removeEventListener('message', handler);
          feishuBindMessageHandlerRef.current = null;
          closeBindFeishuModal();
          setBindMsg(event.data?.message ?? '绑定失败');
          setTimeout(() => setBindMsg(''), 3500);
          return;
        }
        // 飞书 SDK 传来的 tmp_code：在隐藏 iframe 中完成 OAuth，不跳转主页面（必须校验飞书来源）
        const instance = feishuBindQRInstanceRef.current;
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
          const container = document.getElementById('feishuBindIframeContainer');
          if (container) {
            const iframe = document.createElement('iframe');
            iframe.setAttribute('src', iframeSrc);
            iframe.setAttribute('title', '飞书绑定');
            iframe.style.cssText = 'position:absolute;width:0;height:0;border:0;visibility:hidden';
            container.appendChild(iframe);
          }
        }
      };
      feishuBindMessageHandlerRef.current = handler;
      window.addEventListener('message', handler);
    } catch (_) {}
    return () => {
      const el = document.getElementById('feishuBindQRContainer');
      if (el) el.innerHTML = '';
      if (feishuBindMessageHandlerRef.current) {
        window.removeEventListener('message', feishuBindMessageHandlerRef.current);
        feishuBindMessageHandlerRef.current = null;
      }
      feishuBindQRInstanceRef.current = null;
    };
  }, [bindFeishuAuthUrl]);

  return (
    <div className="admin-page">
      <h2>用户管理</h2>
      {bindMsg && (
        <p className={bindMsg === '绑定成功' ? 'bind-msg-success' : 'error'}>{bindMsg}</p>
      )}
      {loading ? (
        <p>加载中...</p>
      ) : (
        <table className="user-table">
          <thead>
            <tr>
              <th>ID</th>
              <th>用户名</th>
              <th>状态</th>
              <th>超级管理员</th>
              <th>飞书绑定</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {users.map((u) => (
              <tr key={u.id}>
                <td>{u.id}</td>
                <td>{u.username}</td>
                <td>{u.status === 'locked' ? '锁定' : '正常'}</td>
                <td>{u.is_super_admin ? '是' : '否'}</td>
                <td>{u.feishu_id ? '已绑定' : '未绑定'}</td>
                <td>
                  {u.status === 'locked' && (
                    <button onClick={() => unlock(u.id)}>解锁</button>
                  )}
                  <button
                    type="button"
                    onClick={(ev) => bindFeishu(ev, u.id)}
                    style={{ marginLeft: 8 }}
                  >
                    绑定飞书
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {bindFeishuUrl != null && (
        <div
          className="bind-feishu-overlay"
          onClick={closeBindFeishuModal}
          role="presentation"
        >
          <div
            className="bind-feishu-modal"
            onClick={(e) => e.stopPropagation()}
            role="dialog"
            aria-labelledby="bind-feishu-title"
            style={{ maxWidth: '420px' }}
          >
            <h3 id="bind-feishu-title" className="bind-feishu-title">
              <span className="bind-feishu-icon" aria-hidden>品</span>
              绑定飞书
            </h3>
            <p className="bind-feishu-desc">
              使用飞书 App 扫码，可将飞书账号绑定到当前用户
            </p>
            <div id="feishuBindIframeContainer" style={{ position: 'absolute', width: 0, height: 0, overflow: 'hidden' }} aria-hidden />
            {bindFeishuAuthUrl ? (
              <div
                id="feishuBindQRContainer"
                className="bind-feishu-qr-sdk"
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
            ) : (
              <div className="bind-feishu-qr">
                <QRCodeSVG value={bindFeishuUrl} size={260} level="M" />
              </div>
            )}
            <button
              type="button"
              className="bind-feishu-close"
              onClick={closeBindFeishuModal}
            >
              关闭
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
