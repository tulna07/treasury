-- ============================================================================
-- 001_seed.sql — Base seed data for Treasury Management System
-- ============================================================================

-- ===========================================================================
-- Branch
-- ===========================================================================
INSERT INTO branches (id, code, name, branch_type) VALUES
    ('a0000000-0000-0000-0000-000000000001', 'HO', 'Hội sở chính - Head Office', 'HEAD_OFFICE');

-- ===========================================================================
-- Roles (10 roles per BRD v3)
-- ===========================================================================
INSERT INTO roles (id, code, name, description, scope) VALUES
    ('b0000000-0000-0000-0000-000000000001', 'DEALER', 'Dealer - Nhân viên Kinh doanh', 'Tạo, sửa, xóa deal; recall, clone', 'ALL'),
    ('b0000000-0000-0000-0000-000000000002', 'DESK_HEAD', 'Trưởng phòng Kinh doanh (TP)', 'Phê duyệt L1, return deal', 'MODULE_SPECIFIC'),
    ('b0000000-0000-0000-0000-000000000003', 'CENTER_DIRECTOR', 'Giám đốc Trung tâm (GĐ)', 'Phê duyệt L2, reject deal', 'MODULE_SPECIFIC'),
    ('b0000000-0000-0000-0000-000000000004', 'DIVISION_HEAD', 'Giám đốc Khối', 'Phê duyệt L2 cho deals vượt hạn mức', 'MODULE_SPECIFIC'),
    ('b0000000-0000-0000-0000-000000000005', 'RISK_OFFICER', 'Nhân viên QLRR', 'Kiểm tra rủi ro L1', 'STEP_SPECIFIC'),
    ('b0000000-0000-0000-0000-000000000006', 'RISK_HEAD', 'Trưởng phòng QLRR', 'Kiểm tra rủi ro L2', 'STEP_SPECIFIC'),
    ('b0000000-0000-0000-0000-000000000007', 'ACCOUNTANT', 'Kế toán viên', 'Hạch toán giao dịch', 'STEP_SPECIFIC'),
    ('b0000000-0000-0000-0000-000000000008', 'CHIEF_ACCOUNTANT', 'Kế toán trưởng', 'Phê duyệt hạch toán, void', 'STEP_SPECIFIC'),
    ('b0000000-0000-0000-0000-000000000009', 'SETTLEMENT_OFFICER', 'Nhân viên TTQT', 'Thanh toán quốc tế, settle', 'STEP_SPECIFIC'),
    ('b0000000-0000-0000-0000-000000000010', 'ADMIN', 'Quản trị hệ thống', 'Quản lý users, cấu hình, audit log', 'ALL');

-- ===========================================================================
-- Permissions — FX Deal
-- ===========================================================================
INSERT INTO permissions (id, code, resource, action, description) VALUES
    ('c0000000-0000-0000-0000-000000000001', 'FX_DEAL.VIEW', 'FX_DEAL', 'VIEW', 'Xem danh sách và chi tiết giao dịch FX'),
    ('c0000000-0000-0000-0000-000000000002', 'FX_DEAL.CREATE', 'FX_DEAL', 'CREATE', 'Tạo giao dịch FX mới'),
    ('c0000000-0000-0000-0000-000000000003', 'FX_DEAL.EDIT', 'FX_DEAL', 'EDIT', 'Sửa giao dịch FX (chỉ khi OPEN)'),
    ('c0000000-0000-0000-0000-000000000004', 'FX_DEAL.APPROVE_L1', 'FX_DEAL', 'APPROVE_L1', 'Phê duyệt FX cấp 1 (Trưởng phòng)'),
    ('c0000000-0000-0000-0000-000000000005', 'FX_DEAL.APPROVE_L2', 'FX_DEAL', 'APPROVE_L2', 'Phê duyệt FX cấp 2 (Giám đốc)'),
    ('c0000000-0000-0000-0000-000000000006', 'FX_DEAL.RECALL', 'FX_DEAL', 'RECALL', 'Thu hồi giao dịch FX về OPEN'),
    ('c0000000-0000-0000-0000-000000000007', 'FX_DEAL.CLONE', 'FX_DEAL', 'CLONE', 'Sao chép giao dịch FX bị từ chối'),
    ('c0000000-0000-0000-0000-000000000008', 'FX_DEAL.CANCEL_REQUEST', 'FX_DEAL', 'CANCEL_REQUEST', 'Yêu cầu hủy giao dịch FX'),
    ('c0000000-0000-0000-0000-000000000009', 'FX_DEAL.CANCEL_APPROVE_L1', 'FX_DEAL', 'CANCEL_APPROVE_L1', 'Phê duyệt hủy FX cấp 1'),
    ('c0000000-0000-0000-0000-000000000010', 'FX_DEAL.CANCEL_APPROVE_L2', 'FX_DEAL', 'CANCEL_APPROVE_L2', 'Phê duyệt hủy FX cấp 2'),
    ('c0000000-0000-0000-0000-000000000030', 'FX_DEAL.DELETE', 'FX_DEAL', 'DELETE', 'Xóa giao dịch FX'),
    ('c0000000-0000-0000-0000-000000000031', 'FX_DEAL.BOOK_L1', 'FX_DEAL', 'BOOK_L1', 'Hạch toán FX cấp 1 (Kế toán viên)'),
    ('c0000000-0000-0000-0000-000000000032', 'FX_DEAL.BOOK_L2', 'FX_DEAL', 'BOOK_L2', 'Hạch toán FX cấp 2 (Kế toán trưởng)'),
    ('c0000000-0000-0000-0000-000000000033', 'FX_DEAL.SETTLE', 'FX_DEAL', 'SETTLE', 'Thanh toán FX (Nhân viên TTQT)'),
    ('c0000000-0000-0000-0000-000000000034', 'FX_DEAL.EXPORT', 'FX_DEAL', 'EXPORT', 'Xuất dữ liệu FX');

