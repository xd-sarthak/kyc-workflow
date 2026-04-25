import axios from 'axios';

const api = axios.create({
  baseURL: '/api/v1',
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor — attach JWT from localStorage
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Response interceptor — handle 401 globally
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token');
      localStorage.removeItem('role');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

// Auth
export const signup = (email, password, role) =>
  api.post('/signup', { email, password, role });

export const login = (email, password) =>
  api.post('/login', { email, password });

// Merchant KYC
export const saveDraft = (formData) =>
  api.post('/kyc/save-draft', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  });

export const submitKYC = () =>
  api.post('/kyc/submit', {});

export const getMySubmission = () =>
  api.get('/kyc/me');

export const getNotifications = () =>
  api.get('/kyc/notifications');

// Reviewer
export const getQueue = (limit = 20, offset = 0) =>
  api.get(`/reviewer/queue?limit=${limit}&offset=${offset}`);

export const getSubmissionDetail = (id) =>
  api.get(`/reviewer/${id}`);

export const transitionSubmission = (id, to, note = '') =>
  api.post(`/reviewer/${id}/transition`, { to, note });

// Metrics
export const getMetrics = () =>
  api.get('/metrics/');

export default api;
