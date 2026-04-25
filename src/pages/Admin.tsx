import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router';
import {
  Users,
  Smartphone,
  Clock,
  ArrowLeft,
  CheckCircle,
  XCircle,
} from 'lucide-react';
import { api } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';
import type { User } from '@/types';

export default function Admin() {
  const navigate = useNavigate();
  const { user } = useAuthStore();
  const [users, setUsers] = useState<User[]>([]);
  const [stats, setStats] = useState({
    totalUsers: 0,
    totalDevices: 0,
    totalMessages: 0,
    activeNow: 0,
  });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (user?.role !== 'admin') {
      navigate('/');
      return;
    }
    loadData();
  }, [user]);

  const loadData = async () => {
    setLoading(true);
    const res = await api.request<User[]>('/api/admin/users');
    if (res.success && res.data) {
      setUsers(res.data);
      setStats({
        totalUsers: res.data.length,
        totalDevices: res.data.reduce((acc, u) => acc + (u as any).device_count || 0, 0),
        totalMessages: 0,
        activeNow: res.data.filter((u) => u.active).length,
      });
    }
    setLoading(false);
  };

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="bg-white rounded-lg border border-[#e9ecef] p-5 animate-pulse">
              <div className="h-8 bg-[#e9ecef] rounded w-16 mb-4" />
              <div className="h-10 bg-[#e9ecef] rounded w-24" />
            </div>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <button
          onClick={() => navigate('/')}
          className="p-2 hover:bg-[#f8f9fa] rounded-lg text-[#65676b] transition-colors"
        >
          <ArrowLeft className="w-5 h-5" />
        </button>
        <div>
          <h2 className="text-lg font-semibold text-[#1c1e21]">Admin Panel</h2>
          <p className="text-sm text-[#65676b]">System overview and user management</p>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        {[
          { label: 'Total Users', value: stats.totalUsers, icon: Users, color: 'text-blue-500', bg: 'bg-blue-50' },
          { label: 'Active Users', value: stats.activeNow, icon: CheckCircle, color: 'text-[#25D366]', bg: 'bg-[#dcf8c6]' },
          { label: 'Total Devices', value: stats.totalDevices, icon: Smartphone, color: 'text-purple-500', bg: 'bg-purple-50' },
          { label: 'Server Uptime', value: '99.9%', icon: Clock, color: 'text-orange-500', bg: 'bg-orange-50' },
        ].map((stat) => {
          const Icon = stat.icon;
          return (
            <div key={stat.label} className="bg-white rounded-lg p-5 border border-[#e9ecef] shadow-sm">
              <div className={`w-10 h-10 rounded-lg ${stat.bg} flex items-center justify-center mb-3`}>
                <Icon className={`w-5 h-5 ${stat.color}`} />
              </div>
              <p className="text-2xl font-bold text-[#1c1e21]">{stat.value}</p>
              <p className="text-sm text-[#65676b] mt-1">{stat.label}</p>
            </div>
          );
        })}
      </div>

      {/* Users Table */}
      <div className="bg-white rounded-lg border border-[#e9ecef] shadow-sm overflow-hidden">
        <div className="px-5 py-4 border-b border-[#e9ecef]">
          <h3 className="font-semibold text-[#1c1e21]">All Users</h3>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-[#f8f9fa] border-b border-[#e9ecef]">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-[#65676b]">User</th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-[#65676b]">Role</th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-[#65676b]">Status</th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-[#65676b]">Rate Limit</th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-[#65676b]">Created</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#e9ecef]">
              {users.map((u) => (
                <tr key={u.id} className="hover:bg-[#f8f9fa] transition-colors">
                  <td className="px-4 py-3">
                    <div>
                      <p className="text-sm font-medium text-[#1c1e21]">{u.name}</p>
                      <p className="text-xs text-[#65676b]">{u.email}</p>
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <span className={`text-xs font-medium px-2 py-1 rounded-full ${
                      u.role === 'admin' ? 'bg-purple-50 text-purple-600' : 'bg-[#f8f9fa] text-[#65676b]'
                    }`}>
                      {u.role}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    {u.active ? (
                      <span className="flex items-center gap-1 text-xs text-[#25D366]">
                        <CheckCircle className="w-3.5 h-3.5" />
                        Active
                      </span>
                    ) : (
                      <span className="flex items-center gap-1 text-xs text-[#ef4444]">
                        <XCircle className="w-3.5 h-3.5" />
                        Inactive
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    <span className="text-xs font-mono text-[#65676b]">{u.rate_limit} req/min</span>
                  </td>
                  <td className="px-4 py-3">
                    <p className="text-xs text-[#65676b]">{new Date(u.created_at).toLocaleDateString()}</p>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
