import { useEffect, useState } from 'react';
import {
  Key,
  Plus,
  Trash2,
  Copy,
  Check,
  Webhook,
  Clock,
  CheckCircle,
  XCircle,
  ArrowUpRight,
} from 'lucide-react';
import { api } from '@/lib/api';
import type { APIKey, WebhookLog } from '@/types';
import { useUIStore } from '@/stores/ui';

export default function APIKeys() {
  const { addToast } = useUIStore();
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [logs, setLogs] = useState<WebhookLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [newKeyName, setNewKeyName] = useState('');
  const [isCreating, setIsCreating] = useState(false);
  const [revealedKey, setRevealedKey] = useState<string | null>(null);
  const [copiedKey, setCopiedKey] = useState<string | null>(null);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    setLoading(true);
    const [keysRes, logsRes] = await Promise.all([
      api.getAPIKeys(),
      api.getWebhookLogs(10, 0),
    ]);
    if (keysRes.success && keysRes.data) {
      setKeys(keysRes.data);
    }
    if (logsRes.success && logsRes.data) {
      setLogs(logsRes.data);
    }
    setLoading(false);
  };

  const handleCreate = async () => {
    if (!newKeyName.trim()) return;
    setIsCreating(true);
    const res = await api.createAPIKey(newKeyName);
    if (res.success && res.data) {
      addToast('success', 'API key created');
      setRevealedKey(res.data.api_key);
      setShowCreateModal(false);
      setNewKeyName('');
      loadData();
    } else {
      addToast('error', res.error?.message || 'Failed to create API key');
    }
    setIsCreating(false);
  };

  const handleRevoke = async (id: number) => {
    if (!confirm('Are you sure? This will immediately revoke access.')) return;
    const res = await api.revokeAPIKey(id);
    if (res.success) {
      addToast('success', 'API key revoked');
      loadData();
    } else {
      addToast('error', res.error?.message || 'Failed to revoke');
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(text);
    setTimeout(() => setCopiedKey(null), 2000);
    addToast('success', 'Copied to clipboard');
  };

  return (
    <div className="space-y-6">
      {/* API Keys Section */}
      <div className="bg-white rounded-lg border border-[#e9ecef] shadow-sm">
        <div className="px-5 py-4 border-b border-[#e9ecef] flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Key className="w-5 h-5 text-[#25D366]" />
            <h3 className="font-semibold text-[#1c1e21]">API Keys</h3>
          </div>
          <button
            onClick={() => setShowCreateModal(true)}
            className="flex items-center gap-2 px-3 py-1.5 bg-[#25D366] hover:bg-[#128C7E] text-white text-sm font-medium rounded-lg transition-colors"
          >
            <Plus className="w-4 h-4" />
            Generate Key
          </button>
        </div>

        {loading ? (
          <div className="p-8 space-y-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="h-12 bg-[#e9ecef] rounded animate-pulse" />
            ))}
          </div>
        ) : keys.length === 0 ? (
          <div className="p-8 text-center">
            <Key className="w-10 h-10 text-[#e9ecef] mx-auto mb-3" />
            <p className="text-sm text-[#65676b]">No API keys yet. Generate one to use the Meta-compatible API.</p>
          </div>
        ) : (
          <div className="divide-y divide-[#e9ecef]">
            {keys.map((key) => (
              <div key={key.id} className="px-5 py-4 flex items-center justify-between hover:bg-[#f8f9fa] transition-colors">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <p className="text-sm font-medium text-[#1c1e21]">{key.name}</p>
                    {key.device_id && (
                      <span className="text-xs px-1.5 py-0.5 rounded bg-blue-50 text-blue-600 font-medium">
                        Device-specific
                      </span>
                    )}
                  </div>
                  <div className="flex items-center gap-2 mt-1">
                    <code className="text-xs font-mono text-[#65676b] bg-[#f8f9fa] px-2 py-0.5 rounded">
                      {key.key_prefix}****
                    </code>
                    {key.last_used_at && (
                      <span className="text-xs text-[#8a8d91]">
                        Last used: {new Date(key.last_used_at).toLocaleString()}
                      </span>
                    )}
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => copyToClipboard(key.key_prefix + ' (demo: use revealed key)')}
                    className="p-1.5 text-[#65676b] hover:bg-[#f8f9fa] rounded-lg transition-colors"
                    title="Copy"
                  >
                    {copiedKey?.includes(key.key_prefix) ? (
                      <Check className="w-4 h-4 text-[#25D366]" />
                    ) : (
                      <Copy className="w-4 h-4" />
                    )}
                  </button>
                  <button
                    onClick={() => handleRevoke(key.id)}
                    className="p-1.5 text-[#ef4444] hover:bg-red-50 rounded-lg transition-colors"
                    title="Revoke"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Revealed Key Banner */}
      {revealedKey && (
        <div className="bg-[#dcf8c6] rounded-lg border border-[#25D366]/20 p-4">
          <div className="flex items-start justify-between">
            <div>
              <p className="text-sm font-medium text-[#128C7E] mb-1">New API Key Generated</p>
              <p className="text-xs text-[#128C7E]/80 mb-2">Copy this now - you won&apos;t be able to see it again!</p>
              <code className="text-sm font-mono bg-white px-3 py-1.5 rounded border border-[#25D366]/20 block w-fit">
                {revealedKey}
              </code>
            </div>
            <button
              onClick={() => setRevealedKey(null)}
              className="p-1 hover:bg-[#25D366]/10 rounded-lg text-[#128C7E]"
            >
              <XCircle className="w-5 h-5" />
            </button>
          </div>
        </div>
      )}

      {/* Webhook Logs Section */}
      <div className="bg-white rounded-lg border border-[#e9ecef] shadow-sm">
        <div className="px-5 py-4 border-b border-[#e9ecef] flex items-center gap-2">
          <Webhook className="w-5 h-5 text-orange-500" />
          <h3 className="font-semibold text-[#1c1e21]">Recent Webhook Deliveries</h3>
        </div>

        {logs.length === 0 ? (
          <div className="p-8 text-center">
            <Webhook className="w-10 h-10 text-[#e9ecef] mx-auto mb-3" />
            <p className="text-sm text-[#65676b]">No webhook deliveries yet. Configure webhooks on your devices.</p>
          </div>
        ) : (
          <div className="divide-y divide-[#e9ecef]">
            {logs.map((log) => (
              <div key={log.id} className="px-5 py-3 flex items-center justify-between hover:bg-[#f8f9fa] transition-colors">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="text-xs font-medium px-2 py-0.5 rounded-full bg-[#f8f9fa] text-[#65676b]">
                      {log.event_type}
                    </span>
                    {log.success ? (
                      <CheckCircle className="w-3.5 h-3.5 text-[#25D366]" />
                    ) : (
                      <XCircle className="w-3.5 h-3.5 text-[#ef4444]" />
                    )}
                  </div>
                  <p className="text-xs text-[#65676b] mt-1 truncate">{log.url}</p>
                </div>
                <div className="flex items-center gap-3 text-xs text-[#65676b]">
                  {log.status_code && (
                    <span className={`font-mono ${log.status_code >= 200 && log.status_code < 300 ? 'text-[#25D366]' : 'text-[#ef4444]'}`}>
                      {log.status_code}
                    </span>
                  )}
                  {log.response_time_ms && (
                    <span className="flex items-center gap-1">
                      <Clock className="w-3 h-3" />
                      {log.response_time_ms}ms
                    </span>
                  )}
                  <span>{new Date(log.created_at).toLocaleTimeString()}</span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Create Key Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-white rounded-xl shadow-lg w-full max-w-md mx-4">
            <div className="flex items-center justify-between px-6 py-4 border-b border-[#e9ecef]">
              <h3 className="font-semibold text-[#1c1e21]">Generate API Key</h3>
              <button
                onClick={() => setShowCreateModal(false)}
                className="p-1 hover:bg-[#f8f9fa] rounded-lg"
              >
                <XCircle className="w-5 h-5 text-[#65676b]" />
              </button>
            </div>
            <div className="p-6">
              <label className="block text-xs font-medium uppercase tracking-wide text-[#65676b] mb-1.5">
                Key Name
              </label>
              <input
                type="text"
                value={newKeyName}
                onChange={(e) => setNewKeyName(e.target.value)}
                className="w-full h-10 px-3 rounded-lg border border-[#e9ecef] text-sm focus:outline-none focus:ring-2 focus:ring-[#25D366]"
                placeholder="e.g. Production API"
              />
            </div>
            <div className="flex items-center justify-end gap-2 px-6 py-4 border-t border-[#e9ecef]">
              <button
                onClick={() => setShowCreateModal(false)}
                className="px-4 py-2 text-sm font-medium text-[#65676b] hover:bg-[#f8f9fa] rounded-lg transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleCreate}
                disabled={isCreating || !newKeyName.trim()}
                className="px-4 py-2 bg-[#25D366] hover:bg-[#128C7E] text-white text-sm font-medium rounded-lg transition-colors flex items-center gap-2 disabled:opacity-50"
              >
                {isCreating && <ArrowUpRight className="w-4 h-4 animate-spin" />}
                Generate
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
