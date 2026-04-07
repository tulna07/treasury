# Nghiên cứu thị trường — Hệ thống Treasury Management tại ngân hàng

> **Mục đích:** Tham khảo để xây dựng sản phẩm phần mềm Treasury riêng cho KienlongBank  
> **Ngày nghiên cứu:** 31/03/2026  
> **Nguồn:** Deep research từ nhiều nguồn quốc tế và trong nước

---

## 1. Tổng quan — Treasury Management System (TMS) là gì?

TMS là hệ thống phần mềm quản lý toàn bộ hoạt động ngân quỹ (Treasury) của ngân hàng, bao gồm:
- Quản lý thanh khoản & dòng tiền
- Giao dịch liên ngân hàng, ngoại tệ, trái phiếu, phái sinh
- Quản lý rủi ro (lãi suất, tỷ giá, tín dụng đối tác)
- Hạch toán, đối chiếu, báo cáo NHNN

---

## 2. Kiến trúc chuẩn — Mô hình 3 tầng (Front / Middle / Back Office)

### 2.1 Front Office (Giao dịch)
| Chức năng | Mô tả |
|-----------|-------|
| Deal Entry | Nhập lệnh giao dịch (FX, MM, Bonds, Derivatives) |
| Pricing & Analytics | Định giá real-time, phân tích trước giao dịch |
| Position Management | Theo dõi vị thế (position) theo từng sản phẩm, đối tác, tiền tệ |
| Dealing Platform | Kết nối sàn giao dịch (Reuters, Bloomberg, 360T) |
| Sales Desk | Hỗ trợ bán sản phẩm Treasury cho khách hàng doanh nghiệp |

### 2.2 Middle Office (Kiểm soát & Rủi ro)
| Chức năng | Mô tả |
|-----------|-------|
| Limit Monitoring | Giám sát hạn mức giao dịch real-time |
| Risk Analytics | Tính VaR, stress test, scenario analysis |
| Compliance | Kiểm tra tuân thủ quy định nội bộ & NHNN |
| P&L Monitoring | Theo dõi lãi/lỗ theo danh mục, dealer, sản phẩm |
| Performance Measurement | Đánh giá hiệu suất đầu tư |

### 2.3 Back Office (Hạch toán & Thanh toán)
| Chức năng | Mô tả |
|-----------|-------|
| Settlement | Thanh toán giao dịch (SWIFT MT300, MT320...) |
| Confirmation | Xác nhận giao dịch với đối tác |
| Accounting | Hạch toán tự động vào core banking (GL) |
| Reconciliation | Đối chiếu giao dịch, tài khoản nostro/vostro |
| Regulatory Reporting | Báo cáo NHNN, kiểm toán |

---

## 3. Các giải pháp Treasury hàng đầu thế giới

### 3.1 So sánh 3 hệ thống lớn nhất

| Tiêu chí | Finastra Kondor | Murex MX.3 | Nasdaq Calypso |
|----------|----------------|------------|----------------|
| **Định vị** | Front-to-back Treasury cho ngân hàng | Enterprise-wide trading & risk | Capital markets & Treasury tích hợp |
| **Thế mạnh** | Cloud-ready, triển khai nhanh, UX hiện đại | Real-time risk, regulatory compliance (FRTB), MXGO cho bank nhỏ | Quản lý balance sheet & thanh khoản tích hợp |
| **Kiến trúc** | Microservices, container, cloud-agnostic | API mở, công nghệ open-source | Unified platform, in-memory processing |
| **Triển khai** | ~6 tháng (LPBank Vietnam) | Lịch sử 100% delivery thành công | Tùy quy mô |
| **Chi phí** | Cao (license + triển khai) | Rất cao | Cao |
| **Phù hợp** | Bank vừa & lớn | Bank lớn, phức tạp | Bank lớn, capital markets |

