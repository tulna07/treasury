# Thu thập yêu cầu Treasury System — Phase 1

**Ngày:** 01/04/2026
**Nguồn:** Trao đổi group "1. Treasury Project"
**Người cung cấp:** Chị Linh Phương (K.NV&ĐCTC), Anh Hưng (VuDuy Hung)
**Chỉ đạo:** Anh Minh Nguyen (Phó TGĐ)

---

## I. SCOPE PHASE 1

- **Mục tiêu:** Số hóa toàn bộ giao dịch Treasury đang thủ công
- **KHÔNG bao gồm:** Tích hợp Flexcube, tự động đẩy sang Core (→ Phase 2)

## II. CẤU TRÚC 5 MODULE

1. **Module FX** — Kinh doanh Ngoại tệ (Spot, Forward, Swap)
2. **Module GTCG** — Giấy tờ có giá (Govi Bond, FI Bond, CCTG)
3. **Module MM** — Kinh doanh tiền tệ (Money Market)
4. **Module Hạn mức** — Phê duyệt hạn mức liên ngân hàng
5. **Module TTQT** — Thanh toán quốc tế

## III. LUỒNG HỦY / SỬA / RECALL (ĐÃ CHỐT)

| Tình huống | Cho phép? | Chi tiết |
|------------|-----------|----------|
| CV recall khi "Chờ K.NV duyệt" | ✅ Có | Về "Open" để sửa, bắt buộc nhập lý do |
| TP recall khi đã duyệt, GĐ chưa duyệt | ✅ Có | Về bước TP review lại, bắt buộc nhập lý do |
| GD bị từ chối → Clone | ✅ Có | CV clone GD cũ để tạo mới, không nhập lại từ đầu |
| Hủy GD đã "Hoàn thành" | ✅ Có | Cần GĐ/PGĐ K.NV duyệt + auto notify P.KTTC bằng email |
| GD bị hủy hiển thị | ✅ Có | Vẫn hiển thị đầy đủ, filter ẩn/hiện, không xóa |

## IV. PHÂN QUYỀN (ĐÃ CHỐT)

### Danh sách Role (8 role — đã đủ)

| Role | Mô tả | Phạm vi dữ liệu |
|------|--------|-----------------|
| CV K.NV | Chuyên viên kinh doanh — tạo GD | Thấy toàn bộ GD |
| TP K.NV | Trưởng phòng — duyệt cấp 1 | Thấy toàn bộ GD |
| GĐ/PGĐ K.NV | Giám đốc/Phó GĐ Khối — duyệt cấp 2 + duyệt hủy | Thấy toàn bộ GD |
| CV QLRR | Chuyên viên QLRR thị trường — duyệt hạn mức cấp 1 | Chỉ thấy GD MM cần duyệt hạn mức |
| TPB QLRR | Trưởng phòng ban QLRR — duyệt hạn mức cấp 2 | Chỉ thấy GD MM cần duyệt hạn mức |
| P.KTTC | Kế toán Tài chính — hạch toán | Chỉ thấy GD chuyển sang bước hạch toán trở đi |
| BP.TTQT | Bộ phận TTQT — duyệt chuyển tiền QT | Chỉ thấy GD TTQT đến hạn |
| Admin | Quản trị hệ thống | Quản lý master data, phân quyền user |

### Ủy quyền
- Không cần chức năng ủy quyền trên hệ thống
- Người được ủy quyền có account login trực tiếp

## V. NOTIFICATION (ĐÃ CHỐT)

| Loại | Kênh |
|------|------|
| Chuyển trạng thái GD thông thường | In-app |
| Hủy GD đã hạch toán | In-app + Email |

## VI. MASTER DATA (ĐÃ CHỐT)

### Danh sách đối tác
- Hiện quản lý bằng Excel → import vào hệ thống
- Quyền thêm/sửa: user được phân quyền Admin
- Thông tin: Mã viết tắt, Tên đầy đủ, CIF, SWIFT code, Pay code/SSI