-- Permissions — Bond Deal
INSERT INTO permissions (id, code, resource, action, description) VALUES
    ('c0000000-0000-0000-0000-000000000011', 'BOND_DEAL.VIEW', 'BOND_DEAL', 'VIEW', 'Xem giao dịch GTCG'),
    ('c0000000-0000-0000-0000-000000000012', 'BOND_DEAL.CREATE', 'BOND_DEAL', 'CREATE', 'Tạo giao dịch GTCG'),
    ('c0000000-0000-0000-0000-000000000013', 'BOND_DEAL.EDIT', 'BOND_DEAL', 'EDIT', 'Sửa giao dịch GTCG'),
    ('c0000000-0000-0000-0000-000000000014', 'BOND_DEAL.APPROVE_L1', 'BOND_DEAL', 'APPROVE_L1', 'Phê duyệt GTCG cấp 1'),
    ('c0000000-0000-0000-0000-000000000015', 'BOND_DEAL.APPROVE_L2', 'BOND_DEAL', 'APPROVE_L2', 'Phê duyệt GTCG cấp 2');

-- Permissions — MM Deal
INSERT INTO permissions (id, code, resource, action, description) VALUES
    ('c0000000-0000-0000-0000-000000000016', 'MM_INTERBANK_DEAL.VIEW', 'MM_INTERBANK_DEAL', 'VIEW', 'Xem giao dịch liên ngân hàng'),
    ('c0000000-0000-0000-0000-000000000017', 'MM_INTERBANK_DEAL.CREATE', 'MM_INTERBANK_DEAL', 'CREATE', 'Tạo giao dịch liên ngân hàng'),
    ('c0000000-0000-0000-0000-000000000018', 'MM_INTERBANK_DEAL.EDIT', 'MM_INTERBANK_DEAL', 'EDIT', 'Sửa giao dịch liên ngân hàng'),
    ('c0000000-0000-0000-0000-000000000019', 'MM_INTERBANK_DEAL.APPROVE_L1', 'MM_INTERBANK_DEAL', 'APPROVE_L1', 'Phê duyệt liên ngân hàng cấp 1');

-- Permissions — Credit Limit
INSERT INTO permissions (id, code, resource, action, description) VALUES
    ('c0000000-0000-0000-0000-000000000020', 'CREDIT_LIMIT.VIEW', 'CREDIT_LIMIT', 'VIEW', 'Xem hạn mức'),
    ('c0000000-0000-0000-0000-000000000021', 'CREDIT_LIMIT.CREATE', 'CREDIT_LIMIT', 'CREATE', 'Tạo hạn mức'),
    ('c0000000-0000-0000-0000-000000000022', 'CREDIT_LIMIT.APPROVE_L1', 'CREDIT_LIMIT', 'APPROVE_L1', 'Phê duyệt hạn mức');

