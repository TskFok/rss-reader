import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import UiStyleControl from '../components/UiStyleControl';
import Modal from '../components/Modal';
import { useAuth } from '../contexts/AuthContext';
import { useTheme } from '../contexts/ThemeContext';
import { useToast } from '../contexts/ToastContext';
import { userSettingsApi } from '../api/client';

export default function Me() {
  const { user, logout } = useAuth();
  const { theme, toggleTheme } = useTheme();
  const navigate = useNavigate();
  const toast = useToast();

  const [passwordLoginEnabled, setPasswordLoginEnabled] = useState(true);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!user?.is_super_admin) return;
    userSettingsApi
      .get()
      .then(({ data }) => {
        if (typeof data.password_login_enabled === 'boolean') {
          setPasswordLoginEnabled(data.password_login_enabled);
        }
      })
      .catch(() => {
        toast.showToast({ message: '加载账号密码登录设置失败', variant: 'error' });
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user?.is_super_admin]);

  const savePasswordLoginEnabled = async (enabled: boolean) => {
    setSaving(true);
    try {
      await userSettingsApi.update({ password_login_enabled: enabled });
      setPasswordLoginEnabled(enabled);
      setConfirmOpen(false);
    } catch {
      toast.showToast({ message: '保存失败，请重试', variant: 'error' });
    } finally {
      setSaving(false);
    }
  };

  const handleTogglePasswordLogin = () => {
    if (passwordLoginEnabled) {
      setConfirmOpen(true);
    } else {
      savePasswordLoginEnabled(true);
    }
  };

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  return (
    <section className="me-page">
      <div className="me-panel">
        <div className="me-panel-header">
          <div>
            <h1>我的</h1>
            <p>账号与界面偏好</p>
          </div>
          <span className="me-user-name">{user?.username}</span>
        </div>

        <div className="me-settings-list">
          <div className="me-setting-row">
            <div>
              <h2>界面风格</h2>
              <p>切换阅读器的整体视觉风格</p>
            </div>
            <UiStyleControl variant="header" />
          </div>

          <div className="me-setting-row">
            <div>
              <h2>颜色模式</h2>
              <p>在浅色与深色模式之间切换</p>
            </div>
            <button
              type="button"
              className="nice-admin-header-theme"
              onClick={toggleTheme}
              aria-label={theme === 'dark' ? '切换到浅色模式' : '切换到深色模式'}
              title={theme === 'dark' ? '切换到浅色模式' : '切换到深色模式'}
            >
              {theme === 'dark' ? '浅色' : '深色'}
            </button>
          </div>

          {user?.is_super_admin && (
            <div className="me-setting-row">
              <div>
                <h2>账号密码登录</h2>
                <p>关闭后全站只能使用飞书登录和注册</p>
              </div>
              <button
                type="button"
                className="nice-admin-header-theme"
                onClick={handleTogglePasswordLogin}
                disabled={saving}
              >
                {passwordLoginEnabled ? '已开启' : '已关闭'}
              </button>
            </div>
          )}

          <div className="me-setting-row">
            <div>
              <h2>登录状态</h2>
              <p>退出当前账号并返回登录页</p>
            </div>
            <button type="button" onClick={handleLogout} className="nice-admin-header-logout">
              退出
            </button>
          </div>
        </div>
      </div>

      <Modal open={confirmOpen} onClose={() => setConfirmOpen(false)} title="关闭账号密码登录">
        <p>关闭后将同时关闭注册，全站只能使用飞书登录，且不会检查飞书是否可用，请确认后再关闭。</p>
        <div className="feeds-modal-actions">
          <button type="button" onClick={() => setConfirmOpen(false)}>取消</button>
          <button type="button" disabled={saving} onClick={() => savePasswordLoginEnabled(false)}>
            确认关闭
          </button>
        </div>
      </Modal>
    </section>
  );
}
