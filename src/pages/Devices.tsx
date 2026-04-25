import { useEffect, useState } from 'react';
import {
  Smartphone,
  Plus,
  Power,
  PowerOff,
  Trash2,
  QrCode,
  X,
  Loader2,
  Webhook,
} from 'lucide-react';
import { api } from '@/lib/api';
import type { Device } from '@/types';
import { StatusBadge } from '@/components/Layout';
import { useUIStore } from '@/stores/ui';

export default function Devices() {
  const { addToast } = useUIStore();
  const [devices, setDevices] = useState<Device[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAddModal, setShowAddModal] = useState(false);
  const [showQRModal, setShowQRModal] = useState<number | null>(null);
  const [showWebhookModal, setShowWebhookModal] = useState<number | null>(null);
  const [newDevice, setNewDevice] = useState({ name: '', webhook_url: '' });
  const [isCreating, setIsCreating] = useState(false);
  const [qrCodeUrl, setQrCodeUrl] = useState<string | null>(null);
  const [isQRLoading, setIsQRLoading] = useState(false);
  const [webhookForm, setWebhookForm] = useState({ url: '', events: ['message_received', 'message_sent'], secret: '' });

  useEffect(() => {
    loadDevices();
  }, []);

  useEffect(() => {
    if (!showQRModal) {
      setQrCodeUrl(null);
      setIsQRLoading(false);
      return;
    }

    let objectUrl: string | null = null;
    let cancelled = false;

    const loadQRCode = async () => {
      setIsQRLoading(true);
      setQrCodeUrl(null);
      const res = await api.getQRCodeBlobUrl(showQRModal);

      if (cancelled) {
        if (res.success && res.data) {
          URL.revokeObjectURL(res.data);
        }
        return;
      }

      if (res.success && res.data) {
        objectUrl = res.data;
        setQrCodeUrl(res.data);
      } else {
        addToast('error', res.error?.message || 'Failed to load QR code');
      }
      setIsQRLoading(false);
    };

    loadQRCode();

    return () => {
      cancelled = true;
      if (objectUrl) {
        URL.revokeObjectURL(objectUrl);
      }
    };
  }, [showQRModal, addToast]);

  const loadDevices = async () => {
    setLoading(true);
    const res = await api.getDevices();
    if (res.success && res.data) {
      setDevices(res.data);
    }
    setLoading(false);
  };

  const handleCreate = async () => {
    if (!newDevice.name.trim()) return;
    setIsCreating(true);
    const res = await api.createDevice(newDevice.name, newDevice.webhook_url);
    if (res.success) {
      addToast('success', 'Device created successfully');
      setShowAddModal(false);
      setNewDevice({ name: '', webhook_url: '' });
      loadDevices();
    } else {
      addToast('error', res.error?.message || 'Failed to create device');
    }
    setIsCreating(false);
  };

  const handleConnect = async (id: number) => {
    const res = await api.connectDevice(id);
    if (res.success) {
      addToast('success', 'Device connecting...');
      setShowQRModal(id);
    } else {
      addToast('error', res.error?.message || 'Failed to connect');
    }
  };

  const handleDisconnect = async (id: number) => {
    const res = await api.disconnectDevice(id);
    if (res.success) {
      addToast('success', 'Device disconnected');
      loadDevices();
    } else {
      addToast('error', res.error?.message || 'Failed to disconnect');
    }
  };

  const handleDelete = async (id: number) => {
    if (!confirm('Are you sure you want to delete this device? All associated messages will be lost.')) return;
    const res = await api.deleteDevice(id);
    if (res.success) {
      addToast('success', 'Device deleted');
      loadDevices();
    } else {
      addToast('error', res.error?.message || 'Failed to delete');
    }
  };

  const handleUpdateWebhook = async (id: number) => {
    const res = await api.updateWebhook(id, webhookForm.url, webhookForm.events, webhookForm.secret);
    if (res.success) {
      addToast('success', 'Webhook settings updated');
      setShowWebhookModal(null);
      loadDevices();
    } else {
      addToast('error', res.error?.message || 'Failed to update webhook');
    }
  };

  if (loading) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {Array.from({ length: 3 }).map((_, i) => (
          <div key={i} className="bg-white rounded-lg border border-[#e9ecef] p-5 animate-pulse">
            <div className="h-4 bg-[#e9ecef] rounded w-24 mb-3" />
            <div className="h-3 bg-[#e9ecef] rounded w-32 mb-4" />
            <div className="h-8 bg-[#e9ecef] rounded w-full" />
          </div>
        ))}
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-[#1c1e21]">WhatsApp Devices</h2>
          <p className="text-sm text-[#65676b]">Manage your connected WhatsApp numbers</p>
        </div>
        <button
          onClick={() => setShowAddModal(true)}
          className="flex items-center gap-2 px-4 py-2 bg-[#25D366] hover:bg-[#128C7E] text-white font-medium rounded-lg transition-colors text-sm"
        >
          <Plus className="w-4 h-4" />
          Add Device
        </button>
      </div>

      {/* Device Grid */}
      {devices.length === 0 ? (
        <div className="bg-white rounded-lg border border-[#e9ecef] p-12 text-center">
          <Smartphone className="w-12 h-12 text-[#e9ecef] mx-auto mb-4" />
          <h3 className="text-base font-medium text-[#1c1e21] mb-2">No devices connected</h3>
          <p className="text-sm text-[#65676b] mb-6 max-w-md mx-auto">
            Add a WhatsApp device by scanning a QR code with your WhatsApp mobile app.
          </p>
          <button
            onClick={() => setShowAddModal(true)}
            className="px-4 py-2 bg-[#25D366] hover:bg-[#128C7E] text-white font-medium rounded-lg transition-colors text-sm"
          >
            Add Your First Device
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {devices.map((device) => (
            <div
              key={device.id}
              className="bg-white rounded-lg border border-[#e9ecef] shadow-sm overflow-hidden"
            >
              {/* Status bar */}
              <div
                className={`h-1 ${
                  device.status === 'connected' ? 'bg-[#25D366]' : 'bg-[#e9ecef]'
                }`}
              />

              <div className="p-5">
                <div className="flex items-start justify-between mb-3">
                  <div>
                    <h3 className="font-semibold text-[#1c1e21]">{device.name}</h3>
                    <p className="text-xs font-mono text-[#65676b] mt-0.5">
                      {device.phone_number || 'Not paired'}
                    </p>
                  </div>
                  <StatusBadge status={device.status} />
                </div>

                <div className="flex items-center gap-2 text-xs text-[#65676b] mb-4">
                  <Smartphone className="w-3.5 h-3.5" />
                  <span>{device.push_name || 'Unknown'}</span>
                </div>

                <div className="flex items-center gap-2">
                  {device.status === 'disconnected' ? (
                    <button
                      onClick={() => handleConnect(device.id)}
                      className="flex items-center gap-1.5 px-3 py-1.5 bg-[#25D366] hover:bg-[#128C7E] text-white text-xs font-medium rounded-lg transition-colors"
                    >
                      <Power className="w-3.5 h-3.5" />
                      Connect
                    </button>
                  ) : (
                    <button
                      onClick={() => handleDisconnect(device.id)}
                      className="flex items-center gap-1.5 px-3 py-1.5 bg-[#e9ecef] hover:bg-[#d1d5db] text-[#65676b] text-xs font-medium rounded-lg transition-colors"
                    >
                      <PowerOff className="w-3.5 h-3.5" />
                      Disconnect
                    </button>
                  )}

                  {device.status === 'connecting' && (
                    <button
                      onClick={() => setShowQRModal(device.id)}
                      className="flex items-center gap-1.5 px-3 py-1.5 bg-blue-50 hover:bg-blue-100 text-blue-600 text-xs font-medium rounded-lg transition-colors"
                    >
                      <QrCode className="w-3.5 h-3.5" />
                      QR Code
                    </button>
                  )}

                  <button
                    onClick={() => {
                      setWebhookForm({
                        url: device.webhook_url || '',
                        events: device.webhook_events ? device.webhook_events.split(',') : ['message_received'],
                        secret: device.webhook_secret || '',
                      });
                      setShowWebhookModal(device.id);
                    }}
                    className="flex items-center gap-1.5 px-3 py-1.5 bg-purple-50 hover:bg-purple-100 text-purple-600 text-xs font-medium rounded-lg transition-colors"
                  >
                    <Webhook className="w-3.5 h-3.5" />
                    Webhook
                  </button>

                  <button
                    onClick={() => handleDelete(device.id)}
                    className="ml-auto p-1.5 text-[#ef4444] hover:bg-red-50 rounded-lg transition-colors"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Add Device Modal */}
      {showAddModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-white rounded-xl shadow-lg w-full max-w-md mx-4">
            <div className="flex items-center justify-between px-6 py-4 border-b border-[#e9ecef]">
              <h3 className="font-semibold text-[#1c1e21]">Add WhatsApp Device</h3>
              <button onClick={() => setShowAddModal(false)} className="p-1 hover:bg-[#f8f9fa] rounded-lg">
                <X className="w-5 h-5 text-[#65676b]" />
              </button>
            </div>
            <div className="p-6 space-y-4">
              <div>
                <label className="block text-xs font-medium uppercase tracking-wide text-[#65676b] mb-1.5">
                  Device Name
                </label>
                <input
                  type="text"
                  value={newDevice.name}
                  onChange={(e) => setNewDevice({ ...newDevice, name: e.target.value })}
                  className="w-full h-10 px-3 rounded-lg border border-[#e9ecef] text-sm focus:outline-none focus:ring-2 focus:ring-[#25D366]"
                  placeholder="e.g. Business Account"
                />
              </div>
              <div>
                <label className="block text-xs font-medium uppercase tracking-wide text-[#65676b] mb-1.5">
                  Webhook URL (optional)
                </label>
                <input
                  type="url"
                  value={newDevice.webhook_url}
                  onChange={(e) => setNewDevice({ ...newDevice, webhook_url: e.target.value })}
                  className="w-full h-10 px-3 rounded-lg border border-[#e9ecef] text-sm focus:outline-none focus:ring-2 focus:ring-[#25D366]"
                  placeholder="https://your-app.com/webhook"
                />
              </div>
            </div>
            <div className="flex items-center justify-end gap-2 px-6 py-4 border-t border-[#e9ecef]">
              <button
                onClick={() => setShowAddModal(false)}
                className="px-4 py-2 text-sm font-medium text-[#65676b] hover:bg-[#f8f9fa] rounded-lg transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleCreate}
                disabled={isCreating || !newDevice.name.trim()}
                className="px-4 py-2 bg-[#25D366] hover:bg-[#128C7E] text-white text-sm font-medium rounded-lg transition-colors flex items-center gap-2 disabled:opacity-50"
              >
                {isCreating && <Loader2 className="w-4 h-4 animate-spin" />}
                Create Device
              </button>
            </div>
          </div>
        </div>
      )}

      {/* QR Code Modal */}
      {showQRModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-white rounded-xl shadow-lg w-full max-w-sm mx-4">
            <div className="flex items-center justify-between px-6 py-4 border-b border-[#e9ecef]">
              <h3 className="font-semibold text-[#1c1e21]">Scan QR Code</h3>
              <button onClick={() => setShowQRModal(null)} className="p-1 hover:bg-[#f8f9fa] rounded-lg">
                <X className="w-5 h-5 text-[#65676b]" />
              </button>
            </div>
            <div className="p-6 text-center">
              <p className="text-sm text-[#65676b] mb-4">
                Open WhatsApp on your phone, go to Settings {'>'} Linked Devices {'>'} Link a Device
              </p>
              <div className="bg-white p-4 rounded-lg border border-[#e9ecef] inline-block">
                {isQRLoading ? (
                  <div className="w-48 h-48 flex items-center justify-center">
                    <Loader2 className="w-8 h-8 animate-spin text-[#25D366]" />
                  </div>
                ) : qrCodeUrl ? (
                  <img
                    src={qrCodeUrl}
                    alt="QR Code"
                    className="w-48 h-48"
                  />
                ) : (
                  <div className="w-48 h-48 flex items-center justify-center text-sm text-[#65676b]">
                    QR unavailable
                  </div>
                )}
              </div>
              <p className="text-xs text-[#8a8d91] mt-4">QR code refreshes automatically</p>
            </div>
          </div>
        </div>
      )}

      {/* Webhook Modal */}
      {showWebhookModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-white rounded-xl shadow-lg w-full max-w-lg mx-4">
            <div className="flex items-center justify-between px-6 py-4 border-b border-[#e9ecef]">
              <h3 className="font-semibold text-[#1c1e21]">Webhook Configuration</h3>
              <button onClick={() => setShowWebhookModal(null)} className="p-1 hover:bg-[#f8f9fa] rounded-lg">
                <X className="w-5 h-5 text-[#65676b]" />
              </button>
            </div>
            <div className="p-6 space-y-4">
              <div>
                <label className="block text-xs font-medium uppercase tracking-wide text-[#65676b] mb-1.5">
                  Webhook URL
                </label>
                <input
                  type="url"
                  value={webhookForm.url}
                  onChange={(e) => setWebhookForm({ ...webhookForm, url: e.target.value })}
                  className="w-full h-10 px-3 rounded-lg border border-[#e9ecef] text-sm focus:outline-none focus:ring-2 focus:ring-[#25D366]"
                  placeholder="https://your-app.com/webhook"
                />
              </div>
              <div>
                <label className="block text-xs font-medium uppercase tracking-wide text-[#65676b] mb-1.5">
                  Secret Token
                </label>
                <input
                  type="text"
                  value={webhookForm.secret}
                  onChange={(e) => setWebhookForm({ ...webhookForm, secret: e.target.value })}
                  className="w-full h-10 px-3 rounded-lg border border-[#e9ecef] text-sm focus:outline-none focus:ring-2 focus:ring-[#25D366]"
                  placeholder="For HMAC signature verification"
                />
              </div>
              <div>
                <label className="block text-xs font-medium uppercase tracking-wide text-[#65676b] mb-2">
                  Events
                </label>
                <div className="space-y-2">
                  {['message_received', 'message_sent', 'message_delivered', 'message_read', 'device_connected', 'device_disconnected'].map((event) => (
                    <label key={event} className="flex items-center gap-2 text-sm text-[#1c1e21]">
                      <input
                        type="checkbox"
                        checked={webhookForm.events.includes(event)}
                        onChange={(e) => {
                          if (e.target.checked) {
                            setWebhookForm({ ...webhookForm, events: [...webhookForm.events, event] });
                          } else {
                            setWebhookForm({ ...webhookForm, events: webhookForm.events.filter((ev) => ev !== event) });
                          }
                        }}
                        className="rounded border-[#e9ecef] text-[#25D366] focus:ring-[#25D366]"
                      />
                      {event.replace(/_/g, ' ').replace(/\b\w/g, (l) => l.toUpperCase())}
                    </label>
                  ))}
                </div>
              </div>
            </div>
            <div className="flex items-center justify-end gap-2 px-6 py-4 border-t border-[#e9ecef]">
              <button
                onClick={() => setShowWebhookModal(null)}
                className="px-4 py-2 text-sm font-medium text-[#65676b] hover:bg-[#f8f9fa] rounded-lg transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={() => handleUpdateWebhook(showWebhookModal)}
                className="px-4 py-2 bg-[#25D366] hover:bg-[#128C7E] text-white text-sm font-medium rounded-lg transition-colors"
              >
                Save Settings
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
