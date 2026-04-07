# SIZING HỆ THỐNG TREASURY — KIENLONGBANK

> Ngày: 06/04/2026 | Phiên bản: 1.0 | Trạng thái: Phase 1

---

## 1. Tổng quan hệ thống

| Thành phần | Công nghệ | Mô tả |
|-----------|-----------|-------|
| Treasury API | Go (Golang) | Backend — 54 API endpoints, 157 files, ~40K dòng code |
| Treasury Web | Next.js 15 (React) | Frontend — 34 pages, ~27K dòng code |
| Auth Server | Zitadel 2.x | SSO/OIDC — quản lý user, role, session |
| Database | PostgreSQL 17 | 47 bảng, 13 migrations |
| Reverse Proxy | Nginx hoặc Caddy | SSL termination, routing |
| File Storage | Local / MinIO | Lưu trữ file đính kèm (ticket, hợp đồng) |

---

## 2. Quy mô sử dụng

| Chỉ số | Giá trị |
|--------|---------|
| Số role | 10 |
| Số user account | 20–30 |
| Đồng thời cao nhất | 10–20 users (giờ giao dịch sáng) |
| Số module | 5 (FX, GTCG, MM, Hạn mức, TTQT) |
| Giao dịch/ngày (ước tính) | 30–50 |
| Giao dịch/năm (ước tính) | ~10,000 |
| Môi trường | Mạng nội bộ KienlongBank (không expose internet) |

---

## 3. Sizing Server — 2 phương án

### Option A: Tối thiểu (1 server)

> Phù hợp: môi trường dev/staging hoặc pilot < 50 users

| Tài nguyên | Spec |
|-----------|------|
| Server | 1 máy (bare metal hoặc VM) |
| CPU | 4 vCPU |
| RAM | 8 GB |
| Storage | 100 GB SSD |
| OS | Ubuntu 22.04 LTS / RHEL 8+ |
| Network | 1 Gbps internal |

### Option B: Khuyến nghị Production (HA-ready)

> Phù hợp: production, hỗ trợ failover

| Tài nguyên | Spec |
|-----------|------|
| **App Server** (×2) | Active-Active hoặc Active-Standby |
| CPU / máy | 4 vCPU |
| RAM / máy | 16 GB |
| **DB Server** (×1) | Riêng biệt hoặc Managed PostgreSQL |
| DB CPU | 4 vCPU |
| DB RAM | 8 GB |
| DB Storage | 200 GB SSD (IOPS ≥ 3000) |
| OS | Ubuntu 22.04 LTS / RHEL 8+ |
| Network | 1 Gbps internal |

---

## 4. Benchmark thực tế (Dev Environment)

> Đo trên Mac mini M2, macOS, PostgreSQL 17

| Service | RAM sử dụng | CPU (idle) |
|---------|------------|------------|
| Treasury API (Go binary) | 44 MB | < 1% |
| Treasury Web (Next.js) | 77 MB | < 1% |
| Zitadel (Auth) | 126 MB | < 1% |
| PostgreSQL | ~50 MB | < 1% |
| **Tổng** | **~300 MB** | — |

→ Hệ thống rất nhẹ nhờ Go + Next.js. RAM thực tế < 500 MB cho toàn bộ stack.

---

## 5. Software Requirements

| Software | Version | Ghi chú |
|----------|---------|---------|
| Docker + Docker Compose | 24+ | Deploy containerized (khuyến nghị) |
| PostgreSQL | 16 hoặc 17 | Native hoặc Docker |
| Node.js | 20 LTS | Cho Treasury Web |
| Go | 1.22+ | Cho Treasury API (hoặc deploy binary) |
| Zitadel | 2.x | Auth server |
| Nginx / Caddy | Latest | Reverse proxy + TLS |

---

## 6. Network & Port Requirements

| Port | Service | Hướng | Ghi chú |
|------|---------|-------|---------|
| 443 (HTTPS) | Web UI | Inbound (user) | SSL termination tại reverse proxy |
| 8080 | Treasury API | Internal | Không expose ra ngoài |
| 3000 | Treasury Web | Internal | Không expose ra ngoài |
| 8443 | Zitadel | Internal | Auth server |
| 5432 | PostgreSQL | Internal | DB — chỉ cho app server kết nối |

---

## 7. Ước tính dung lượng lưu trữ

### Database (PostgreSQL)

| Loại dữ liệu | Records/năm | Dung lượng/năm |
|--------------|-------------|----------------|
| Giao dịch FX/Bond/MM | ~10,000 | ~500 MB |
| Audit trail | ~100,000 | ~1 GB |
| Tỷ giá lịch sử (poll 5 phút) | ~500,000 | ~2 GB |
| Master data (đối tác, TP, user) | ~1,000 | < 10 MB |
| **Tổng DB** | | **~4 GB/năm** |

### File Storage

| Loại file | Ước tính/năm |
|-----------|-------------|
| File đính kèm (PDF, ticket) | ~5 GB |
| PDF export | ~1 GB |
| **Tổng file** | **~6 GB/năm** |

### Tổng dung lượng

| Năm | DB | Files | Tổng |
|-----|-----|-------|------|
| Năm 1 | 4 GB | 6 GB | **10 GB** |
| Năm 3 | 12 GB | 18 GB | **30 GB** |
| Năm 5 | 20 GB | 30 GB | **50 GB** |

→ Storage 100–200 GB SSD đủ dùng 5+ năm.

---

## 8. Backup & Recovery

| Hạng mục | Đề xuất |
|---------|---------|
| Database backup | `pg_dump` hàng ngày, lưu 30 ngày |
| File backup | rsync/rclone hàng ngày |
| Backup storage | Separate disk hoặc NAS |
| RTO (Recovery Time) | < 2 giờ |
| RPO (Recovery Point) | < 24 giờ (daily backup) |

---

## 9. Monitoring (khuyến nghị)

| Tool | Mục đích |
|------|---------|
| Prometheus + Grafana | Metrics hệ thống (CPU, RAM, disk) |
| Loki / ELK | Log aggregation |
| Uptime Kuma | Health check endpoints |
| PostgreSQL pg_stat | DB performance |

---

## 10. Deployment Architecture

```
                    ┌─────────────┐
                    │   Nginx     │
                    │  (SSL/443)  │
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
        ┌─────┴─────┐ ┌───┴────┐ ┌─────┴─────┐
        │ Treasury  │ │Treasury│ │  Zitadel  │
        │   Web     │ │  API   │ │  (Auth)   │
        │  :3000    │ │ :8080  │ │  :8443    │
        └───────────┘ └───┬────┘ └───────────┘
                          │
                    ┌─────┴─────┐
                    │PostgreSQL │
                    │  :5432    │
                    └───────────┘
```

---

## 11. Checklist triển khai

- [ ] Chuẩn bị server theo Option A hoặc B
- [ ] Cài đặt Docker + Docker Compose
- [ ] Deploy PostgreSQL + chạy migrations
- [ ] Deploy Zitadel + cấu hình OIDC
- [ ] Deploy Treasury API + Treasury Web
- [ ] Cấu hình Nginx reverse proxy + SSL
- [ ] Import seed data (đối tác, trái phiếu, user)
- [ ] Test kết nối end-to-end
- [ ] Cấu hình backup schedule
- [ ] Phân quyền user theo ma trận BRD

---

*Tài liệu sizing — Hệ thống Treasury KienlongBank | Phase 1 | 06/04/2026*
