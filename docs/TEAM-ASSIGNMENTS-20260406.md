# Phân công công việc — Treasury Project (06/04/2026)

## Bối cảnh
- Dev phase 1 hoàn tất (BRD chưa final)
- Chuyển sang giai đoạn review, hoàn thiện và triển khai nội bộ

## 1. Nhóm Nghiệp vụ + BA
- Review BRD v3 — đánh dấu chỗ cần bổ sung/chỉnh sửa
- Review 303 Test Cases — kiểm tra đủ case, đúng nghiệp vụ
- Không tự soạn tài liệu — mô tả yêu cầu bằng text, tag KAI để soạn
- Khi có thay đổi: lập danh sách rõ ràng (page, field, logic) → gửi lên group

## 2. Dev (Anh Bay)
- Pull code từ GitLab: git.kiloba.ai/treasury/app
- Dev bằng Claude Code ở local
- Ràng buộc: code khớp BRD + Test Cases, build pass trước khi push
- Sync code lên git thường xuyên → KAI pull về review + E2E

## 3. DevOps
- Deploy lên mạng nội bộ ngân hàng
- Stack: PostgreSQL + Zitadel + Treasury API (Go) + Treasury Web (Next.js)

## 4. Nguyên tắc AI First (toàn team)
- Không tự soạn tài liệu — mô tả → KAI soạn
- Không tự viết code từ đầu — dùng Claude Code
- Luôn đặt ràng buộc để verify:
  - Nghiệp vụ: yêu cầu map vào BRD section
  - Dev: code pass build + match test case
  - BA: test case cover hết BRD

## Nguồn
Chỉ đạo từ anh Minh Nguyen (@minhngvan) — 06/04/2026
