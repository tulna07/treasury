# Nhận xét BRD Treasury System v1

**Tài liệu:** BM01_BRD_TREASURY_SYSTEM
**Soạn thảo:** Trần Thị Linh Phương, Dương Thanh Tùng (K.NV&ĐCTC)
**Ngày review:** 01/04/2026

---

## I. TỔNG QUAN TÀI LIỆU

### Cấu trúc: 5 Module chính
1. **Module 1:** Kinh doanh Ngoại tệ (FX) — Spot, Forward, Swap
2. **Module 2:** Giấy tờ có giá (GTCG) — Govi Bond, FI Bond, CCTG
3. **Module 3:** Kinh doanh tiền tệ (MM — Money Market)
4. **Module 4:** Phê duyệt hạn mức liên ngân hàng
5. **Module 5:** Thanh toán quốc tế (TTQT)

### Luồng nghiệp vụ chung
```
CV tạo GD (Open) → TP K.NV duyệt → GĐ K.NV duyệt → [QLRR duyệt hạn mức (chỉ MM)] → P.KTTC hạch toán → [TTQT nếu có] → Hoàn thành
```

### Người dùng hệ thống
- K.NV&ĐCTC (Khối Nguồn vốn & Định chế Tài chính)
- P.QLRR (Phòng Quản lý rủi ro)
- P.KTTC (Phòng Kế toán – Tài chính)
- TTTT (Trung tâm Thanh toán)

---

## II. ĐIỂM MẠNH

1. **Mô tả field-level chi tiết** — Mỗi trường có: tên, loại (bắt buộc/tự động/tùy chọn), mô tả dữ liệu, ví dụ cụ thể
2. **Luồng phê duyệt rõ ràng** — Workflow có phân cấp: CV → TP → GĐ → các phòng ban liên quan
3. **Xử lý TTQT logic** — Tự động kiểm tra trường TTQT để quyết định có chuyển sang TTTT hay không
4. **Phân hệ hạn mức** — Tách riêng có/không TSBĐ, tính toán hạn mức tự động
5. **Yêu cầu phi chức năng** — Đề cập bảo mật (mạng nội bộ), lưu trữ vĩnh viễn, backup hàng tuần

---

## III. CÁC VẤN ĐỀ CẦN LÀM RÕ / BỔ SUNG

### A. Luồng nghiệp vụ

| # | Vấn đề | Module | Mức độ |
|---|--------|--------|--------|
| 1 | **Luồng hủy chưa hoàn thiện** — BRD chưa mô tả chi tiết: (a) Ai được hủy? (b) Ở trạng thái nào được hủy? (c) Hủy có cần phê duyệt không? (d) Giao dịch đã hạch toán rồi có được reverse không? | Tất cả | Cao |
| 2 | **Luồng sửa/chỉnh sửa giao dịch** — Chỉ đề cập "TP duyệt rồi thì CV không được sửa", nhưng chưa rõ: nếu phát hiện sai sót sau khi TP duyệt thì xử lý thế nào? | Tất cả | Cao |
| 3 | **Giao dịch SWAP chân 2 khi nào trigger?** — Chân 1 và chân 2 có luồng phê duyệt riêng hay chung? Nếu chân 1 hoàn thành, chân 2 tự động chuyển TTQT vào ngày thực hiện? | FX | Cao |
| 4 | **Module MM: Giao dịch "Nhận tiền gửi" khi đáo hạn** — Khi đến ngày đáo hạn, hệ thống có tự động tạo giao dịch nhận tiền về không? Hay cần CV tạo thủ công? | MM | Trung bình |
| 5 | **Module QLRR: Quy trình khi hạn mức vượt** — Nếu "Giá trị còn lại sau giao dịch" < 0, hệ thống block hay vẫn cho duyệt với cảnh báo? | Module 4 | Cao |

### B. Data & Tính toán

| # | Vấn đề | Module | Mức độ |
|---|--------|--------|--------|
| 6 | **Công thức "Thành tiền chân 2" SWAP copy sai** — Đang ghi "Tỷ giá chân 1" thay vì "Tỷ giá chân 2" | FX | Cao (lỗi) |
| 7 | **Format số không nhất quán** — Module FX dùng dấu "," ngăn nghìn + "." thập phân (chuẩn US), nhưng Module GTCG dùng dấu "." ngăn nghìn + "," thập phân (chuẩn VN). Cần thống nhất | Tất cả | Trung bình |
| 8 | **Tỷ giá quy đổi USD→VND** trong hạn mức — Dùng tỷ giá nào? Tỷ giá NHNN? Tỷ giá nội bộ? Cập nhật theo ngày? | Module 4 | Cao |
| 9 | **Công thức "Số tiền tại ngày đáo hạn" MM** — Có vẻ sai dấu ngoặc: `Gốc * [(1 + LS/100) * Kỳ hạn] / 365` → đúng phải là `Gốc * (1 + LS/100 * Kỳ hạn / 365)` hay `Gốc + Gốc * LS/100 * Kỳ hạn / 365`? | MM | Cao (lỗi) |
| 10 | **Date format không nhất quán** — Module FX dùng mm/dd/yyyy, Module GTCG dùng dd/mm/yyyy | Tất cả | Trung bình |

