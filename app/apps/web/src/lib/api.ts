/**
 * API client cho Treasury System
 * Wrapper fetch với xử lý lỗi, authentication, và auto refresh token.
 *
 * Token flow (HttpOnly cookies):
 * - Login → server set treasury_access_token + treasury_refresh_token (HttpOnly)
 * - Mỗi request gửi kèm cookies (credentials: "include")
 * - Access token hết hạn → 401 → auto call /auth/refresh → retry request gốc
 * - Refresh token hết hạn → redirect login
 */

const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_URL || "/api/v1";

interface ApiOptions extends RequestInit {
  params?: Record<string, string>;
  /** Nếu true, không retry khi 401 (tránh infinite loop) */
  _noRetry?: boolean;
}

interface ApiError {
  error: string;
  status: number;
}

class ApiClient {
  private baseUrl: string;
  /** Đang refresh → queue các request khác chờ */
  private refreshPromise: Promise<boolean> | null = null;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  /**
   * Gọi POST /auth/refresh để lấy access token mới.
   * Refresh token cookie tự động gửi theo request (HttpOnly, path=/api/v1/auth/refresh).
   * Returns true nếu refresh thành công, false nếu thất bại.
   */
  private async refreshAccessToken(): Promise<boolean> {
    try {
      const resp = await fetch(`${this.baseUrl}/auth/refresh`, {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
      });
      return resp.ok;
    } catch {
      return false;
    }
  }

  /**
   * Đảm bảo chỉ 1 refresh request chạy tại 1 thời điểm.
   * Các request 401 đồng thời sẽ chờ cùng 1 promise.
   */
  private async tryRefresh(): Promise<boolean> {
    if (!this.refreshPromise) {
      this.refreshPromise = this.refreshAccessToken().finally(() => {
        this.refreshPromise = null;
      });
    }
    return this.refreshPromise;
  }

  private async request<T>(
    endpoint: string,
    options: ApiOptions = {}
  ): Promise<T> {
    const { params, _noRetry, ...fetchOptions } = options;

    let url = `${this.baseUrl}${endpoint}`;
    if (params) {
      const searchParams = new URLSearchParams(params);
      url += `?${searchParams.toString()}`;
    }

    const response = await fetch(url, {
      ...fetchOptions,
      headers: {
        "Content-Type": "application/json",
        ...fetchOptions.headers,
      },
      credentials: "include",
    });

    // 401 + chưa retry → thử refresh token
    if (response.status === 401 && !_noRetry) {
      // Không refresh cho chính endpoint auth (login, refresh, logout)
      if (!endpoint.startsWith("/auth/")) {
        const refreshed = await this.tryRefresh();
        if (refreshed) {
          // Retry request gốc với flag _noRetry để tránh infinite loop
          return this.request<T>(endpoint, { ...options, _noRetry: true });
        }
      }

      // Refresh thất bại → clear auth state + redirect login
      if (typeof window !== "undefined" && !endpoint.startsWith("/auth/")) {
        const { useAuthStore } = await import("./auth-store");
        useAuthStore.setState({ user: null, isAuthenticated: false });
        // Chỉ redirect nếu chưa ở trang login
        if (!window.location.pathname.includes("/login")) {
          window.location.href = "/login";
        }
      }
    }

    if (!response.ok) {
      const error: ApiError = {
        error: `Lỗi ${response.status}: ${response.statusText}`,
        status: response.status,
      };
      try {
        const body = await response.json();
        error.error = body.error?.message || body.error || error.error;
      } catch {
        // Giữ lỗi mặc định
      }
      throw error;
    }

    // 204 No Content — no body to parse
    if (response.status === 204) {
      return undefined as T;
    }

    return response.json();
  }

  get<T>(endpoint: string, options?: ApiOptions) {
    return this.request<T>(endpoint, { ...options, method: "GET" });
  }

  post<T>(endpoint: string, data?: unknown, options?: ApiOptions) {
    return this.request<T>(endpoint, {
      ...options,
      method: "POST",
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  put<T>(endpoint: string, data?: unknown, options?: ApiOptions) {
    return this.request<T>(endpoint, {
      ...options,
      method: "PUT",
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  delete<T>(endpoint: string, options?: ApiOptions) {
    return this.request<T>(endpoint, { ...options, method: "DELETE" });
  }
}

export const api = new ApiClient(API_BASE_URL);
