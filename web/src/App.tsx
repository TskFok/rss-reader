import { BrowserRouter, Routes, Route, Navigate, useLocation } from 'react-router-dom';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import { ThemeProvider } from './contexts/ThemeContext';
import Login from './pages/Login';
import Register from './pages/Register';
import Home from './pages/Home';
import Favorites from './pages/Favorites';
import Feeds from './pages/Feeds';
import SummaryHistory from './pages/SummaryHistory';
import ErrorLogs from './pages/ErrorLogs';
import Layout from './components/Layout';

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  useAuth(); // ensure we're inside AuthProvider
  const location = useLocation();
  if (typeof window === 'undefined') return <div className="loading">加载中...</div>;
  if (!localStorage.getItem('token')) {
    return <Navigate to="/login" replace state={{ from: location }} />;
  }
  return <>{children}</>;
}

function AppRoutes() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/register" element={<Register />} />
      <Route
        path="/"
        element={
          <ProtectedRoute>
            <Layout />
          </ProtectedRoute>
        }
      >
        <Route index element={<Home />} />
        <Route path="favorites" element={<Favorites />} />
        <Route path="feeds" element={<Feeds />} />
        <Route path="summary-history" element={<SummaryHistory />} />
        <Route path="error-logs" element={<ErrorLogs />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}

export default function App() {
  return (
    <ThemeProvider>
      <AuthProvider>
        <BrowserRouter>
          <AppRoutes />
        </BrowserRouter>
      </AuthProvider>
    </ThemeProvider>
  );
}
