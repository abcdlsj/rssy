import { create } from 'zustand';

interface LayoutStore {
  isSidebarOpen: boolean;
  toggleSidebar: () => void;
}

export const useLayoutStore = create<LayoutStore>((set) => ({
  isSidebarOpen: false,
  toggleSidebar: () => set((state) => ({ isSidebarOpen: !state.isSidebarOpen })),
}));