-- Permissions — Settlement
INSERT INTO permissions (id, code, resource, action, description) VALUES
    ('c0000000-0000-0000-0000-000000000023', 'INTERNATIONAL_PAYMENT.VIEW', 'INTERNATIONAL_PAYMENT', 'VIEW', 'Xem thanh toán quốc tế'),
    ('c0000000-0000-0000-0000-000000000024', 'INTERNATIONAL_PAYMENT.CREATE', 'INTERNATIONAL_PAYMENT', 'CREATE', 'Tạo lệnh thanh toán'),
    ('c0000000-0000-0000-0000-000000000025', 'INTERNATIONAL_PAYMENT.SETTLE', 'INTERNATIONAL_PAYMENT', 'SETTLE', 'Thực hiện thanh toán');

-- Permissions — System / Admin
INSERT INTO permissions (id, code, resource, action, description) VALUES
    ('c0000000-0000-0000-0000-000000000026', 'SYSTEM.MANAGE', 'SYSTEM', 'MANAGE', 'Quản trị hệ thống');

-- ===========================================================================
-- Role-Permission Mapping
-- ===========================================================================

-- DEALER: FX create/view/edit/recall/clone/delete + Bond/MM create/view/edit
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('b0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000001'), -- FX VIEW
    ('b0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000002'), -- FX CREATE
    ('b0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000003'), -- FX EDIT
    ('b0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000030'), -- FX DELETE
    ('b0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000006'), -- FX RECALL
    ('b0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000007'), -- FX CLONE
    ('b0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000008'), -- FX CANCEL_REQUEST
    ('b0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000011'), -- BOND VIEW
    ('b0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000012'), -- BOND CREATE
    ('b0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000013'), -- BOND EDIT
    ('b0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000016'), -- MM VIEW
    ('b0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000017'), -- MM CREATE
    ('b0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000018'), -- MM EDIT
    ('b0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000034'); -- FX EXPORT

-- DESK_HEAD: FX view/approve_l1/cancel_approve_l1 + Bond/MM view/approve_l1
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('b0000000-0000-0000-0000-000000000002', 'c0000000-0000-0000-0000-000000000001'), -- FX VIEW
    ('b0000000-0000-0000-0000-000000000002', 'c0000000-0000-0000-0000-000000000004'), -- FX APPROVE_L1
    ('b0000000-0000-0000-0000-000000000002', 'c0000000-0000-0000-0000-000000000009'), -- FX CANCEL_APPROVE_L1
    ('b0000000-0000-0000-0000-000000000002', 'c0000000-0000-0000-0000-000000000011'), -- BOND VIEW
    ('b0000000-0000-0000-0000-000000000002', 'c0000000-0000-0000-0000-000000000014'), -- BOND APPROVE_L1
    ('b0000000-0000-0000-0000-000000000002', 'c0000000-0000-0000-0000-000000000016'), -- MM VIEW
    ('b0000000-0000-0000-0000-000000000002', 'c0000000-0000-0000-0000-000000000019'), -- MM APPROVE_L1
    ('b0000000-0000-0000-0000-000000000002', 'c0000000-0000-0000-0000-000000000020'), -- LIMIT VIEW
    ('b0000000-0000-0000-0000-000000000002', 'c0000000-0000-0000-0000-000000000034'); -- FX EXPORT

-- CENTER_DIRECTOR: FX view/approve_l2/cancel_approve_l2 + Bond/MM view/approve_l2 + Limit
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('b0000000-0000-0000-0000-000000000003', 'c0000000-0000-0000-0000-000000000001'), -- FX VIEW
    ('b0000000-0000-0000-0000-000000000003', 'c0000000-0000-0000-0000-000000000005'), -- FX APPROVE_L2
    ('b0000000-0000-0000-0000-000000000003', 'c0000000-0000-0000-0000-000000000010'), -- FX CANCEL_APPROVE_L2
    ('b0000000-0000-0000-0000-000000000003', 'c0000000-0000-0000-0000-000000000011'), -- BOND VIEW
    ('b0000000-0000-0000-0000-000000000003', 'c0000000-0000-0000-0000-000000000015'), -- BOND APPROVE_L2
    ('b0000000-0000-0000-0000-000000000003', 'c0000000-0000-0000-0000-000000000016'), -- MM VIEW
    ('b0000000-0000-0000-0000-000000000003', 'c0000000-0000-0000-0000-000000000020'), -- LIMIT VIEW
    ('b0000000-0000-0000-0000-000000000003', 'c0000000-0000-0000-0000-000000000022'), -- LIMIT APPROVE
    ('b0000000-0000-0000-0000-000000000003', 'c0000000-0000-0000-0000-000000000034'); -- FX EXPORT