### 3.2 Các giải pháp khác đáng chú ý
- **Temenos** — Core banking + Treasury module (SacomBank, VIB, PVComBank dùng)
- **Oracle Flexcube OBTR** — Treasury module mở rộng trên Flexcube (liên quan trực tiếp KienlongBank)
- **Kyriba** — TMS cho corporate treasury, được Euromoney bình chọn Best TMS 2025
- **GTreasury** — Cloud-native, phù hợp mid-market

---

## 4. Thực trạng tại Việt Nam

### 4.1 Các ngân hàng Việt Nam đã triển khai TMS

| Ngân hàng | Hệ thống | Năm | Ghi chú |
|-----------|----------|-----|---------|
| **LPBank** | Finastra Kondor | 2024 | Triển khai 6 tháng, front-to-back |
| **Techcombank** | Finastra FusionCapital | 2016 | Tối ưu markets & treasury |
| **VPBank** | Finastra Fusion Kondor + Fusion Risk | 2020 | Treasury + quản lý rủi ro |
| **SacomBank** | Temenos | — | Core banking + Treasury |
| **VIB** | Temenos (cloud) | — | Core banking mới nhất |
| **PVComBank** | Temenos | — | Core banking + Treasury |

### 4.2 Thực trạng chung
- Nhiều bank nhỏ-vừa vẫn **dùng thủ công hoặc Excel** cho Treasury
- Core banking (Flexcube, Temenos) có module Treasury nhưng **chỉ ở mức hạch toán** — thiếu Front & Middle Office
- Xu hướng: Tách riêng hệ thống Treasury khỏi core, kết nối qua API

---

## 5. Oracle Flexcube Treasury (OBTR) — Liên quan trực tiếp KienlongBank

### 5.1 Khả năng hiện có
- Hỗ trợ đa dạng sản phẩm: FX, Money Market, Bonds, Derivatives
- Xử lý back-office: settlement, confirmation, accounting
- Tạo SWIFT messages tự động (MT103, MT202, MT300)
- Hỗ trợ multi-entity, multi-currency
- Tuân thủ quy định: SFTR, EMIR

### 5.2 Hạn chế
| Hạn chế | Ảnh hưởng |
|---------|-----------|
| **Chỉ mạnh back-office** | Thiếu Front Office (dealing, pricing) và Middle Office (risk analytics) |
| **Triển khai phức tạp** | Cần chuyên gia Oracle, thời gian dài |
| **Chi phí cao** | License + customization tốn kém |
| **Workflow cứng** | Không phù hợp quy trình riêng của từng bank, cần customize nhiều |
| **UX lỗi thời** | Giao diện cũ, không thân thiện |
| **Support phân mảnh** | Hỗ trợ từ Oracle chậm, đắt, chia nhiều bộ phận |

### 5.3 Kết luận cho KienlongBank
> Flexcube OBTR hiện tại **chỉ đáp ứng phần hạch toán (Back Office)**. Để vận hành Treasury hiệu quả, KienlongBank cần bổ sung **Front Office + Middle Office** — đây chính là phần mềm cần xây dựng.

---

## 6. Phân tích Build vs Buy cho KienlongBank

### 6.1 Mua giải pháp có sẵn (Kondor, Murex, Calypso)

| ✅ Ưu điểm | ❌ Nhược điểm |
|------------|--------------|
| Triển khai nhanh (6-12 tháng) | Chi phí rất cao (hàng triệu USD license) |
| Đã kiểm chứng thị trường | Phụ thuộc vendor, vendor lock-in |
| Có sẵn regulatory compliance | Customization hạn chế |
| Vendor support & update | Không tạo lợi thế cạnh tranh |
| | Tích hợp Flexcube có thể phức tạp |

### 6.2 Xây dựng riêng (Custom TMS)

| ✅ Ưu điểm | ❌ Nhược điểm |
|------------|--------------|
| Phù hợp 100% quy trình KienlongBank | Thời gian phát triển dài (12-24 tháng) |
| Toàn quyền kiểm soát & mở rộng | Cần đội ngũ có chuyên môn Treasury + Tech |
| Tích hợp mượt với Flexcube core | Chi phí phát triển ban đầu cao |
| Không phụ thuộc vendor | Phải tự maintain & update |
| Tạo lợi thế cạnh tranh | Rủi ro dự án nếu quản lý kém |
| Chi phí dài hạn thấp hơn | |

