import { useEffect, useState } from 'react';
import {
  User,
  Mail,
  Lock,
  Save,
  Loader2,
  AlertTriangle,
  Trash2,
} from 'lucide-react';
import { api } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';
import { useUIStore } from '@/stores/ui';
import type { User as UserType } from '@/types';

export default function Settings() {
  const { logout } = useAuthStore();
  const { addToast } = useUIStore();
  const [user, setUser] = useState<UserType | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [form, setForm] = useState({ name: '', email: '', password: '', confirmPassword: '' });
  const [showDeleteModal, setShowDeleteModal] = useState(false);

  useEffect(() => {
    loadProfile();
  }, []);

  const loadProfile = async () => {
    const res = await api.getProfile();
    if (res.success && res.data) {
      setUser(res.data);
      setForm((prev) => ({ ...prev, name: res.data!.name, email: res.data!.email }));
    }
  };

  const handleUpdate = async () => {
    if (form.password && form.password !== form.confirmPassword) {
      addToast('error', 'Passwords do not match');
      return;
    }

    setIsLoading(true);
    const updateData: { name?: string; email?: string; password?: string } = {};
    if (form.name && form.name !== user?.name) updateData.name = form.name;
    if (form.email && form.email !== user?.email) updateData.email = form.email;
    if (form.password) updateData.password = form.password;

    if (Object.keys(updateData).length === 0) {
      addToast('info', 'No changes to save');
      setIsLoading(false);
      return;
    }

    const res = await api.updateProfile(updateData);
    if (res.success) {
      addToast('success', 'Profile updated successfully');
      loadProfile();
      setForm((prev) => ({ ...prev, password: '', confirmPassword: '' }));
    } else {
      addToast('error', res.error?.message || 'Failed to update profile');
    }
    setIsLoading(false);
  };

  return (
    <div className="max-w-2xl space-y-6">
      {/* Profile Section */}
      <div className="bg-white rounded-lg border border-[#e9ecef] shadow-sm">
        <div className="px-5 py-4 border-b border-[#e9ecef]">
          <h3 className="font-semibold text-[#1c1e21]">Profile Settings</h3>
        </div>
        <div className="p-5 space-y-4">
          <div>
            <label className="block text-xs font-medium uppercase tracking-wide text-[#65676b] mb-1.5">
              Full Name
            </label>
            <div className="relative">
              <User className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[#8a8d91]" />
              <input
                type="text"
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                className="w-full h-10 pl-9 pr-3 rounded-lg border border-[#e9ecef] text-sm focus:outline-none focus:ring-2 focus:ring-[#25D366]"
              />
            </div>
          </div>

          <div>
            <label className="block text-xs font-medium uppercase tracking-wide text-[#65676b] mb-1.5">
              Email Address
            </label>
            <div className="relative">
              <Mail className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[#8a8d91]" />
              <input
                type="email"
                value={form.email}
                onChange={(e) => setForm({ ...form, email: e.target.value })}
                className="w-full h-10 pl-9 pr-3 rounded-lg border border-[#e9ecef] text-sm focus:outline-none focus:ring-2 focus:ring-[#25D366]"
              />
            </div>
          </div>

          <div className="border-t border-[#e9ecef] pt-4">
            <p className="text-sm font-medium text-[#1c1e21] mb-3">Change Password</p>
            <div className="space-y-3">
              <div className="relative">
                <Lock className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[#8a8d91]" />
                <input
                  type="password"
                  value={form.password}
                  onChange={(e) => setForm({ ...form, password: e.target.value })}
                  className="w-full h-10 pl-9 pr-3 rounded-lg border border-[#e9ecef] text-sm focus:outline-none focus:ring-2 focus:ring-[#25D366]"
                  placeholder="New password"
                />
              </div>
              <div className="relative">
                <Lock className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[#8a8d91]" />
                <input
                  type="password"
                  value={form.confirmPassword}
                  onChange={(e) => setForm({ ...form, confirmPassword: e.target.value })}
                  className="w-full h-10 pl-9 pr-3 rounded-lg border border-[#e9ecef] text-sm focus:outline-none focus:ring-2 focus:ring-[#25D366]"
                  placeholder="Confirm new password"
                />
              </div>
            </div>
          </div>

          <button
            onClick={handleUpdate}
            disabled={isLoading}
            className="flex items-center gap-2 px-4 py-2 bg-[#25D366] hover:bg-[#128C7E] text-white text-sm font-medium rounded-lg transition-colors disabled:opacity-50"
          >
            {isLoading && <Loader2 className="w-4 h-4 animate-spin" />}
            <Save className="w-4 h-4" />
            Save Changes
          </button>
        </div>
      </div>

      {/* Danger Zone */}
      <div className="bg-white rounded-lg border border-red-200 shadow-sm">
        <div className="px-5 py-4 border-b border-red-100">
          <h3 className="font-semibold text-red-700 flex items-center gap-2">
            <AlertTriangle className="w-4 h-4" />
            Danger Zone
          </h3>
        </div>
        <div className="p-5">
          <p className="text-sm text-[#65676b] mb-4">
            Deleting your account will permanently remove all your devices, messages, and API keys. This action cannot be undone.
          </p>
          <button
            onClick={() => setShowDeleteModal(true)}
            className="flex items-center gap-2 px-4 py-2 bg-red-50 hover:bg-red-100 text-red-600 text-sm font-medium rounded-lg transition-colors border border-red-200"
          >
            <Trash2 className="w-4 h-4" />
            Delete Account
          </button>
        </div>
      </div>

      {/* Delete Confirmation Modal */}
      {showDeleteModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-white rounded-xl shadow-lg w-full max-w-sm mx-4 p-6">
            <div className="flex items-center gap-3 mb-4">
              <div className="w-10 h-10 rounded-full bg-red-50 flex items-center justify-center">
                <AlertTriangle className="w-5 h-5 text-red-600" />
              </div>
              <div>
                <h3 className="font-semibold text-[#1c1e21]">Delete Account?</h3>
                <p className="text-sm text-[#65676b]">This action is permanent and cannot be undone.</p>
              </div>
            </div>
            <div className="flex items-center justify-end gap-2">
              <button
                onClick={() => setShowDeleteModal(false)}
                className="px-4 py-2 text-sm font-medium text-[#65676b] hover:bg-[#f8f9fa] rounded-lg transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={() => {
                  logout();
                  addToast('success', 'Account deleted');
                }}
                className="px-4 py-2 bg-red-600 hover:bg-red-700 text-white text-sm font-medium rounded-lg transition-colors"
              >
                Delete Account
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