-- DIVISION_HEAD: same as CENTER_DIRECTOR + LIMIT CREATE
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('b0000000-0000-0000-0000-000000000004', 'c0000000-0000-0000-0000-000000000001'), -- FX VIEW
    ('b0000000-0000-0000-0000-000000000004', 'c0000000-0000-0000-0000-000000000005'), -- FX APPROVE_L2
    ('b0000000-0000-0000-0000-000000000004', 'c0000000-0000-0000-0000-000000000010'), -- FX CANCEL_APPROVE_L2
    ('b0000000-0000-0000-0000-000000000004', 'c0000000-0000-0000-0000-000000000011'), -- BOND VIEW
    ('b0000000-0000-0000-0000-000000000004', 'c0000000-0000-0000-0000-000000000015'), -- BOND APPROVE_L2
    ('b0000000-0000-0000-0000-000000000004', 'c0000000-0000-0000-0000-000000000016'), -- MM VIEW
    ('b0000000-0000-0000-0000-000000000004', 'c0000000-0000-0000-0000-000000000020'), -- LIMIT VIEW
    ('b0000000-0000-0000-0000-000000000004', 'c0000000-0000-0000-0000-000000000021'), -- LIMIT CREATE
    ('b0000000-0000-0000-0000-000000000004', 'c0000000-0000-0000-0000-000000000022'), -- LIMIT APPROVE
    ('b0000000-0000-0000-0000-000000000004', 'c0000000-0000-0000-0000-000000000034'); -- FX EXPORT

-- ACCOUNTANT: FX/Bond/MM view + booking L1
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('b0000000-0000-0000-0000-000000000007', 'c0000000-0000-0000-0000-000000000001'), -- FX VIEW
    ('b0000000-0000-0000-0000-000000000007', 'c0000000-0000-0000-0000-000000000031'), -- FX BOOK_L1
    ('b0000000-0000-0000-0000-000000000007', 'c0000000-0000-0000-0000-000000000011'), -- BOND VIEW
    ('b0000000-0000-0000-0000-000000000007', 'c0000000-0000-0000-0000-000000000016'); -- MM VIEW

-- CHIEF_ACCOUNTANT: same as ACCOUNTANT + booking L2
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('b0000000-0000-0000-0000-000000000008', 'c0000000-0000-0000-0000-000000000001'), -- FX VIEW
    ('b0000000-0000-0000-0000-000000000008', 'c0000000-0000-0000-0000-000000000032'), -- FX BOOK_L2
    ('b0000000-0000-0000-0000-000000000008', 'c0000000-0000-0000-0000-000000000011'), -- BOND VIEW
    ('b0000000-0000-0000-0000-000000000008', 'c0000000-0000-0000-0000-000000000016'); -- MM VIEW

-- SETTLEMENT_OFFICER: FX/Bond/MM view + Settlement
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('b0000000-0000-0000-0000-000000000009', 'c0000000-0000-0000-0000-000000000001'), -- FX VIEW
    ('b0000000-0000-0000-0000-000000000009', 'c0000000-0000-0000-0000-000000000033'), -- FX SETTLE
    ('b0000000-0000-0000-0000-000000000009', 'c0000000-0000-0000-0000-000000000011'), -- BOND VIEW
    ('b0000000-0000-0000-0000-000000000009', 'c0000000-0000-0000-0000-000000000016'), -- MM VIEW
    ('b0000000-0000-0000-0000-000000000009', 'c0000000-0000-0000-0000-000000000023'), -- SETTLEMENT VIEW
    ('b0000000-0000-0000-0000-000000000009', 'c0000000-0000-0000-0000-000000000024'), -- SETTLEMENT CREATE
    ('b0000000-0000-0000-0000-000000000009', 'c0000000-0000-0000-0000-000000000025'); -- SETTLEMENT SETTLE

-- RISK_OFFICER + RISK_HEAD: MM/Limit view only, NO FX VIEW
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('b0000000-0000-0000-0000-000000000005', 'c0000000-0000-0000-0000-000000000011'), -- BOND VIEW
    ('b0000000-0000-0000-0000-000000000005', 'c0000000-0000-0000-0000-000000000016'), -- MM VIEW
    ('b0000000-0000-0000-0000-000000000005', 'c0000000-0000-0000-0000-000000000020'); -- LIMIT VIEW

INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('b0000000-0000-0000-0000-000000000006', 'c0000000-0000-0000-0000-000000000011'), -- BOND VIEW
    ('b0000000-0000-0000-0000-000000000006', 'c0000000-0000-0000-0000-000000000016'), -- MM VIEW
    ('b0000000-0000-0000-0000-000000000006', 'c0000000-0000-0000-0000-000000000020'); -- LIMIT VIEW

-- ADMIN: System manage + all views
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('b0000000-0000-0000-0000-000000000010', 'c0000000-0000-0000-0000-000000000001'), -- FX VIEW
    ('b0000000-0000-0000-0000-000000000010', 'c0000000-0000-0000-0000-000000000011'), -- BOND VIEW
    ('b0000000-0000-0000-0000-000000000010', 'c0000000-0000-0000-0000-000000000016'), -- MM VIEW
    ('b0000000-0000-0000-0000-000000000010', 'c0000000-0000-0000-0000-000000000020'), -- LIMIT VIEW
    ('b0000000-0000-0000-0000-000000000010', 'c0000000-0000-0000-0000-000000000023'), -- SETTLEMENT VIEW
    ('b0000000-0000-0000-0000-000000000010', 'c0000000-0000-0000-0000-000000000026'), -- SYSTEM MANAGE
    ('b0000000-0000-0000-0000-000000000010', 'c0000000-0000-0000-0000-000000000034'); -- FX EXPORT

-- ===========================================================================
-- Users (5 test users)
-- ===========================================================================
-- Password: Treasury@2026 (bcrypt hash, cost=10)
INSERT INTO users (id, username, full_name, email, branch_id, department, position, password_hash) VALUES
    ('d0000000-0000-0000-0000-000000000001', 'dealer01', 'Nguyễn Văn An', 'an.nv@kienlongbank.com', 'a0000000-0000-0000-0000-000000000001', 'K.NV', 'Nhân viên Kinh doanh', '$2a$12$pE4SAyzW1QGpEpaWA4hpWOLqF/fmDb3DPK/pLvkzo1XYvW4RTP9Zu'),
    ('d0000000-0000-0000-0000-000000000002', 'deskhead01', 'Trần Thị Bình', 'binh.tt@kienlongbank.com', 'a0000000-0000-0000-0000-000000000001', 'K.NV', 'Trưởng phòng Kinh doanh', '$2a$12$pE4SAyzW1QGpEpaWA4hpWOLqF/fmDb3DPK/pLvkzo1XYvW4RTP9Zu'),
    ('d0000000-0000-0000-0000-000000000003', 'director01', 'Lê Minh Cường', 'cuong.lm@kienlongbank.com', 'a0000000-0000-0000-0000-000000000001', 'K.NV', 'Giám đốc Trung tâm KDNT', '$2a$12$pE4SAyzW1QGpEpaWA4hpWOLqF/fmDb3DPK/pLvkzo1XYvW4RTP9Zu'),
    ('d0000000-0000-0000-0000-000000000004', 'accountant01', 'Phạm Thị Dung', 'dung.pt@kienlongbank.com', 'a0000000-0000-0000-0000-000000000001', 'P.KTTC', 'Kế toán viên', '$2a$12$pE4SAyzW1QGpEpaWA4hpWOLqF/fmDb3DPK/pLvkzo1XYvW4RTP9Zu'),
    ('d0000000-0000-0000-0000-000000000005', 'settlement01', 'Hoàng Văn Em', 'em.hv@kienlongbank.com', 'a0000000-0000-0000-0000-000000000001', 'BP.TTQT', 'Nhân viên TTQT', '$2a$12$pE4SAyzW1QGpEpaWA4hpWOLqF/fmDb3DPK/pLvkzo1XYvW4RTP9Zu');

