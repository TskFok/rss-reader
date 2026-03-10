import { Outlet, Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';

export default function Layout() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  return (
    <div className="layout">
      <nav className="nav">
        <Link to="/">首页</Link>
        <Link to="/favorites">收藏</Link>
        <Link to="/feeds">系统设置</Link>
        <span className="user">
          {user?.username}
          <button onClick={handleLogout} className="logout-btn">
            退出
          </button>
        </span>
      </nav>
      <main className="main">
        <Outlet />
      </main>
    </div>
  );
}
