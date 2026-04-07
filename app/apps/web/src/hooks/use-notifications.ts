"use client";

import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { toast } from "sonner";

// ─── Types ────────────────────────────────────────────────────

export interface Notification {
  id: string;
  title: string;
  message: string;
  type: string;
  is_read: boolean;
  related_id?: string;
  related_module?: string;
  created_at: string;
}

interface NotificationListResponse {
  data: Notification[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

interface ApiResponse<T> {
  success: boolean;
  data: T;
}

// ─── SSE Hook ─────────────────────────────────────────────────

export function useSSENotifications() {
  const [unreadCount, setUnreadCount] = useState(0);

  useEffect(() => {
    const es = new EventSource("/api/v1/notifications/stream");

    es.addEventListener("badge_update", (e) => {
      const data = JSON.parse(e.data);
      setUnreadCount(data.count);
    });

    es.addEventListener("notification", (e) => {
      const notif = JSON.parse(e.data);
      toast.info(notif.title, { description: notif.message });
      setUnreadCount((prev) => prev + 1);
    });

    es.onerror = () => {
      // EventSource auto-reconnects
    };

    return () => es.close();
  }, []);

  return { unreadCount, setUnreadCount };
}

// ─── REST Hooks ───────────────────────────────────────────────

export function useNotifications(page: number) {
  return useQuery({
    queryKey: ["notifications", page],
    queryFn: async () => {
      const res = await api.get<ApiResponse<NotificationListResponse>>(
        "/notifications",
        { params: { page: String(page), page_size: "20" } }
      );
      return res.data;
    },
  });
}

export function useUnreadNotifications() {
  return useQuery({
    queryKey: ["notifications", "unread"],
    queryFn: async () => {
      const res = await api.get<ApiResponse<NotificationListResponse>>(
        "/notifications",
        { params: { page: "1", page_size: "5", is_read: "false" } }
      );
      return res.data;
    },
    refetchInterval: 30000,
  });
}

export function useMarkRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      await api.post(`/notifications/${id}/read`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notifications"] });
    },
  });
}

export function useMarkAllRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      await api.post("/notifications/read-all");
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notifications"] });
    },
  });
}