### 6.3 Đề xuất cho KienlongBank: **Hybrid — Build Custom + Tích hợp Flexcube**

**Lý do:**
1. Flexcube đã có → giữ nguyên Back Office (hạch toán, settlement, SWIFT)
2. Build custom **Front Office** (dealing, position, pricing) + **Middle Office** (risk, limit, compliance)
3. Kết nối với Flexcube qua API/interface → tận dụng đầu tư hiện có
4. Phát triển theo giai đoạn (phase) → giảm rủi ro

---

## 7. Đề xuất kiến trúc & Roadmap

### 7.1 Kiến trúc tổng quan

```
┌─────────────────────────────────────────────────┐
│              CUSTOM TMS (Xây mới)               │
│  ┌─────────────┐  ┌──────────────────────────┐  │
│  │ FRONT OFFICE │  │     MIDDLE OFFICE        │  │
│  │ • Deal Entry │  │ • Risk Analytics (VaR)   │  │
│  │ • FX/MM/Bond │  │ • Limit Monitoring       │  │
│  │ • Position   │  │ • P&L Tracking           │  │
│  │ • Pricing    │  │ • Compliance & Reporting │  │
│  └──────┬──────┘  └────────────┬─────────────┘  │
│         │                      │                 │
│  ┌──────┴──────────────────────┴─────────────┐  │
│  │          INTEGRATION LAYER (API)          │  │
│  └──────────────────┬────────────────────────┘  │
└─────────────────────┼───────────────────────────┘
                      │
┌─────────────────────┼───────────────────────────┐
│     FLEXCUBE CORE (Giữ nguyên — Back Office)    │
│  • Settlement  • Accounting  • SWIFT Messaging  │
│  • Reconciliation  • GL Integration             │
└─────────────────────────────────────────────────┘
```

### 7.2 Roadmap đề xuất

| Phase | Thời gian | Nội dung |
|-------|-----------|----------|
| **Phase 1** | Tháng 1-3 | Khảo sát nghiệp vụ chi tiết, thiết kế kiến trúc, prototype |
| **Phase 2** | Tháng 4-8 | Xây dựng Front Office (Deal Entry, FX, MM, Bonds, Position) |
| **Phase 3** | Tháng 9-12 | Xây dựng Middle Office (Risk, Limit, P&L, Compliance) |
| **Phase 4** | Tháng 13-15 | Tích hợp Flexcube, UAT, Go-live |
| **Phase 5** | Tháng 16+ | Mở rộng (Derivatives, AI analytics, Auto-hedging) |

---

## 8. Tech Stack đề xuất

| Thành phần | Công nghệ | Lý do |
|------------|-----------|-------|
| Backend | Golang / Java | Hiệu suất cao, phù hợp tài chính |
| Frontend | React / Next.js | UX hiện đại, responsive |
| Database | PostgreSQL + TimescaleDB | Dữ liệu giao dịch + time-series (market data) |
| Cache | Redis | Real-time position, pricing |
| Message Queue | Kafka / RabbitMQ | Event-driven, xử lý bất đồng bộ |
| API Gateway | Kong / custom | Kết nối Flexcube, market data |
| Reporting | Apache Superset / custom | Dashboard, báo cáo NHNN |
| Auth | Zitadel / Keycloak | SSO, phân quyền chi tiết |

---

> **Kết luận:** KienlongBank nên xây dựng Custom TMS cho Front & Middle Office, giữ nguyên Flexcube làm Back Office. Tiếp cận Hybrid giúp tận dụng đầu tư hiện có, đồng thời tạo ra sản phẩm phù hợp đặc thù ngân hàng và có lợi thế cạnh tranh dài hạn.
