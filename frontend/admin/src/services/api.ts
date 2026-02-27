import axios, { AxiosInstance, AxiosError } from 'axios';
import { useAuthStore } from '@/store/authStore';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';

class ApiService {
  private client: AxiosInstance;

  constructor() {
    this.client = axios.create({
      baseURL: API_BASE_URL,
      headers: {
        'Content-Type': 'application/json',
      },
      timeout: 30000,
    });

    this.setupInterceptors();
  }

  private setupInterceptors() {
    // Request interceptor
    this.client.interceptors.request.use(
      (config) => {
        const token = useAuthStore.getState().token;
        if (token) {
          config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
      },
      (error) => Promise.reject(error)
    );

    // Response interceptor
    this.client.interceptors.response.use(
      (response) => response,
      (error: AxiosError) => {
        if (error.response?.status === 401) {
          useAuthStore.getState().logout();
          window.location.href = '/login';
        }
        return Promise.reject(error);
      }
    );
  }

  getClient(): AxiosInstance {
    return this.client;
  }
}

export const apiService = new ApiService();
export const api = apiService.getClient();

// API endpoints
export const endpoints = {
  auth: {
    login: '/api/v1/auth/login',
    logout: '/api/v1/auth/logout',
    refresh: '/api/v1/auth/refresh',
    me: '/api/v1/auth/me',
  },
  drivers: {
    list: '/api/v1/admin/drivers',
    detail: (id: string) => `/api/v1/admin/drivers/${id}`,
    approve: (id: string) => `/api/v1/admin/drivers/${id}/approve`,
    reject: (id: string) => `/api/v1/admin/drivers/${id}/reject`,
    documents: (id: string) => `/api/v1/admin/drivers/${id}/documents`,
    vehicles: (id: string) => `/api/v1/admin/drivers/${id}/vehicles`,
  },
  riders: {
    list: '/api/v1/admin/riders',
    detail: (id: string) => `/api/v1/admin/riders/${id}`,
    suspend: (id: string) => `/api/v1/admin/riders/${id}/suspend`,
    ban: (id: string) => `/api/v1/admin/riders/${id}/ban`,
  },
  trips: {
    list: '/api/v1/admin/trips',
    detail: (id: string) => `/api/v1/admin/trips/${id}`,
    active: '/api/v1/admin/trips/active',
    stats: '/api/v1/admin/trips/stats',
  },
  vehicles: {
    list: '/api/v1/admin/vehicles',
    detail: (id: string) => `/api/v1/admin/vehicles/${id}`,
    verify: (id: string) => `/api/v1/admin/vehicles/${id}/verify`,
  },
  analytics: {
    dashboard: '/api/v1/admin/analytics/dashboard',
    revenue: '/api/v1/admin/analytics/revenue',
    trips: '/api/v1/admin/analytics/trips',
  },
  compliance: {
    reports: '/api/v1/admin/compliance/reports',
    report: (id: string) => `/api/v1/admin/compliance/reports/${id}`,
  },
  settings: {
    get: '/api/v1/admin/settings',
    update: '/api/v1/admin/settings',
  },
};