-- Additional 5 users (total 10 for all roles)
-- Password: P@ssw0rd123 (bcrypt hash, cost=10)
INSERT INTO users (id, username, full_name, email, branch_id, department, position, password_hash) VALUES
    ('d0000000-0000-0000-0000-000000000006', 'risk01', 'Ngô Thị Phương', 'phuong.nt@kienlongbank.com', 'a0000000-0000-0000-0000-000000000001', 'P.QLRR', 'Nhân viên QLRR', '$2a$12$pE4SAyzW1QGpEpaWA4hpWOLqF/fmDb3DPK/pLvkzo1XYvW4RTP9Zu'),
    ('d0000000-0000-0000-0000-000000000007', 'riskhead01', 'Đỗ Văn Giang', 'giang.dv@kienlongbank.com', 'a0000000-0000-0000-0000-000000000001', 'P.QLRR', 'Trưởng phòng QLRR', '$2a$12$pE4SAyzW1QGpEpaWA4hpWOLqF/fmDb3DPK/pLvkzo1XYvW4RTP9Zu'),
    ('d0000000-0000-0000-0000-000000000008', 'divhead01', 'Vũ Hoàng Hải', 'hai.vh@kienlongbank.com', 'a0000000-0000-0000-0000-000000000001', 'K.NV', 'Giám đốc Khối', '$2a$12$pE4SAyzW1QGpEpaWA4hpWOLqF/fmDb3DPK/pLvkzo1XYvW4RTP9Zu'),
    ('d0000000-0000-0000-0000-000000000009', 'chiefacc01', 'Bùi Thị Khánh', 'khanh.bt@kienlongbank.com', 'a0000000-0000-0000-0000-000000000001', 'P.KTTC', 'Kế toán trưởng', '$2a$12$pE4SAyzW1QGpEpaWA4hpWOLqF/fmDb3DPK/pLvkzo1XYvW4RTP9Zu'),
    ('d0000000-0000-0000-0000-000000000010', 'admin01', 'Nguyễn Văn Minh', 'minh.nv@kienlongbank.com', 'a0000000-0000-0000-0000-000000000001', 'K.CN', 'Quản trị viên hệ thống', '$2a$12$pE4SAyzW1QGpEpaWA4hpWOLqF/fmDb3DPK/pLvkzo1XYvW4RTP9Zu');

-- User-Role assignments
INSERT INTO user_roles (user_id, role_id) VALUES
    ('d0000000-0000-0000-0000-000000000001', 'b0000000-0000-0000-0000-000000000001'), -- dealer01 = DEALER
    ('d0000000-0000-0000-0000-000000000002', 'b0000000-0000-0000-0000-000000000002'), -- deskhead01 = DESK_HEAD
    ('d0000000-0000-0000-0000-000000000003', 'b0000000-0000-0000-0000-000000000003'), -- director01 = CENTER_DIRECTOR
    ('d0000000-0000-0000-0000-000000000004', 'b0000000-0000-0000-0000-000000000007'), -- accountant01 = ACCOUNTANT
    ('d0000000-0000-0000-0000-000000000005', 'b0000000-0000-0000-0000-000000000009'), -- settlement01 = SETTLEMENT_OFFICER
    ('d0000000-0000-0000-0000-000000000006', 'b0000000-0000-0000-0000-000000000005'), -- risk01 = RISK_OFFICER
    ('d0000000-0000-0000-0000-000000000007', 'b0000000-0000-0000-0000-000000000006'), -- riskhead01 = RISK_HEAD
    ('d0000000-0000-0000-0000-000000000008', 'b0000000-0000-0000-0000-000000000004'), -- divhead01 = DIVISION_HEAD
    ('d0000000-0000-0000-0000-000000000009', 'b0000000-0000-0000-0000-000000000008'), -- chiefacc01 = CHIEF_ACCOUNTANT
    ('d0000000-0000-0000-0000-000000000010', 'b0000000-0000-0000-0000-000000000010'); -- admin01 = ADMIN

