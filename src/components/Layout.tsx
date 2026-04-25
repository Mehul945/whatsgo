import { Link, useLocation, useNavigate } from 'react-router';
import {
  LayoutDashboard,
  Smartphone,
  MessageSquare,
  Key,
  Settings,
  LogOut,
  ChevronLeft,
  ChevronRight,
  Shield,
  Wifi,
  WifiOff,
  Loader2,
} from 'lucide-react';
import { useAuthStore } from '@/stores/auth';
import { useUIStore } from '@/stores/ui';
import { cn } from '@/lib/utils';

const navItems = [
  { path: '/', label: 'Dashboard', icon: LayoutDashboard },
  { path: '/devices', label: 'Devices', icon: Smartphone },
  { path: '/messages', label: 'Messages', icon: MessageSquare },
  { path: '/api-keys', label: 'API & Webhooks', icon: Key },
  { path: '/settings', label: 'Settings', icon: Settings },
];

export default function Layout({ children }: { children: React.ReactNode }) {
  const location = useLocation();
  const navigate = useNavigate();
  const { user, logout, isAuthenticated } = useAuthStore();
  const { sidebarCollapsed, toggleSidebar } = useUIStore();

  if (!isAuthenticated) {
    return <>{children}</>;
  }

  return (
    <div className="flex h-screen bg-[#f8f9fa]">
      {/* Sidebar */}
      <aside
        className={cn(
          'bg-white border-r border-[#e9ecef] flex flex-col transition-all duration-300',
          sidebarCollapsed ? 'w-16' : 'w-64'
        )}
      >
        {/* Logo */}
        <div className="h-16 flex items-center px-4 border-b border-[#e9ecef]">
          <div className="w-8 h-8 rounded-lg bg-[#25D366] flex items-center justify-center">
            <MessageSquare className="w-4 h-4 text-white" />
          </div>
          {!sidebarCollapsed && (
            <span className="ml-3 font-bold text-lg text-[#1c1e21]">WhatsGo</span>
          )}
        </div>

        {/* Nav Items */}
        <nav className="flex-1 py-4 px-2 space-y-1">
          {navItems.map((item) => {
            const isActive = location.pathname === item.path;
            const Icon = item.icon;
            return (
              <Link
                key={item.path}
                to={item.path}
                className={cn(
                  'flex items-center px-3 py-2.5 rounded-lg text-sm font-medium transition-colors',
                  isActive
                    ? 'bg-[#dcf8c6] text-[#128C7E] border-l-2 border-[#25D366]'
                    : 'text-[#65676b] hover:bg-[#f8f9fa] hover:text-[#1c1e21]'
                )}
              >
                <Icon className={cn('w-5 h-5', sidebarCollapsed ? '' : 'mr-3')} />
                {!sidebarCollapsed && <span>{item.label}</span>}
              </Link>
            );
          })}
        </nav>

        {/* Admin link */}
        {user?.role === 'admin' && (
          <div className="px-2 pb-2">
            <Link
              to="/admin"
              className={cn(
                'flex items-center px-3 py-2.5 rounded-lg text-sm font-medium transition-colors',
                location.pathname === '/admin'
                  ? 'bg-[#dcf8c6] text-[#128C7E] border-l-2 border-[#25D366]'
                  : 'text-[#65676b] hover:bg-[#f8f9fa] hover:text-[#1c1e21]'
              )}
            >
              <Shield className={cn('w-5 h-5', sidebarCollapsed ? '' : 'mr-3')} />
              {!sidebarCollapsed && <span>Admin</span>}
            </Link>
          </div>
        )}

        {/* Collapse button */}
        <div className="p-2 border-t border-[#e9ecef]">
          <button
            onClick={toggleSidebar}
            className="w-full flex items-center justify-center p-2 rounded-lg hover:bg-[#f8f9fa] text-[#65676b]"
          >
            {sidebarCollapsed ? (
              <ChevronRight className="w-5 h-5" />
            ) : (
              <>
                <ChevronLeft className="w-5 h-5 mr-2" />
                <span className="text-sm">Collapse</span>
              </>
            )}
          </button>
        </div>

        {/* User */}
        <div className="p-3 border-t border-[#e9ecef]">
          <div className="flex items-center">
            <div className="w-8 h-8 rounded-full bg-[#25D366] flex items-center justify-center text-white text-sm font-bold">
              {user?.name?.charAt(0)?.toUpperCase() || 'U'}
            </div>
            {!sidebarCollapsed && (
              <div className="ml-3 flex-1 min-w-0">
                <p className="text-sm font-medium text-[#1c1e21] truncate">{user?.name}</p>
                <p className="text-xs text-[#65676b] truncate">{user?.email}</p>
              </div>
            )}
            {!sidebarCollapsed && (
              <button
                onClick={() => {
                  logout();
                  navigate('/login');
                }}
                className="ml-2 p-1.5 rounded-lg hover:bg-[#f8f9fa] text-[#65676b]"
              >
                <LogOut className="w-4 h-4" />
              </button>
            )}
          </div>
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex-1 flex flex-col overflow-hidden">
        {/* Top bar */}
        <header className="h-16 bg-white border-b border-[#e9ecef] flex items-center justify-between px-6">
          <h1 className="text-xl font-semibold text-[#1c1e21]">
            {navItems.find((n) => n.path === location.pathname)?.label || 'WhatsGo'}
          </h1>
          <div className="flex items-center gap-3">
            <StatusIndicator />
          </div>
        </header>

        {/* Page content */}
        <div className="flex-1 overflow-auto p-6">{children}</div>
      </main>
    </div>
  );
}

function StatusIndicator() {
  // This would ideally check WebSocket connection status
  return (
    <div className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-[#f8f9fa] text-xs font-medium text-[#65676b]">
      <Wifi className="w-3.5 h-3.5 text-[#25D366]" />
      <span>Connected</span>
    </div>
  );
}

export function StatusBadge({ status }: { status: string }) {
  if (status === 'connected') {
    return (
      <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-[#dcf8c6] text-[#128C7E]">
        <Wifi className="w-3 h-3" />
        Online
      </span>
    );
  }
  if (status === 'connecting') {
    return (
      <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-amber-100 text-amber-700">
        <Loader2 className="w-3 h-3 animate-spin" />
        Connecting
      </span>
    );
  }
  return (
    <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-[#e9ecef] text-[#65676b]">
      <WifiOff className="w-3 h-3" />
      Offline
    </span>
  );
}
