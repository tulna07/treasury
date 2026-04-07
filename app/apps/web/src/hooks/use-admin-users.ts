"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

// ─── Types ────────────────────────────────────────────────────

export interface AdminUser {
  id: string;
  username: string;
  full_name: string;
  email: string;
  department: string;
  position: string;
  branch_id: string;
  branch_name: string;
  is_active: boolean;
  roles: string[];
  role_names: string[];
  last_login_at: string | null;
  created_at: string;
}

export interface AdminUserFilters {
  search?: string;
  department?: string;
  is_active?: string;
  page?: number;
  page_size?: number;
}

export interface CreateUserRequest {
  username: string;
  password: string;
  full_name: string;
  email: string;
  branch_id: string;
  department: string;
  position: string;
}

export interface UpdateUserRequest {
  full_name: string;
  email: string;
  department: string;
  position: string;
}

export interface ReasonRequest {
  reason: string;
}

export interface ResetPasswordResponse {
  temp_password: string;
}

export interface AdminUserListResponse {
  data: AdminUser[];
  total: number;
  page: number;
  page_size: number;
}

// ─── API Response Wrapper ─────────────────────────────────────

interface ApiResponse<T> {
  success: boolean;
  data: T;
}

// ─── Query Keys ───────────────────────────────────────────────

const adminUserKeys = {
  all: ["admin-users"] as const,
  lists: () => [...adminUserKeys.all, "list"] as const,
  list: (filters: AdminUserFilters) => [...adminUserKeys.lists(), filters] as const,
  details: () => [...adminUserKeys.all, "detail"] as const,
  detail: (id: string) => [...adminUserKeys.details(), id] as const,
};

// ─── Hooks ────────────────────────────────────────────────────

export function useAdminUsers(filters: AdminUserFilters = {}) {
  const params: Record<string, string> = {};
  if (filters.search) params.search = filters.search;
  if (filters.department) params.department = filters.department;
  if (filters.is_active) params.is_active = filters.is_active;
  if (filters.page) params.page = String(filters.page);
  if (filters.page_size) params.page_size = String(filters.page_size);

  return useQuery({
    queryKey: adminUserKeys.list(filters),
    queryFn: async () => {
      const res = await api.get<ApiResponse<AdminUserListResponse>>(
        "/admin/users",
        { params }
      );
      return res.data;
    },
  });
}

export function useAdminUser(id: string) {
  return useQuery({
    queryKey: adminUserKeys.detail(id),
    queryFn: async () => {
      const res = await api.get<ApiResponse<AdminUser>>(`/admin/users/${id}`);
      return res.data;
    },
    enabled: !!id,
  });
}

export function useCreateAdminUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: CreateUserRequest) => {
      const res = await api.post<ApiResponse<AdminUser>>("/admin/users", data);
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: adminUserKeys.lists() });
    },
  });
}

export function useUpdateAdminUser(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: UpdateUserRequest) => {
      const res = await api.put<ApiResponse<AdminUser>>(`/admin/users/${id}`, data);
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: adminUserKeys.lists() });
      queryClient.invalidateQueries({ queryKey: adminUserKeys.detail(id) });
    },
  });
}

export function useLockUser(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: ReasonRequest) => {
      const res = await api.post<ApiResponse<AdminUser>>(
        `/admin/users/${id}/lock`,
        data
      );
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: adminUserKeys.lists() });
      queryClient.invalidateQueries({ queryKey: adminUserKeys.detail(id) });
    },
  });
}

export function useUnlockUser(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: ReasonRequest) => {
      const res = await api.post<ApiResponse<AdminUser>>(
        `/admin/users/${id}/unlock`,
        data
      );
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: adminUserKeys.lists() });
      queryClient.invalidateQueries({ queryKey: adminUserKeys.detail(id) });
    },
  });
}

export function useResetPassword(id: string) {
  return useMutation({
    mutationFn: async (data: ReasonRequest) => {
      const res = await api.post<ApiResponse<ResetPasswordResponse>>(
        `/admin/users/${id}/reset-password`,
        data
      );
      return res.data;
    },
  });
}

export function useAssignRole(userId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (data: { role_code: string; reason: string }) => {
      const res = await api.post<ApiResponse<unknown>>(
        `/admin/users/${userId}/roles`,
        data
      );
      return res.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: adminUserKeys.detail(userId) });
      queryClient.invalidateQueries({ queryKey: adminUserKeys.lists() });
    },
  });
}

export function useRevokeRole(userId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ roleCode, reason }: { roleCode: string; reason: string }) => {
      const res = await api.delete<ApiResponse<unknown>>(
        `/admin/users/${userId}/roles/${roleCode}`,
        {
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ reason }),
        }
      );
      return res;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: adminUserKeys.detail(userId) });
      queryClient.invalidateQueries({ queryKey: adminUserKeys.lists() });
    },
  });
}
