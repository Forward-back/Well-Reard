-- Migration: seed knowledge_custom_field_defs (部门清单 + 自定义字段初版)
-- Description: 用户审改通过的初版种子数据(2026-10-17)。
-- 设计约定: department 行承载固定列「部门」的枚举(options 数组),
-- 其余行是 §2.14 维度 D 的自定义字段定义。tenant_id=0 为全局默认,
-- 各租户可通过同名行覆盖。
INSERT INTO knowledge_custom_field_defs (id, tenant_id, name, field_type, options, description)
VALUES
    ('def-department', 0, 'department', 'enum',
     '["市场部","产品部","技术部","运营部","风控部","客服部","财务部","综合管理部","合规部"]',
     '部门枚举(固定列 department 的可选值,上传表单下拉+检索过滤数据源)'),
    ('def-business-line', 0, 'business_line', 'enum',
     '["收单业务","创新业务","行业合作","其他"]',
     '业务条线自定义字段'),
    ('def-doc-scope', 0, 'doc_scope', 'enum',
     '["内部","商户","合作机构"]',
     '适用范围自定义字段')
ON CONFLICT (tenant_id, name) DO NOTHING;
