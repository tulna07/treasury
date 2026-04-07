"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

// ─── Types ────────────────────────────────────────────────────

export interface AdminRole {
  id: string;
  code: string;
  name: string;
  description: string;
  scope: string;
}

export interface RolePermission {
  id: string;
  code: string;
  name: string;
  description: string;
}

interface RolePermissionResponse {
  role_code: string;
  role_name: string;
  permissions: string[];
}

// ─── API Response Wrapper ─────────────────────────────────────

interface ApiResponse<T> {
  success: boolean;
  data: T;
}

// ─── Query Keys ───────────────────────────────────────────────

export const roleKeys = {
  all: ["admin-roles"] as const,
  list: () => [...roleKeys.all, "list"] as const,
  permissions: (code: string) => [...roleKeys.all, "permissions", code] as const,
};

// ─── Hooks ────────────────────────────────────────────────────

export function useAdminRoles() {
  return useQuery({
    queryKey: roleKeys.list(),
    queryFn: async () => {
      const res = await api.get<ApiResponse<AdminRole[]>>("/admin/roles");
      return res.data;
    },
  });
}

export function useRolePermissions(code: string) {
  return useQuery({
    queryKey: roleKeys.permissions(code),
    queryFn: async () => {
      const res = await api.get<ApiResponse<RolePermissionResponse>>(
        `/admin/roles/${code}/permissions`
      );
      return res.data.permissions;
    },
    enabled: !!code,
  });
}

// ─── All Permissions ─────────────────────────────────────────

export function useAllPermissions() {
  return useQuery({
    queryKey: ["admin-permissions"],
    queryFn: async () => {
      const res = await api.get<ApiResponse<RolePermission[]>>(
        "/admin/permissions"
      );
      return res.data;
    },
  });
}

// ─── Update Role Permissions ─────────────────────────────────

export function useUpdateRolePermissions() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      roleCode,
      permissions,
    }: {
      roleCode: string;
      permissions: string[];
    }) => {
      await api.put(`/admin/roles/${roleCode}/permissions`, { permissions });
    },
    onSuccess: (_, { roleCode }) => {
      queryClient.invalidateQueries({
        queryKey: roleKeys.permissions(roleCode),
      });
    },
  });
}