### C. Master Data & Danh mục

| # | Vấn đề | Mức độ |
|---|--------|--------|
| 11 | **Danh sách đối tác** — Cần import từ file Excel, nhưng chưa có cấu trúc file mẫu. Ai quản lý? Cập nhật thế nào? | Cao |
| 12 | **Danh mục trái phiếu** (Import 2) — Cần cấu trúc file, source dữ liệu (HNX? VBMA?), tần suất cập nhật | Cao |
| 13 | **Pay code / SSI** — Danh sách quản lý ở đâu? Ai có quyền thêm/sửa? | Trung bình |
| 14 | **Danh sách CIF khách hàng** — Lấy từ Core Banking (Flexcube) hay quản lý riêng? | Trung bình |

### D. Phân quyền & Audit

| # | Vấn đề | Mức độ |
|---|--------|--------|
| 15 | **Ma trận phân quyền chi tiết chưa có** — Cần bảng: Role × Chức năng × Quyền (View/Create/Edit/Approve/Delete) | Cao |
| 16 | **Audit trail** — Giao dịch tài chính cần log đầy đủ: ai làm gì, lúc nào, thay đổi gì. BRD chưa đề cập | Cao |
| 17 | **Maker-Checker** — Nguyên tắc 4 mắt có áp dụng cho tất cả module không? Hiện tại FX và GTCG chỉ cần 2 cấp duyệt (TP → GĐ), MM cần thêm QLRR | Trung bình |

### E. Tích hợp hệ thống

| # | Vấn đề | Mức độ |
|---|--------|--------|
| 18 | **Tích hợp Flexcube** — Hạch toán tại P.KTTC hiện thủ công hay tự động đẩy sang Core? Nếu thủ công thì hệ thống mới chỉ là workflow, chưa phải Treasury system thực sự | Cao |
| 19 | **Tích hợp Reuters/Bloomberg** — Lấy tỷ giá, giá trái phiếu tự động hay nhập tay? | Trung bình |
| 20 | **Tích hợp SWIFT** — Module TTQT "đẩy điện" nghĩa là gì cụ thể? Giao diện với hệ thống SWIFT hiện tại? | Cao |
| 21 | **Notification** — Gửi thông báo cho CV khi bị từ chối: qua email? In-app? Cả hai? | Trung bình |

### F. Báo cáo & Dashboard

| # | Vấn đề | Mức độ |
|---|--------|--------|
| 22 | **Chỉ có 1 báo cáo** (tổng hợp hạn mức trong ngày). Cần bổ sung: báo cáo vị thế ngoại tệ, báo cáo thanh khoản, báo cáo lãi/lỗ, báo cáo danh mục GTCG | Trung bình |
| 23 | **Dashboard** — Tổng quan giao dịch trong ngày, vị thế, hạn mức hiện tại? | Trung bình |

---

## IV. ĐỀ XUẤT BỔ SUNG CHO VERSION 2

1. **Hoàn thiện luồng hủy/sửa** cho tất cả module (chị Phương Linh đang làm)
2. **Thống nhất format số và ngày** xuyên suốt tài liệu
3. **Bổ sung ma trận phân quyền** chi tiết theo Role
4. **Mô tả rõ tích hợp** với Flexcube, SWIFT, nguồn dữ liệu thị trường
5. **Bổ sung master data specification** — cấu trúc file import, source, quy trình cập nhật
6. **Bổ sung requirements cho audit trail** và logging
7. **Sửa lỗi công thức** tính toán (SWAP chân 2, MM số tiền đáo hạn)

---

## V. TÓM TẮT ĐÁNH GIÁ

| Tiêu chí | Đánh giá |
|----------|----------|
| Độ chi tiết field-level | ⭐⭐⭐⭐ Tốt |
| Luồng nghiệp vụ chính | ⭐⭐⭐ Khá (cần bổ sung luồng hủy/sửa) |
| Master data | ⭐⭐ Cần bổ sung nhiều |
| Phân quyền | ⭐⭐ Cần ma trận chi tiết |
| Tích hợp hệ thống | ⭐⭐ Chưa rõ ràng |
| Báo cáo | ⭐⭐ Còn thiếu |
| Tính nhất quán | ⭐⭐ Format số/ngày chưa thống nhất |

**Tổng thể:** BRD v1 là nền tảng tốt, mô tả chi tiết ở mức field cho các màn hình chính. Cần bổ sung phần luồng xử lý ngoại lệ, master data, phân quyền, và tích hợp để hoàn thiện.
