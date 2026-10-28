-- 银联商务融合项目：员工表（注册工号校验用）
-- 工号格式：字母前缀 + 数字（如 UMS10001）

CREATE TABLE IF NOT EXISTS employees (
    id INT AUTO_INCREMENT PRIMARY KEY,
    emp_no VARCHAR(20) NOT NULL UNIQUE COMMENT '工号',
    name VARCHAR(50) NOT NULL COMMENT '姓名',
    department VARCHAR(100) DEFAULT NULL COMMENT '部门',
    status TINYINT NOT NULL DEFAULT 1 COMMENT '1=在职 0=离职',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='员工表（注册校验）';

INSERT INTO employees (emp_no, name, department, status) VALUES
('UMS10001', '张伟', '产品部', 1),
('UMS10002', '王芳', '研发部', 1),
('UMS10003', '李强', '运营部', 1),
('UMS10004', '刘洋', '风控部', 1),
('UMS10005', '陈静', '合规部', 1),
('UMS10006', '杨帆', '数据部', 1),
('UMS10007', '赵磊', '市场部', 1),
('UMS10008', '孙丽', '客服部', 1),
('UMS10009', '周杰', '技术部', 1),
('UMS10010', '吴敏', '财务部', 1),
('UMS10011', '郑浩', '产品部', 0),
('UMS10012', '冯雪', '运营部', 0);
