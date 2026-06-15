import { useEffect, useState } from 'react';
import { Outlet, Link, useLocation, useSearchParams } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import {
  getInitialSidebarCollapsed,
  setStoredSidebarCollapsed,
} from '../utils/sidebarCollapsed';
import { navLinkIsActive } from '../utils/navLinkActive';

const mainNavItems: { to: string; label: string }[] = [
  { to: '/', label: '首页' },
  { to: '/favorites', label: '收藏' },
  { to: '/summary-history', label: '总结历史' },
  { to: '/feeds?tab=ai-summary', label: 'AI 总结' },
  { to: '/me', label: '我的' },
];

const feedsTabItems: { tab: string; label: string; icon: string; superAdminOnly?: boolean }[] = [
  { tab: 'categories', label: '订阅分类', icon: '分' },
  { tab: 'feeds', label: '订阅列表', icon: '订' },
  { tab: 'proxies', label: '代理', icon: '代' },
  { tab: 'ai-models', label: 'AI 模型', icon: '模' },
  { tab: 'summary-templates', label: '总结模版', icon: '版' },
  { tab: 'ai-summary-schedule', label: '定时总结', icon: '时' },
  { tab: 'feishu', label: '飞书机器人', icon: '飞' },
  { tab: 'users', label: '用户管理', icon: '用', superAdminOnly: true },
];

export default function Layout() {
  const { user } = useAuth();
  const location = useLocation();
  const [searchParams] = useSearchParams();
  const feedsTab = location.pathname === '/feeds' ? searchParams.get('tab') || 'feeds' : null;
  const [sidebarCollapsed, setSidebarCollapsed] = useState(() => getInitialSidebarCollapsed());

  useEffect(() => {
    setStoredSidebarCollapsed(sidebarCollapsed);
  }, [sidebarCollapsed]);

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
              className={`nice-admin-sidebar-link ${navLinkIsActive(to, location.pathname, location.search) ? 'active' : ''}`}
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
      <button
        type="button"
        className="nice-admin-mobile-menu-btn"
        onClick={() => setSidebarCollapsed((c) => !c)}
        aria-label="打开或关闭侧边导航"
      >
        菜单
      </button>
      <div className="nice-admin-main-wrap">
        <main className="nice-admin-content">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
