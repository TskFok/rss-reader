import { useState } from 'react';
import { Outlet, Link, useNavigate, useLocation, useSearchParams } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { useTheme } from '../contexts/ThemeContext';

const mainNavItems: { to: string; label: string }[] = [
  { to: '/', label: '首页' },
  { to: '/favorites', label: '收藏' },
  { to: '/summary-history', label: '总结历史' },
  { to: '/feeds?tab=ai-summary', label: 'AI 总结' },
];

const feedsTabItems: { tab: string; label: string; icon: string; superAdminOnly?: boolean }[] = [
  { tab: 'categories', label: '订阅分类', icon: '分' },
  { tab: 'feeds', label: '订阅列表', icon: '订' },
  { tab: 'proxies', label: '代理', icon: '代' },
  { tab: 'ai-models', label: 'AI 模型', icon: '模' },
  { tab: 'ai-summary-schedule', label: '定时总结', icon: '时' },
  { tab: 'users', label: '用户管理', icon: '用', superAdminOnly: true },
];

export default function Layout() {
  const { user, logout } = useAuth();
  const { theme, toggleTheme } = useTheme();
  const navigate = useNavigate();
  const location = useLocation();
  const [searchParams] = useSearchParams();
  const feedsTab = location.pathname === '/feeds' ? searchParams.get('tab') || 'feeds' : null;
  const [sidebarCollapsed, setSidebarCollapsed] = useState(() =>
    typeof window !== 'undefined' && window.innerWidth <= 768
  );

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  const feedsNavItems = feedsTabItems.filter((item) => !item.superAdminOnly || user?.is_super_admin);

  return (
    <div className="nice-admin-layout">
      <aside className={`nice-admin-sidebar ${sidebarCollapsed ? 'collapsed' : ''}`}>
        <div className="nice-admin-sidebar-brand">
          <span className="nice-admin-sidebar-logo">RSS</span>
          {!sidebarCollapsed && <span className="nice-admin-sidebar-title">RSS Reader</span>}
        </div>
        <nav className="nice-admin-sidebar-nav">
          {mainNavItems.map(({ to, label }) => (
            <Link
              key={to}
              to={to}
              className={`nice-admin-sidebar-link ${location.pathname === to || (to !== '/' && location.pathname.startsWith(to)) ? 'active' : ''}`}
            >
              <span className="nice-admin-sidebar-icon">{label.slice(0, 1)}</span>
              {!sidebarCollapsed && <span>{label}</span>}
            </Link>
          ))}
          {!sidebarCollapsed && (
            <div className="nice-admin-sidebar-group-title">系统设置</div>
          )}
          {feedsNavItems.map(({ tab, label, icon }) => (
            <Link
              key={tab}
              to={`/feeds?tab=${tab}`}
              className={`nice-admin-sidebar-link ${feedsTab === tab ? 'active' : ''}`}
            >
              <span className="nice-admin-sidebar-icon">{icon}</span>
              {!sidebarCollapsed && <span>{label}</span>}
            </Link>
          ))}
          <Link
            to="/error-logs"
            className={`nice-admin-sidebar-link ${location.pathname.startsWith('/error-logs') ? 'active' : ''}`}
          >
            <span className="nice-admin-sidebar-icon">错</span>
            {!sidebarCollapsed && <span>错误日志</span>}
          </Link>
        </nav>
        <button
          type="button"
          className="nice-admin-sidebar-toggle"
          onClick={() => setSidebarCollapsed((c) => !c)}
          aria-label={sidebarCollapsed ? '展开侧边栏' : '收起侧边栏'}
        >
          {sidebarCollapsed ? '›' : '‹'}
        </button>
      </aside>
      <div className="nice-admin-main-wrap">
        <header className="nice-admin-header">
          <div className="nice-admin-header-left">
            <button
              type="button"
              className="nice-admin-header-menu-btn"
              onClick={() => setSidebarCollapsed((c) => !c)}
              aria-label="切换侧边栏"
            >
              ☰
            </button>
            <span className="nice-admin-header-breadcrumb">
              {location.pathname === '/feeds' && feedsTab
                ? (feedsTab === 'ai-summary'
                  ? 'AI 总结'
                  : (feedsNavItems.find((i) => i.tab === feedsTab)?.label ?? '系统设置'))
                : mainNavItems.find((i) => i.to === location.pathname || (i.to !== '/' && location.pathname.startsWith(i.to)))?.label ?? '首页'}
            </span>
          </div>
          <div className="nice-admin-header-right">
            <button
              type="button"
              className="nice-admin-header-theme"
              onClick={toggleTheme}
              aria-label="切换明暗模式"
              title={theme === 'dark' ? '切换到浅色模式' : '切换到深色模式'}
            >
              {theme === 'dark' ? '☀️' : '🌙'}
            </button>
            <span className="nice-admin-header-user">{user?.username}</span>
            <button type="button" onClick={handleLogout} className="nice-admin-header-logout">
              退出
            </button>
          </div>
        </header>
        <main className="nice-admin-content">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
