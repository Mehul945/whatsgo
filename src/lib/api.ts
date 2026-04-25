import { API_BASE_URL } from '@/types';
import type { ApiResponse, LoginResponse, User, Device, Message, APIKey, WebhookLog, DashboardStats, SendMessageRequest } from '@/types';

class ApiClient {
  private token: string | null = null;

  setToken(token: string | null) {
    this.token = token;
  }

  async request<T>(endpoint: string, options: RequestInit = {}): Promise<ApiResponse<T>> {
    return this.makeRequest<T>(endpoint, options);
  }

  private async makeRequest<T>(endpoint: string, options: RequestInit = {}): Promise<ApiResponse<T>> {
    const url = `${API_BASE_URL}${endpoint}`;
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...((options.headers as Record<string, string>) || {}),
    };

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    try {
      const response = await fetch(url, {
        ...options,
        headers,
      });

      const data = await response.json();
      return data;
    } catch (error) {
      return {
        success: false,
        error: {
          code: 'NETWORK_ERROR',
          message: error instanceof Error ? error.message : 'Network error',
          status: 0,
        },
      };
    }
  }

  // Auth
  async login(email: string, password: string): Promise<ApiResponse<LoginResponse>> {
    return this.makeRequest('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    });
  }

  async register(email: string, password: string, name: string): Promise<ApiResponse<LoginResponse>> {
    return this.makeRequest('/api/auth/register', {
      method: 'POST',
      body: JSON.stringify({ email, password, name }),
    });
  }

  async refreshToken(refreshToken: string): Promise<ApiResponse<{ access_token: string; refresh_token: string }>> {
    return this.makeRequest('/api/auth/refresh', {
      method: 'POST',
      body: JSON.stringify({ refresh_token: refreshToken }),
    });
  }

  // Dashboard
  async getStats(): Promise<ApiResponse<DashboardStats>> {
    return this.makeRequest('/api/dashboard/stats');
  }

  // Devices
  async getDevices(): Promise<ApiResponse<Device[]>> {
    return this.makeRequest('/api/devices');
  }

  async createDevice(name: string, webhookUrl?: string): Promise<ApiResponse<Device>> {
    return this.makeRequest('/api/devices', {
      method: 'POST',
      body: JSON.stringify({ name, webhook_url: webhookUrl }),
    });
  }

  async connectDevice(id: number): Promise<ApiResponse<{ status: string }>> {
    return this.makeRequest(`/api/devices/${id}/connect`, {
      method: 'POST',
    });
  }

  async disconnectDevice(id: number): Promise<ApiResponse<{ status: string }>> {
    return this.makeRequest(`/api/devices/${id}/disconnect`, {
      method: 'POST',
    });
  }

  async deleteDevice(id: number): Promise<ApiResponse<{ message: string }>> {
    return this.makeRequest(`/api/devices/${id}`, {
      method: 'DELETE',
    });
  }

  async updateWebhook(id: number, webhookUrl: string, events: string[], secret: string): Promise<ApiResponse<{ message: string }>> {
    return this.makeRequest(`/api/devices/${id}/webhook`, {
      method: 'PUT',
      body: JSON.stringify({ webhook_url: webhookUrl, webhook_events: events, webhook_secret: secret }),
    });
  }

  getQRCodeUrl(id: number): string {
    return `${API_BASE_URL}/api/devices/${id}/qr`;
  }

  async getQRCodeBlobUrl(id: number): Promise<ApiResponse<string>> {
    const response = await this.makeBlobRequest(`/api/devices/${id}/qr`);
    if (!response.success || !response.data) {
      return {
        success: false,
        error: response.error,
      };
    }

    return {
      success: true,
      data: URL.createObjectURL(response.data),
    };
  }

  private async makeBlobRequest(endpoint: string, options: RequestInit = {}): Promise<ApiResponse<Blob>> {
    const url = `${API_BASE_URL}${endpoint}`;
    const headers: Record<string, string> = {
      ...((options.headers as Record<string, string>) || {}),
    };

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    try {
      const response = await fetch(url, {
        ...options,
        headers,
      });

      if (!response.ok) {
        let message = response.statusText;
        try {
          const errorData = await response.json();
          message = errorData.error?.message || message;
        } catch {
          // Keep the HTTP status text when the response is not JSON.
        }

        return {
          success: false,
          error: {
            code: 'HTTP_ERROR',
            message,
            status: response.status,
          },
        };
      }

      return {
        success: true,
        data: await response.blob(),
      };
    } catch (error) {
      return {
        success: false,
        error: {
          code: 'NETWORK_ERROR',
          message: error instanceof Error ? error.message : 'Network error',
          status: 0,
        },
      };
    }
  }

  // Messages
  async getMessages(params?: { limit?: number; offset?: number; direction?: string; device_id?: string; status?: string }): Promise<ApiResponse<Message[]>> {
    const searchParams = new URLSearchParams();
    if (params?.limit) searchParams.set('limit', String(params.limit));
    if (params?.offset) searchParams.set('offset', String(params.offset));
    if (params?.direction) searchParams.set('direction', params.direction);
    if (params?.device_id) searchParams.set('device_id', params.device_id);
    if (params?.status) searchParams.set('status', params.status);
    return this.makeRequest(`/api/messages?${searchParams.toString()}`);
  }

  async sendMessage(request: SendMessageRequest): Promise<ApiResponse<{ id: number; status: string; message_id: string }>> {
    return this.makeRequest('/api/messages/send', {
      method: 'POST',
      body: JSON.stringify(request),
    });
  }

  // API Keys
  async getAPIKeys(): Promise<ApiResponse<APIKey[]>> {
    return this.makeRequest('/api/apikeys');
  }

  async createAPIKey(name: string, deviceId?: number): Promise<ApiResponse<{ name: string; api_key: string; key_prefix: string; created_at: string }>> {
    return this.makeRequest('/api/apikeys', {
      method: 'POST',
      body: JSON.stringify({ name, device_id: deviceId }),
    });
  }

  async revokeAPIKey(id: number): Promise<ApiResponse<{ message: string }>> {
    return this.makeRequest(`/api/apikeys/${id}`, {
      method: 'DELETE',
    });
  }

  // Webhooks
  async getWebhookLogs(limit?: number, offset?: number): Promise<ApiResponse<WebhookLog[]>> {
    const searchParams = new URLSearchParams();
    if (limit) searchParams.set('limit', String(limit));
    if (offset) searchParams.set('offset', String(offset));
    return this.makeRequest(`/api/webhooks/logs?${searchParams.toString()}`);
  }

  // User
  async getProfile(): Promise<ApiResponse<User>> {
    return this.makeRequest('/api/user/profile');
  }

  async updateProfile(data: { name?: string; email?: string; password?: string }): Promise<ApiResponse<{ message: string }>> {
    return this.makeRequest('/api/user/profile', {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }
}

export const api = new ApiClient();
