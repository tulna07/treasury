"use client";

import { create } from "zustand";
import { persist, createJSONStorage } from "zustand/middleware";
import { api } from "./api";

export interface User {
  id: string;
  username: string;
  name: string;
  email: string;
  avatar?: string;
  roles: string[];
  roleLabel: string;
  department: string;
  branchId: string;
  branchName: string;
  permissions: string[];
}

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (username: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  fetchUser: () => Promise<void>;
}

function getRoleLabel(roles: string[]): string {
  if (roles.length === 0) return "User";
  // Use the first role as display label, formatted from CODE_CASE to Title Case
  return roles[0].replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}

interface ApiUserProfile {
  id: string;
  username: string;
  full_name: string;
  email: string;
  roles: string[];
  permissions: string[];
  branch_id: string;
  branch_name: string;
  department: string;
  position: string;
  is_active: boolean;
}

interface ApiResponse<T> {
  success: boolean;
  data: T;
  error?: { message: string };
}

function profileToUser(profile: ApiUserProfile): User {
  return {
    id: profile.id,
    username: profile.username,
    name: profile.full_name,
    email: profile.email,
    roles: profile.roles,
    roleLabel: getRoleLabel(profile.roles),
    department: profile.department || profile.branch_name || "",
    branchId: profile.branch_id,
    branchName: profile.branch_name,
    permissions: profile.permissions ?? [],
  };
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      isAuthenticated: false,
      isLoading: false,

      login: async (username: string, password: string) => {
        set({ isLoading: true });
        try {
          const res = await api.post<ApiResponse<{ user: ApiUserProfile }>>("/auth/login", {
            username,
            password,
          });
          set({
            user: profileToUser(res.data.user),
            isAuthenticated: true,
            isLoading: false,
          });
        } catch (err) {
          set({ isLoading: false });
          throw err;
        }
      },

      logout: async () => {
        try {
          await api.post("/auth/logout");
        } catch {
          // Clear local state even if API call fails
        }
        set({ user: null, isAuthenticated: false });
      },

      fetchUser: async () => {
        set({ isLoading: true });
        try {
          const res = await api.get<ApiResponse<ApiUserProfile>>("/auth/me");
          set({
            user: profileToUser(res.data),
            isAuthenticated: true,
            isLoading: false,
          });
        } catch {
          set({ user: null, isAuthenticated: false, isLoading: false });
        }
      },
    }),
    {
      name: "treasury-auth",
      storage: createJSONStorage(() => sessionStorage),
      partialize: (state) => ({
        user: state.user,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
);
