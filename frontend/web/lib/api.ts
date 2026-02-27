import axios from 'axios';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8000';

export const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

api.interceptors.request.use(
  (config) => {
    const token = typeof window !== 'undefined' ? localStorage.getItem('access_token') : null;
    if (token) {
      config.headers.Authorization = 'Bearer ' + token;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    if (error.response?.status === 401) {
      if (typeof window !== 'undefined') {
        localStorage.removeItem('access_token');
        window.location.href = '/login';
      }
    }
    return Promise.reject(error);
  }
);

export const authApi = {
  login: (email: string, password: string) =>
    api.post('/auth/login', { email, password }),
  register: (data: any) => api.post('/auth/register', data),
  refresh: () => api.post('/auth/refresh'),
  logout: () => api.post('/auth/logout'),
};

export const riderApi = {
  getProfile: () => api.get('/riders/profile'),
  updateProfile: (data: any) => api.patch('/riders/profile', data),
  getTrips: () => api.get('/riders/trips'),
  bookTrip: (data: any) => api.post('/trips', data),
  getPaymentMethods: () => api.get('/riders/payment-methods'),
  addPaymentMethod: (data: any) => api.post('/riders/payment-methods', data),
};

export const driverApi = {
  getProfile: () => api.get('/drivers/profile'),
  updateProfile: (data: any) => api.patch('/drivers/profile', data),
  getDocuments: () => api.get('/drivers/documents'),
  uploadDocument: (formData: FormData) =>
    api.post('/drivers/documents', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    }),
  getEarnings: () => api.get('/drivers/earnings'),
  toggleAvailability: (available: boolean) =>
    api.patch('/drivers/availability', { available }),
  getTrips: () => api.get('/drivers/trips'),
};

export default api;