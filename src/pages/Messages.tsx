import { useEffect, useRef, useState } from 'react';
import {
  Send,
  ArrowUp,
  ArrowDown,
  Check,
  CheckCheck,
  X,
  Loader2,
  MessageSquare,
} from 'lucide-react';
import { api } from '@/lib/api';
import type { Message, Device } from '@/types';
import { useUIStore } from '@/stores/ui';

const tabs = ['All', 'Sent', 'Received', 'Failed'];
const tabDirection: Record<string, string> = {
  All: '',
  Sent: 'outbound',
  Received: 'inbound',
  Failed: '',
};

const statusIcons: Record<string, React.ReactNode> = {
  sent: <Check className="w-3.5 h-3.5 text-[#8a8d91]" />,
  delivered: <CheckCheck className="w-3.5 h-3.5 text-[#8a8d91]" />,
  read: <CheckCheck className="w-3.5 h-3.5 text-blue-500" />,
  failed: <X className="w-3.5 h-3.5 text-[#ef4444]" />,
  pending: <Loader2 className="w-3.5 h-3.5 text-[#f59e0b] animate-spin" />,
};

export default function Messages() {
  const { addToast } = useUIStore();
  const [messages, setMessages] = useState<Message[]>([]);
  const [devices, setDevices] = useState<Device[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState('All');
  const [showSendModal, setShowSendModal] = useState(false);
  const [sendForm, setSendForm] = useState({
    device_id: 0,
    recipient: '',
    type: 'text',
    content: '',
  });
  const [isSending, setIsSending] = useState(false);
  const [page, setPage] = useState(0);
  const [total, setTotal] = useState(0);
  const messagesRequestId = useRef(0);
  const limit = 20;

  useEffect(() => {
    loadDevices();
  }, []);

  useEffect(() => {
    loadMessages();
  }, [activeTab, page]);

  useEffect(() => {
    const interval = window.setInterval(() => {
      loadMessages(false);
    }, 5000);

    return () => window.clearInterval(interval);
  }, [activeTab, page]);

  const loadDevices = async () => {
    const res = await api.getDevices();
    if (res.success && res.data) {
      setDevices(res.data);
      if (res.data.length > 0 && sendForm.device_id === 0) {
        setSendForm((prev) => ({ ...prev, device_id: res.data![0].id }));
      }
    }
  };

  const loadMessages = async (showLoading = true) => {
    const requestId = messagesRequestId.current + 1;
    messagesRequestId.current = requestId;
    if (showLoading) setLoading(true);
    const direction = tabDirection[activeTab];
    const status = activeTab === 'Failed' ? 'failed' : undefined;
    const res = await api.getMessages({ limit, offset: page * limit, direction, status });
    if (requestId !== messagesRequestId.current) return;
    if (res.success && res.data) {
      setMessages(res.data);
      setTotal(res.meta?.total || 0);
    }
    if (showLoading) setLoading(false);
  };

  const handleSend = async () => {
    if (!sendForm.recipient || !sendForm.content) return;
    setIsSending(true);
    const res = await api.sendMessage(sendForm);
    if (res.success) {
      addToast('success', 'Message sent successfully');
      setShowSendModal(false);
      setSendForm({ device_id: devices[0]?.id || 0, recipient: '', type: 'text', content: '' });
      loadMessages();
    } else {
      addToast('error', res.error?.message || 'Failed to send message');
    }
    setIsSending(false);
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-[#1c1e21]">Messages</h2>
          <p className="text-sm text-[#65676b]">View and manage your WhatsApp messages</p>
        </div>
        <button
          onClick={() => setShowSendModal(true)}
          className="flex items-center gap-2 px-4 py-2 bg-[#25D366] hover:bg-[#128C7E] text-white font-medium rounded-lg transition-colors text-sm"
        >
          <Send className="w-4 h-4" />
          Send Message
        </button>
      </div>

      {/* Tabs */}
      <div className="flex items-center gap-1 bg-white rounded-lg border border-[#e9ecef] p-1 w-fit">
        {tabs.map((tab) => (
          <button
            key={tab}
            onClick={() => { setActiveTab(tab); setPage(0); }}
            className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
              activeTab === tab
                ? 'bg-[#dcf8c6] text-[#128C7E]'
                : 'text-[#65676b] hover:bg-[#f8f9fa]'
            }`}
          >
            {tab}
          </button>
        ))}
      </div>

      {/* Message List */}
      <div className="bg-white rounded-lg border border-[#e9ecef] shadow-sm overflow-hidden">
        {loading ? (
          <div className="p-8 space-y-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="flex items-center gap-3 animate-pulse">
                <div className="w-8 h-8 rounded-full bg-[#e9ecef]" />
                <div className="flex-1 space-y-2">
                  <div className="h-3 bg-[#e9ecef] rounded w-32" />
                  <div className="h-3 bg-[#e9ecef] rounded w-48" />
                </div>
              </div>
            ))}
          </div>
        ) : messages.length === 0 ? (
          <div className="p-12 text-center">
            <MessageSquare className="w-10 h-10 text-[#e9ecef] mx-auto mb-3" />
            <h4 className="text-sm font-medium text-[#1c1e21] mb-1">No messages yet</h4>
            <p className="text-sm text-[#65676b]">Send your first message or wait for incoming messages</p>
          </div>
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className="bg-[#f8f9fa] border-b border-[#e9ecef]">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-[#65676b]">Direction</th>
                    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-[#65676b]">Contact</th>
                    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-[#65676b]">Type</th>
                    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-[#65676b]">Content</th>
                    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-[#65676b]">Status</th>
                    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-[#65676b]">Time</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-[#e9ecef]">
                  {messages.map((msg) => (
                    <tr key={msg.id} className="hover:bg-[#f8f9fa] transition-colors">
                      <td className="px-4 py-3">
                        {msg.direction === 'outbound' ? (
                          <ArrowUp className="w-4 h-4 text-blue-500" />
                        ) : (
                          <ArrowDown className="w-4 h-4 text-[#25D366]" />
                        )}
                      </td>
                      <td className="px-4 py-3">
                        <p className="text-sm font-medium text-[#1c1e21] font-mono">{msg.recipient}</p>
                      </td>
                      <td className="px-4 py-3">
                        <span className="text-xs font-medium px-2 py-1 rounded-full bg-[#f8f9fa] text-[#65676b] capitalize">
                          {msg.message_type}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <p className="text-sm text-[#1c1e21] truncate max-w-xs">{msg.content || '-'}</p>
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-1.5">
                          {statusIcons[msg.status] || <Check className="w-3.5 h-3.5" />}
                          <span className="text-xs text-[#65676b] capitalize">{msg.status}</span>
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        <p className="text-xs text-[#65676b]">
                          {new Date(msg.created_at).toLocaleString()}
                        </p>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Pagination */}
            {total > limit && (
              <div className="flex items-center justify-between px-4 py-3 border-t border-[#e9ecef]">
                <p className="text-xs text-[#65676b]">
                  Showing {page * limit + 1}-{Math.min((page + 1) * limit, total)} of {total}
                </p>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => setPage(Math.max(0, page - 1))}
                    disabled={page === 0}
                    className="px-3 py-1.5 text-sm font-medium rounded-lg border border-[#e9ecef] hover:bg-[#f8f9fa] disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    Previous
                  </button>
                  <button
                    onClick={() => setPage(page + 1)}
                    disabled={(page + 1) * limit >= total}
                    className="px-3 py-1.5 text-sm font-medium rounded-lg border border-[#e9ecef] hover:bg-[#f8f9fa] disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    Next
                  </button>
                </div>
              </div>
            )}
          </>
        )}
      </div>

      {/* Send Message Modal */}
      {showSendModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-white rounded-xl shadow-lg w-full max-w-md mx-4">
            <div className="flex items-center justify-between px-6 py-4 border-b border-[#e9ecef]">
              <h3 className="font-semibold text-[#1c1e21]">Send WhatsApp Message</h3>
              <button
                onClick={() => setShowSendModal(false)}
                className="p-1 hover:bg-[#f8f9fa] rounded-lg"
              >
                <X className="w-5 h-5 text-[#65676b]" />
              </button>
            </div>
            <div className="p-6 space-y-4">
              <div>
                <label className="block text-xs font-medium uppercase tracking-wide text-[#65676b] mb-1.5">
                  From Device
                </label>
                <select
                  value={sendForm.device_id}
                  onChange={(e) => setSendForm({ ...sendForm, device_id: Number(e.target.value) })}
                  className="w-full h-10 px-3 rounded-lg border border-[#e9ecef] text-sm focus:outline-none focus:ring-2 focus:ring-[#25D366] bg-white"
                >
                  {devices.map((d) => (
                    <option key={d.id} value={d.id}>
                      {d.name} {d.phone_number ? `(${d.phone_number})` : ''}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-xs font-medium uppercase tracking-wide text-[#65676b] mb-1.5">
                  Recipient
                </label>
                <input
                  type="text"
                  value={sendForm.recipient}
                  onChange={(e) => setSendForm({ ...sendForm, recipient: e.target.value })}
                  className="w-full h-10 px-3 rounded-lg border border-[#e9ecef] text-sm focus:outline-none focus:ring-2 focus:ring-[#25D366]"
                  placeholder="+1234567890"
                />
                <p className="text-xs text-[#8a8d91] mt-1">International format with country code</p>
              </div>
              <div>
                <label className="block text-xs font-medium uppercase tracking-wide text-[#65676b] mb-1.5">
                  Message Type
                </label>
                <select
                  value={sendForm.type}
                  onChange={(e) => setSendForm({ ...sendForm, type: e.target.value })}
                  className="w-full h-10 px-3 rounded-lg border border-[#e9ecef] text-sm focus:outline-none focus:ring-2 focus:ring-[#25D366] bg-white"
                >
                  <option value="text">Text</option>
                  <option value="image">Image</option>
                  <option value="template">Template</option>
                </select>
              </div>
              <div>
                <label className="block text-xs font-medium uppercase tracking-wide text-[#65676b] mb-1.5">
                  Message
                </label>
                <textarea
                  value={sendForm.content}
                  onChange={(e) => setSendForm({ ...sendForm, content: e.target.value })}
                  className="w-full h-24 px-3 py-2 rounded-lg border border-[#e9ecef] text-sm focus:outline-none focus:ring-2 focus:ring-[#25D366] resize-none"
                  placeholder="Type your message..."
                />
              </div>
            </div>
            <div className="flex items-center justify-end gap-2 px-6 py-4 border-t border-[#e9ecef]">
              <button
                onClick={() => setShowSendModal(false)}
                className="px-4 py-2 text-sm font-medium text-[#65676b] hover:bg-[#f8f9fa] rounded-lg transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleSend}
                disabled={isSending || !sendForm.recipient || !sendForm.content}
                className="px-4 py-2 bg-[#25D366] hover:bg-[#128C7E] text-white text-sm font-medium rounded-lg transition-colors flex items-center gap-2 disabled:opacity-50"
              >
                {isSending && <Loader2 className="w-4 h-4 animate-spin" />}
                Send Message
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
