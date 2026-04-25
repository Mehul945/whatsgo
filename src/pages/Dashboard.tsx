import { useEffect, useState } from 'react';
import {
  Smartphone,
  MessageSquare,
  Calendar,
  Webhook,
  ArrowUpRight,
  ArrowDownRight,
} from 'lucide-react';
import { api } from '@/lib/api';
import type { DashboardStats, Device } from '@/types';
import { StatusBadge } from '@/components/Layout';
import { useNavigate } from 'react-router';

const statConfig = [
  { key: 'active_devices' as keyof DashboardStats, label: 'Active Devices', icon: Smartphone, color: 'text-[#25D366]', bg: 'bg-[#dcf8c6]' },
  { key: 'messages_today' as keyof DashboardStats, label: 'Messages Today', icon: MessageSquare, color: 'text-blue-500', bg: 'bg-blue-50' },
  { key: 'messages_this_month' as keyof DashboardStats, label: 'This Month', icon: Calendar, color: 'text-purple-500', bg: 'bg-purple-50' },
  { key: 'webhook_deliveries' as keyof DashboardStats, label: 'Webhook Deliveries', icon: Webhook, color: 'text-orange-500', bg: 'bg-orange-50' },
];

export default function Dashboard() {
  const navigate = useNavigate();
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [devices, setDevices] = useState<Device[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    setLoading(true);
    const [statsRes, devicesRes] = await Promise.all([
      api.getStats(),
      api.getDevices(),
    ]);

    if (statsRes.success && statsRes.data) {
      setStats(statsRes.data);
    }
    if (devicesRes.success && devicesRes.data) {
      setDevices(devicesRes.data.slice(0, 5));
    }
    setLoading(false);
  };

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="bg-white rounded-lg p-5 border border-[#e9ecef] animate-pulse">
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
      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {statConfig.map((stat) => {
          const Icon = stat.icon;
          const value = stats?.[stat.key] ?? 0;
          return (
            <div
              key={stat.key}
              className="bg-white rounded-lg p-5 border border-[#e9ecef] shadow-sm hover:shadow-md transition-shadow"
            >
              <div className="flex items-center justify-between mb-4">
                <div className={`w-10 h-10 rounded-lg ${stat.bg} flex items-center justify-center`}>
                  <Icon className={`w-5 h-5 ${stat.color}`} />
                </div>
                <span className="text-xs font-medium text-[#25D366] flex items-center gap-0.5">
                  <ArrowUpRight className="w-3 h-3" />
                  Live
                </span>
              </div>
              <p className="text-2xl font-bold text-[#1c1e21]">{value.toLocaleString()}</p>
              <p className="text-sm text-[#65676b] mt-1">{stat.label}</p>
            </div>
          );
        })}
      </div>

      {/* Device Status */}
      <div className="bg-white rounded-lg border border-[#e9ecef] shadow-sm">
        <div className="px-5 py-4 border-b border-[#e9ecef] flex items-center justify-between">
          <h3 className="font-semibold text-[#1c1e21]">Device Status</h3>
          <button
            onClick={() => navigate('/devices')}
            className="text-sm text-[#128C7E] hover:text-[#25D366] font-medium flex items-center gap-1"
          >
            View All
            <ArrowDownRight className="w-3.5 h-3.5" />
          </button>
        </div>

        {devices.length === 0 ? (
          <div className="p-8 text-center">
            <Smartphone className="w-10 h-10 text-[#e9ecef] mx-auto mb-3" />
            <h4 className="text-sm font-medium text-[#1c1e21] mb-1">No devices yet</h4>
            <p className="text-sm text-[#65676b] mb-4">Add your first WhatsApp device to get started</p>
            <button
              onClick={() => navigate('/devices')}
              className="px-4 py-2 bg-[#25D366] hover:bg-[#128C7E] text-white text-sm font-medium rounded-lg transition-colors"
            >
              Add Device
            </button>
          </div>
        ) : (
          <div className="divide-y divide-[#e9ecef]">
            {devices.map((device) => (
              <div key={device.id} className="px-5 py-3 flex items-center justify-between hover:bg-[#f8f9fa] transition-colors">
                <div className="flex items-center gap-3">
                  <div
                    className={`w-2 h-2 rounded-full ${
                      device.status === 'connected' ? 'bg-[#25D366]' : 'bg-[#e9ecef]'
                    }`}
                  />
                  <div>
                    <p className="text-sm font-medium text-[#1c1e21]">{device.name}</p>
                    <p className="text-xs text-[#65676b] font-mono">{device.phone_number || 'Not paired'}</p>
                  </div>
                </div>
                <StatusBadge status={device.status} />
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
