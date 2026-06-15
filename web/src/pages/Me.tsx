import { useNavigate } from 'react-router-dom';
import UiStyleControl from '../components/UiStyleControl';
import { useAuth } from '../contexts/AuthContext';
import { useTheme } from '../contexts/ThemeContext';

export default function Me() {
  const { user, logout } = useAuth();
  const { theme, toggleTheme } = useTheme();
  const navigate = useNavigate();

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
    </section>
  );
}