### Danh mục trái phiếu (Govi Bond)
- Source: K.NV tự quản lý
- Cập nhật: khi có phát hành mới hoặc thay đổi thông tin

### Pay code / SSI
- 1 đối tác có thể có nhiều Pay code, kể cả cùng 1 loại tiền
- Quản lý bởi K.NV
- Nhập ban đầu 1 lần, cập nhật khi có thay đổi

## VII. FORMAT THỐNG NHẤT (ĐÃ CHỐT)

| Loại | Format |
|------|--------|
| Số (tiền, khối lượng) | Dấu "," ngăn nghìn, "." thập phân (ví dụ: 10,000,000.05) |
| Ngày | dd/mm/yyyy |
| Lãi suất | Dấu "." thập phân, đơn vị %/năm (ví dụ: 4.50%/năm) |

## VIII. CÔNG THỨC ĐÃ XÁC NHẬN

### FX — Thành tiền
- Cặp USD/VND: Khối lượng × Tỷ giá → VND
- Cặp USD/...: Khối lượng / Tỷ giá → USD
- Cặp .../USD: Khối lượng × Tỷ giá → USD

### FX SWAP — Thành tiền chân 2
- Dùng **Tỷ giá chân 2** (BRD v1 ghi "chân 1" là typo)

### GTCG — Tổng giá trị giao dịch
- = Số lượng × Giá thanh toán

### GTCG — Kỳ hạn còn lại
- = Ngày đáo hạn − Ngày thanh toán (đơn vị: ngày)

### MM — Số tiền tại ngày đáo hạn
- = Số tiền gốc + (Số tiền gốc × Lãi suất/100 × Kỳ hạn / 365)

## IX. LOGIC ĐẶC BIỆT (ĐÃ CHỐT)

### SWAP chân 2 & TTQT
- Chân 2 tự động xuất hiện trên danh sách TTQT vào đúng ngày thực hiện
- Không cần thao tác thêm

### MM đáo hạn
- Chỉ theo dõi trên danh sách, không tự động tạo GD nhận tiền mới
- Hiển thị theo ngày đáo hạn

### Hạch toán GTCG
- Chỉ khi chiều GD = "Bên mua" → chọn loại hạch toán (HTM/AFS/HFT)
- Chiều GD = "Bên bán" → không cần chọn

## X. BÁO CÁO

- Phase 1 sẽ có báo cáo (không chỉ export Excel)
- Báo cáo tổng hợp hạn mức trong ngày (đã có trong BRD v1)
- Danh sách báo cáo cụ thể: **chị Linh Phương sẽ cung cấp sau**

## XI. YÊU CẦU PHI CHỨC NĂNG

- Hệ thống chạy trong **mạng nội bộ** KienlongBank
- Lưu trữ **vĩnh viễn**, backup **1 lần/tuần**
- Hỗ trợ **nhiều user đồng thời**
- **Audit trail:** log toàn bộ hành động (xem chi tiết phần riêng)

## XII. AUDIT TRAIL (ĐÃ ĐỀ XUẤT, CHỜ CONFIRM)

- Mọi hành động trên GD phải ghi log (tạo/sửa/duyệt/từ chối/hủy/recall)
- Thông tin log: User ID, họ tên, đơn vị, timestamp, trạng thái trước→sau, giá trị cũ→mới, lý do
- Snapshot hạn mức tại thời điểm QLRR duyệt
- Append-only, không cho phép sửa/xóa log
- Mỗi GD có tab "Lịch sử" hiển thị timeline
- Quyền xem: TP trở lên + QLRR + KTTC

---

## CÒN CHỜ

1. ☐ Danh sách báo cáo cụ thể Phase 1 (chị Linh Phương)
2. ☐ Confirm chi tiết Audit Trail
3. ☐ Review & bổ sung nghiệp vụ nếu phát sinh
4. ☐ Luồng hủy chi tiết (chị Linh Phương đang hoàn thiện)

---

*Tài liệu này làm đầu vào để KAI viết BRD v2 hoàn chỉnh.*
