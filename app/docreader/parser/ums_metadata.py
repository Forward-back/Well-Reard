"""银联商务（UMS）文档元数据抽取 —— 从 YLSW ingest 移植。

从文件名/标题提取：文号（doc_number）、机构（org_name）、日期（doc_date）、
文档类型（doc_type）、产品（product_name）。抽取结果写入 ReadResponse 的
metadata map，由 Go 侧写入 knowledges/chunks 元数据列（迁移 000093）。

命名约定：
  新格式: {TYPE}_{DATE}_{ORG}_{PRODUCT}_{TITLE}_{VERSION}_{ZH}.{ext}
  旧格式: 兼容兜底，用正则从文件名/标题中提取。
"""

import re
import logging

logger = logging.getLogger(__name__)

# ── 类型前缀映射 ──
TYPE_PREFIX_MAP = {
    "REG": "regulation", "MAN": "manual", "RPT": "report",
    "NWS": "news", "TBL": "table", "GEN": "general",
}

# ── 旧格式关键词分类 ──
REGULATION_KEYWORDS = [
    "通知", "办法", "条例", "公告", "规定", "法", "指引",
    "指导意见", "实施方案", "暂行办法", "管理办法", "令",
    "意见", "批复", "监督管理办法", "风险提示",
]
MANUAL_KEYWORDS = [
    "操作手册", "产品手册", "入网手册", "配置手册", "使用指南",
    "产品介绍", "手册", "介绍", "部署手册", "技术指南", "服务指引",
    "改造指引", "授权登录指引", "实施指引",
]
REPORT_KEYWORDS = ["报告", "财报", "研究", "分析", "季报", "年报",
                   "三季度报告", "三季度财报", "运行总体情况"]
NEWS_KEYWORDS = ["来了", "成功举办", "获刑", "退场", "热潮", "值得关注",
                 "飙涨", "增资", "缩水", "盘点", "嵌入", "迈出"]

# ── 文号正则（从标题中提取，新旧格式通用）─────────────────
DOC_NUMBER_PATTERNS = [
    re.compile(r"银发(?:\{|（)\s*(\d+)\s*(?:）|\})号"),
    re.compile(r"银发\s*(\d+)\s*号"),
    re.compile(r"银监办发(?:\{|（)\s*(\d+)\s*(?:）|\})号"),
    re.compile(r"银监办发\s*(\d+)\s*号"),
    re.compile(r"银监发(?:\{|（)\s*(\d+)\s*(?:）|\})号"),
    re.compile(r"银监发\s*(\d+)\s*号"),
    re.compile(r"银办发(?:\{|（)\s*(\d+)\s*(?:）|\})号"),
    re.compile(r"银办发\s*(\d+)\s*号"),
    re.compile(r"(?:中国人民银行|海关总署)?公告(?:\{|（)\s*(\d+)\s*(?:）|\})\s*第\s*(\d+)\s*号"),
    re.compile(r"中国人民银行令(?:\{|（)\s*(\d+)\s*(?:）|\})\s*第\s*(\d+)\s*号"),
    re.compile(r"汇发\s*(\d+)\s*号"),
]

# ── 旧格式：机构名称正则 ──
ORG_NAME_PATTERNS = [
    re.compile(r"中国人民银行"),
    re.compile(r"中国银监会"),
    re.compile(r"国家外汇管理局"),
    re.compile(r"国务院办公厅"),
    re.compile(r"国家发展改革委"),
    re.compile(r"中国银监会办公厅"),
    re.compile(r"中国人民银行办公厅"),
    re.compile(r"银联商务"),
    re.compile(r"中国银联"),
    re.compile(r"拉卡拉"),
    re.compile(r"移卡"),
    re.compile(r"抖音支付"),
    re.compile(r"支付宝"),
    re.compile(r"浦发银行"),
    re.compile(r"交通银行"),
    re.compile(r"光大银行"),
    re.compile(r"桂林银行"),
    re.compile(r"快手"),
    re.compile(r"洞见科技"),
    re.compile(r"御风算法"),
    re.compile(r"大华捷通"),
    re.compile(r"一川"),
    re.compile(r"中金点睛"),
]

# ── 日期正则（合并新旧格式）─────────────────────────────
DATE_PATTERNS = [
    re.compile(r"(\d{4})\s*年\s*第?\s*([\d一二三四五六七八九十]{1,2})\s*季度"),
    re.compile(r"(\d{4})\s*年\s*([\d一二三四五六七八九十]{1,2})\s*月"),
    re.compile(r"(\d{8})"),
    re.compile(r"(\d{4})-(\d{2})-(\d{2})"),
]