-- ===========================================================================
-- Counterparties (10 real VN banks)
-- ===========================================================================
INSERT INTO counterparties (id, code, full_name, short_name, cif, swift_code, country_code) VALUES
    ('e0000000-0000-0000-0000-000000000001', 'MSB', 'Ngân hàng TMCP Hàng Hải Việt Nam', 'Maritime Bank', 'CIF-MSB-001', 'MCABORVX', 'VN'),
    ('e0000000-0000-0000-0000-000000000002', 'ACB', 'Ngân hàng TMCP Á Châu', 'ACB', 'CIF-ACB-001', 'ASCBVNVX', 'VN'),
    ('e0000000-0000-0000-0000-000000000003', 'VCB', 'Ngân hàng TMCP Ngoại thương Việt Nam', 'Vietcombank', 'CIF-VCB-001', 'BFTVVNVX', 'VN'),
    ('e0000000-0000-0000-0000-000000000004', 'TCB', 'Ngân hàng TMCP Kỹ thương Việt Nam', 'Techcombank', 'CIF-TCB-001', 'VTCBVNVX', 'VN'),
    ('e0000000-0000-0000-0000-000000000005', 'VPB', 'Ngân hàng TMCP Việt Nam Thịnh Vượng', 'VPBank', 'CIF-VPB-001', 'VPBKVNVX', 'VN'),
    ('e0000000-0000-0000-0000-000000000006', 'MBB', 'Ngân hàng TMCP Quân đội', 'MB Bank', 'CIF-MBB-001', 'MSCBVNVX', 'VN'),
    ('e0000000-0000-0000-0000-000000000007', 'BID', 'Ngân hàng TMCP Đầu tư và Phát triển Việt Nam', 'BIDV', 'CIF-BID-001', 'BIDVVNVX', 'VN'),
    ('e0000000-0000-0000-0000-000000000008', 'CTG', 'Ngân hàng TMCP Công thương Việt Nam', 'VietinBank', 'CIF-CTG-001', 'ICBVVNVX', 'VN'),
    ('e0000000-0000-0000-0000-000000000009', 'STB', 'Ngân hàng TMCP Sài Gòn Thương Tín', 'Sacombank', 'CIF-STB-001', 'SGTTVNVX', 'VN'),
    ('e0000000-0000-0000-0000-000000000010', 'SHB', 'Ngân hàng TMCP Sài Gòn - Hà Nội', 'SHB', 'CIF-SHB-001', 'SHBAVNVX', 'VN');

-- ===========================================================================
-- Currencies (8 major ones)
-- ===========================================================================
INSERT INTO currencies (id, code, numeric_code, name, decimal_places) VALUES
    ('f0000000-0000-0000-0000-000000000001', 'VND', 704, 'Vietnamese Dong', 0),
    ('f0000000-0000-0000-0000-000000000002', 'USD', 840, 'US Dollar', 2),
    ('f0000000-0000-0000-0000-000000000003', 'EUR', 978, 'Euro', 2),
    ('f0000000-0000-0000-0000-000000000004', 'GBP', 826, 'British Pound', 2),
    ('f0000000-0000-0000-0000-000000000005', 'AUD', 036, 'Australian Dollar', 2),
    ('f0000000-0000-0000-0000-000000000006', 'JPY', 392, 'Japanese Yen', 0),
    ('f0000000-0000-0000-0000-000000000007', 'CHF', 756, 'Swiss Franc', 2),
    ('f0000000-0000-0000-0000-000000000008', 'KRW', 410, 'South Korean Won', 0);

-- ===========================================================================
-- Currency Pairs (10 pairs)
-- ===========================================================================
INSERT INTO currency_pairs (id, base_currency, quote_currency, pair_code, rate_decimal_places, calculation_rule, result_currency) VALUES
    ('10000000-0000-0000-0000-000000000001', 'USD', 'VND', 'USD/VND', 2, 'MULTIPLY', 'VND'),
    ('10000000-0000-0000-0000-000000000002', 'EUR', 'USD', 'EUR/USD', 4, 'MULTIPLY', 'USD'),
    ('10000000-0000-0000-0000-000000000003', 'USD', 'JPY', 'USD/JPY', 2, 'MULTIPLY', 'JPY'),
    ('10000000-0000-0000-0000-000000000004', 'AUD', 'USD', 'AUD/USD', 4, 'MULTIPLY', 'USD'),
    ('10000000-0000-0000-0000-000000000005', 'GBP', 'USD', 'GBP/USD', 4, 'MULTIPLY', 'USD'),
    ('10000000-0000-0000-0000-000000000006', 'EUR', 'GBP', 'EUR/GBP', 4, 'DIVIDE', 'GBP'),
    ('10000000-0000-0000-0000-000000000007', 'EUR', 'JPY', 'EUR/JPY', 2, 'MULTIPLY', 'JPY'),
    ('10000000-0000-0000-0000-000000000008', 'USD', 'KRW', 'USD/KRW', 2, 'MULTIPLY', 'KRW'),
    ('10000000-0000-0000-0000-000000000009', 'USD', 'CHF', 'USD/CHF', 4, 'DIVIDE', 'CHF'),
    ('10000000-0000-0000-0000-000000000010', 'AUD', 'JPY', 'AUD/JPY', 2, 'MULTIPLY', 'JPY');

