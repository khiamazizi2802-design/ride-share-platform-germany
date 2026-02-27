import { create } from 'zustand';
import { DashboardStats, Notification } from '@/types';

interface DashboardState {
  stats: DashboardStats;
  notifications: Notification[];
  isSidebarOpen: boolean;
  setStats: (stats: DashboardStats) => void;
  addNotification: (notification: Notification) => void;
  markNotificationAsRead: (id: string) => void;
  toggleSidebar: () => void;
  setSidebarOpen: (open: boolean) => void;
}

export const useDashboardStore = create<DashboardState>((set) => ({
  stats: {
    todayRevenue: 0,
    todayTrips: 0,
    activeTrips: 0,
    onlineDrivers: 0,
    pendingVerifications: 0,
    openSupportTickets: 0,
  },
  notifications: [],
  isSidebarOpen: true,

  setStats: (stats) => set({ stats }),

  addNotification: (notification) =>
    set((state) => ({
      notifications: [notification, ...state.notifications],
    })),

  markNotificationAsRead: (id) =>
    set((state) => ({
      notifications: state.notifications.map((n) =>
        n.id === id ? { ...n, isRead: true } : n
      ),
    })),

  toggleSidebar: () =>
    set((state) => ({ isSidebarOpen: !state.isSidebarOpen })),

  setSidebarOpen: (open) => set({ isSidebarOpen: open }),
}));