def _is_new_format(filename: str) -> bool:
    """检测是否新命名规范: TYPE_XXXXXXXX_ORG_PRODUCT_..."""
    name_no_ext = re.sub(r"\.(txt|pdf|docx?|xlsx|doc)$", "", filename, flags=re.IGNORECASE)
    parts = name_no_ext.split("_")
    if len(parts) < 6:
        return False
    return parts[0] in TYPE_PREFIX_MAP


def _classify_legacy(file_name: str) -> str:
    """旧格式：关键词匹配判断文档类型。"""
    clean = file_name.lower()
    if clean.endswith(".xlsx"):
        return "table"
    if any(kw in clean for kw in NEWS_KEYWORDS):
        return "news"
    clean_no_academy = clean.replace("研究院", "")
    if any(kw in clean_no_academy for kw in REPORT_KEYWORDS):
        return "report"
    if any(kw in clean for kw in MANUAL_KEYWORDS):
        return "manual"
    if any(kw in clean for kw in REGULATION_KEYWORDS):
        return "regulation"
    return "general"


def _extract_doc_number(text: str) -> str:
    """从文本中提取文号。"""
    for pattern in DOC_NUMBER_PATTERNS:
        m = pattern.search(text)
        if m:
            groups = m.groups()
            if len(groups) >= 2 and groups[0] and groups[1]:
                matched = m.group(0)
                for p in ["银发", "银监办发", "银监发", "银办发", "汇发"]:
                    if p in matched:
                        return f"{p}{groups[0]}{groups[1]}号"
                return re.sub(r"\s+", "", m.group(0))
            return re.sub(r"\s+", "", m.group(0))
    return ""


def _extract_date_from_title(title: str) -> str:
    """从标题文本中提取日期字符串。"""
    for pattern in DATE_PATTERNS:
        m = pattern.search(title)
        if m:
            return m.group(0)
    m = re.search(r"(?:银发|银监发|银监办发|银办发|汇发)\s*\{?\s*(\d{4})", title)
    if m:
        return f"{m.group(1)}0000"
    m = re.search(r"〔(\d{4})〕", title)
    if m:
        return f"{m.group(1)}0000"
    return ""


def extract_ums_metadata(file_name: str) -> dict:
    """从文件名提取 UMS 元数据，返回非空字段字典。

    优先走新格式命名解析，兜底旧正则。
    """
    if not file_name:
        return {}

    meta = {
        "doc_type": "",
        "org_name": "",
        "doc_number": "",
        "doc_date": "",
        "product_name": "",
    }

    if _is_new_format(file_name):
        name_no_ext = re.sub(r"\.(txt|pdf|docx?|xlsx|doc)$", "", file_name, flags=re.IGNORECASE)
        parts = name_no_ext.split("_")

        meta["doc_type"] = TYPE_PREFIX_MAP.get(parts[0], "general")
        meta["doc_date"] = parts[1] if len(parts) > 1 else ""
        meta["org_name"] = parts[2] if len(parts) > 2 else ""
        meta["product_name"] = parts[3] if len(parts) > 3 else ""

        lang_idx = len(parts) - 1
        version_idx = lang_idx - 1
        title_start = 4
        title_end = version_idx
        if title_end > title_start:
            title = "_".join(parts[title_start:title_end])
        else:
            title = parts[title_start] if title_start < len(parts) else ""

        meta["doc_number"] = _extract_doc_number(title)
        title_date = _extract_date_from_title(title)
        if title_date:
            meta["doc_date"] = title_date
    else:
        clean = re.sub(r"^\d{3,4}-", "", file_name)
        clean = re.sub(r"\.(txt|pdf|docx?|xlsx|doc)$", "", clean, flags=re.IGNORECASE)

        meta["doc_number"] = _extract_doc_number(clean)
        for pattern in ORG_NAME_PATTERNS:
            m = pattern.search(clean)
            if m:
                meta["org_name"] = m.group(0)
                break
        for pattern in DATE_PATTERNS:
            m = pattern.search(clean)
            if m:
                meta["doc_date"] = m.group(0)
                break
        meta["doc_type"] = _classify_legacy(file_name)

    return {k: v for k, v in meta.items() if v}


def enrich_metadata(metadata: dict, file_name: str) -> dict:
    """解析结果 metadata 注入 UMS 域元数据（非空才覆盖，兼容 docreader 自带字段）。"""
    try:
        ums = extract_ums_metadata(file_name)
    except Exception as e:
        logger.warning("UMS 元数据抽取失败（跳过）: %s", e)
        return metadata

    if not ums:
        return metadata

    if metadata is None:
        metadata = {}
    for k, v in ums.items():
        metadata.setdefault(k, v)
    logger.info("UMS 元数据: file=%s → %s", file_name, ums)
    return metadata