-- ===========================================================================
-- Settlement Instructions (5 for KLB + counterparties)
-- ===========================================================================
-- KLB internal SSI (INTERNAL)
INSERT INTO settlement_instructions (id, counterparty_id, currency_code, owner_type, account_number, bank_name, swift_code, is_default) VALUES
    ('20000000-0000-0000-0000-000000000001', 'e0000000-0000-0000-0000-000000000001', 'USD', 'INTERNAL', '1001-USD-KLB', 'Kienlongbank', 'KLBKVNVX', true),
    ('20000000-0000-0000-0000-000000000002', 'e0000000-0000-0000-0000-000000000001', 'VND', 'INTERNAL', '1001-VND-KLB', 'Kienlongbank', 'KLBKVNVX', true),
    ('20000000-0000-0000-0000-000000000003', 'e0000000-0000-0000-0000-000000000001', 'USD', 'COUNTERPARTY', '2001-USD-MSB', 'Maritime Bank', 'MCABORVX', true),
    ('20000000-0000-0000-0000-000000000004', 'e0000000-0000-0000-0000-000000000002', 'USD', 'COUNTERPARTY', '2001-USD-ACB', 'ACB', 'ASCBVNVX', true),
    ('20000000-0000-0000-0000-000000000005', 'e0000000-0000-0000-0000-000000000003', 'VND', 'COUNTERPARTY', '2001-VND-VCB', 'Vietcombank', 'BFTVVNVX', true);

-- ===========================================================================
-- FX Status Transition Rules (complete state machine)
-- ===========================================================================
INSERT INTO status_transition_rules (deal_module, from_status, to_status, required_role, requires_reason, requires_confirmation) VALUES
    -- Dealer submits → Pending L2 (TP approves L1 implicitly)
    ('FX', 'OPEN', 'PENDING_L2_APPROVAL', 'DESK_HEAD', false, true),
    -- TP returns to dealer
    ('FX', 'OPEN', 'OPEN', 'DESK_HEAD', true, false),
    -- GĐ approves L2 → Pending Booking
    ('FX', 'PENDING_L2_APPROVAL', 'PENDING_BOOKING', 'CENTER_DIRECTOR', false, true),
    -- GĐ rejects → Rejected
    ('FX', 'PENDING_L2_APPROVAL', 'REJECTED', 'CENTER_DIRECTOR', true, true),
    -- Division head can also approve L2
    ('FX', 'PENDING_L2_APPROVAL', 'PENDING_BOOKING', 'DIVISION_HEAD', false, true),
    ('FX', 'PENDING_L2_APPROVAL', 'REJECTED', 'DIVISION_HEAD', true, true),
    -- Dealer recall from pending L2
    ('FX', 'PENDING_L2_APPROVAL', 'OPEN', 'DEALER', true, true),
    -- Accountant books → Pending Chief Accountant
    ('FX', 'PENDING_BOOKING', 'PENDING_CHIEF_ACCOUNTANT', 'ACCOUNTANT', false, true),
    -- Accountant voids
    ('FX', 'PENDING_BOOKING', 'VOIDED_BY_ACCOUNTING', 'ACCOUNTANT', true, true),
    -- Chief accountant approves → Pending Settlement
    ('FX', 'PENDING_CHIEF_ACCOUNTANT', 'PENDING_SETTLEMENT', 'CHIEF_ACCOUNTANT', false, true),
    -- Chief accountant voids
    ('FX', 'PENDING_CHIEF_ACCOUNTANT', 'VOIDED_BY_ACCOUNTING', 'CHIEF_ACCOUNTANT', true, true),
    -- Settlement officer settles → Completed
    ('FX', 'PENDING_SETTLEMENT', 'COMPLETED', 'SETTLEMENT_OFFICER', false, true),
    -- Settlement officer voids
    ('FX', 'PENDING_SETTLEMENT', 'VOIDED_BY_SETTLEMENT', 'SETTLEMENT_OFFICER', true, true),
    -- Cancel flow: Dealer requests cancel
    ('FX', 'COMPLETED', 'CANCELLED', 'DESK_HEAD', true, true),
    ('FX', 'PENDING_SETTLEMENT', 'CANCELLED', 'DESK_HEAD', true, true);
