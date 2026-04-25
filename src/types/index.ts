const isProd = import.meta.env.PROD;
export const API_BASE_URL = import.meta.env.VITE_API_URL || (isProd ? '' : 'http://localhost:8080');

export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
    status: number;
  };
  meta?: {
    page: number;
    limit: number;
    total: number;
    offset: number;
  };
}

export interface User {
  id: number;
  email: string;
  name: string;
  role: string;
  active: boolean;
  rate_limit: number;
  created_at: string;
  updated_at: string;
}

export interface Device {
  id: number;
  user_id: number;
  name: string;
  phone_number: string;
  jid: string;
  push_name: string;
  status: 'connected' | 'disconnected' | 'connecting';
  webhook_url: string;
  webhook_events: string;
  webhook_secret: string;
  created_at: string;
  updated_at: string;
}

export interface Message {
  id: number;
  device_id: number;
  user_id: number;
  direction: 'inbound' | 'outbound';
  recipient: string;
  message_type: string;
  content: string;
  media_url: string;
  media_mime_type: string;
  status: string;
  whatsapp_message_id: string;
  error_message: string;
  created_at: string;
  updated_at: string;
}

export interface APIKey {
  id: number;
  user_id: number;
  device_id?: number;
  name: string;
  key_prefix: string;
  last_used_at?: string;
  created_at: string;
  revoked_at?: string;
}

export interface WebhookLog {
  id: number;
  device_id: number;
  event_type: string;
  payload: string;
  url: string;
  status_code?: number;
  response_time_ms?: number;
  success: boolean;
  error: string;
  attempt_number: number;
  created_at: string;
}

export interface DashboardStats {
  active_devices: number;
  messages_today: number;
  messages_this_month: number;
  webhook_deliveries: number;
}

export interface LoginResponse {
  user: User;
  access_token: string;
  refresh_token: string;
}

export interface SendMessageRequest {
  device_id: number;
  recipient: string;
  type: string;
  content: string;
  media_url?: string;
}

export interface MetaMessageRequest {
  messaging_product: string;
  recipient_type: string;
  to: string;
  type: string;
  text?: { body: string };
  image?: { link: string; caption: string };
  template?: { name: string; language: { code: string } };
}
