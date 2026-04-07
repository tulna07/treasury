# BRD v3 Changelog — Tổng hợp phản hồi từ nghiệp vụ

## Nguồn phản hồi
- **Chị Linh Phương (K.NV&ĐCTC)** — 48 comment trên PDF + 3 phần chat
- **Chị Linh Trương (BA)** — 17 câu hỏi review
- **Chị Nhàn Nguyễn** — Upload ticket + Master data đối tác

---

## 1. THAY ĐỔI CHỨC DANH (toàn bộ tài liệu)

| Gốc | Sửa thành |
|-----|-----------|
| CV/CVKD — Chuyên viên Kinh doanh | **Chuyên viên Kinh doanh / Chuyên viên Hỗ trợ giao dịch** |
| GĐ/PGĐ K.NV | **GĐ/PGĐ Trung tâm KDV/QLV; GĐ/PGĐ Khối** |
| TP K.NV | **TP Kinh doanh ngoại tệ** |

→ Cập nhật tất cả luồng phê duyệt, ma trận phân quyền, danh sách trạng thái, audit trail

## 2. QUY ƯỚC ĐỊNH DẠNG

- **Bond (GTCG):** Dấu "." ngăn nghìn, dấu "," ngăn thập phân (chuẩn kế toán VN)
- **FX, MM:** Dấu "," ngăn nghìn, dấu "." ngăn thập phân
- Thống nhất đơn vị tiền: **VND** (bỏ VNĐ)

## 3. MODULE FX — BỔ SUNG

- Bổ sung trường **"Ngày giao dịch"** cho cả Spot/Forward và Swap (bắt buộc, mặc định TODAY)
- Swap: Sửa tên trường → "Ngày Giá trị chân 1/2" + "Ngày thực hiện chân 1/2" (thống nhất)
- Bổ sung trường **attach file ticket/hợp đồng** trên tất cả màn hình nhập liệu
- Bổ sung **công thức cặp tiền chéo** (EUR/GBP, EUR/JPY...) — không có USD
- Popup **xác nhận khi từ chối** GD (tránh click nhầm) — áp dụng tất cả module
- Phase 2: Tự động sinh **SWIFT MT300** sau khi FX được duyệt
- Phase 2: Luồng **"Redeal"** khi lỗi sinh SWIFT tự động

## 4. MODULE GTCG — SỬA

- **Bỏ trường "Loại giao dịch (phụ)"** trong FI Bond/CCTG — trùng với "Loại GTCG"
- **Block cứng** khi bán vượt tồn kho GTCG
- Sẽ bổ sung list tồn kho GTCG + trường thông tin (chờ chị LP)

## 5. MODULE MM — TÁCH 3 LUỒNG

### 5.1. Giao dịch Liên ngân hàng (giữ nguyên từ BRD v2, có sửa)
- Bổ sung **hình thức "Vay/Cho vay"** ngoài "Gửi tiền/Nhận tiền gửi"
- Trường "Chỉ dẫn TT đối tác": "Tự động" → **"Bắt buộc"** (mặc định tự động, cho phép nhập tay)
- Trường "Số Ticket": "Bắt buộc" → **"Không bắt buộc"**
- Bổ sung trường **"Số tiền lãi"** (phục vụ SWIFT MT320)
- Hỗ trợ **3 day count convention**: Actual/365, Actual/360, Actual/Actual
- Format lại công thức MM cho chuẩn
- Phase 2: Tự động sinh **SWIFT MT320** sau khi MM được duyệt

### 5.2. Giao dịch OMO (MỚI)
- Luồng: CV → TP → GĐ → KTTC (2 cấp) → Hoàn thành
- KHÔNG qua QLRR, KHÔNG qua TTQT, KHÔNG có thanh toán trong nước
- KHÔNG chiếm hạn mức
- Đối tác cố định: Sở giao dịch NHNN
- Trường thông tin: Phiên GD, Ngày GD, Đối tác (tự động), Mã TP, TCPH, LS coupon, Ngày đáo hạn, LS trúng thầu, Kỳ hạn, Ngày TT 1, Ngày TT 2, Hair cut, Attach file
- Bỏ trường "Loại giao dịch" (đã tách luồng riêng)

### 5.3. Giao dịch Repo KBNN (MỚI)
- Luồng phê duyệt **giống OMO**: CV → TP → GĐ → KTTC (2 cấp) → Hoàn thành
- KHÔNG qua QLRR, KHÔNG qua TTQT
- KHÔNG chiếm hạn mức
- Trường thông tin giống OMO

## 6. LUỒNG PHÊ DUYỆT — THAY ĐỔI

- **Bổ sung Lãnh đạo P.KTTC** phê duyệt cho TẤT CẢ nghiệp vụ (2 cấp KTTC)
- Luồng hủy: thêm **duyệt cấp 1 (TP/PP)** trước GĐ
- Hủy GD có TTQT: email thông báo **cả TTQT** (không chỉ KTTC)
- Hủy áp dụng cả trạng thái **"Chờ TTQT"** (không chỉ "Hoàn thành")
- Clone GD: bổ sung khi **KTTC từ chối** (không chỉ GĐ từ chối)

## 7. HẠN MỨC — SỬA

- FX **có thể chiếm hạn mức** (VD: ABBank — FX + MM dùng chung)
- Tỷ giá quy đổi VND: **(Tỷ giá mua + bán chuyển khoản) / 2** — ban hành cuối ngày làm việc liền trước
- Hạn mức đã sử dụng đầu ngày (Không TSBĐ): MM đáo hạn **SAU** ngày GD (không bao gồm đáo TẠI ngày GD)
- Hạn mức chỉ nhập kết quả đã duyệt (không build luồng phê duyệt riêng)

## 8. MASTER DATA

- SWIFT code: **Không bắt buộc** (MM/Bond có đối tác không có SWIFT)
- Mã đối tác = mã nội bộ K.NV (không phải BIC code Core)
- Danh sách đối tác quản lý bằng Excel riêng

## 9. PHÂN QUYỀN

- Thêm role **"GĐ Trung tâm KDV/QLV"**
- Thêm role **"Lãnh đạo P.KTTC"**

## 10. CÁC CÂU TRẢ LỜI ĐÃ CHỐT

- Chữ ký số CA: KHÔNG. Phê duyệt User + Audit trail
- Tolerance tỷ giá FX: KHÔNG cần
- TSBĐ: Ghi nhận text (không tích hợp phong tỏa)
- Format lưu trữ: PDF export
- Citad: KHÔNG đi qua Treasury (tự động trên Core)
- Upload ticket: 1 file = 1 GD

## 11. CHỜ BỔ SUNG

- ☐ File ticket mẫu + mapping trường
- ☐ Danh sách báo cáo Phase 1
- ☐ List tồn kho GTCG + trường thông tin
