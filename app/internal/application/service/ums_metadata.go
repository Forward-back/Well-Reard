package service

import (
	"regexp"
	"strings"
)

// 银联商务（UMS）文档元数据抽取 —— 从 YLSW ingest 移植到 Go 侧。
//
// 入库解析由 Gaze 内置 anydoc 完成（不走 docreader 容器），因此元数据
// 抽取在 Go 侧基于文件名执行，结果写入 knowledges/chunks 元数据列（迁移 000093）。
//
// 命名约定：
//  新格式: {TYPE}_{DATE}_{ORG}_{PRODUCT}_{TITLE}_{VERSION}_{ZH}.{ext}
//  旧格式: 兼容兜底，用正则从文件名/标题中提取。

// TYPE_PREFIX_MAP 新格式类型前缀 → 文档类型（与 YLSW doc_classifier 值域对齐）
var umsTypePrefixMap = map[string]string{
	"REG": "regulation", "MAN": "manual", "PDT": "product", "RPT": "report",
	"NWS": "news", "TBL": "table", "GEN": "general",
}

// 旧格式关键词分类
var umsRegulationKeywords = []string{
	"通知", "办法", "条例", "公告", "规定", "法", "指引",
	"指导意见", "实施方案", "暂行办法", "管理办法", "令",
	"意见", "批复", "监督管理办法", "风险提示",
}
var umsManualKeywords = []string{
	"操作手册", "产品手册", "入网手册", "配置手册", "使用指南",
	"产品介绍", "手册", "介绍", "部署手册", "技术指南", "服务指引",
	"改造指引", "授权登录指引", "实施指引",
}
var umsReportKeywords = []string{"报告", "财报", "研究", "分析", "季报", "年报",
	"三季度报告", "三季度财报", "运行总体情况"}
var umsNewsKeywords = []string{"来了", "成功举办", "获刑", "退场", "热潮", "值得关注",
	"飙涨", "增资", "缩水", "盘点", "嵌入", "迈出"}

// 文号正则（Go RE2 不支持非捕获组，已改写为字符类）
var umsDocNumberPatterns = []*regexp.Regexp{
	regexp.MustCompile(`银发[{（]\s*(\d+)\s*[}）]号`),
	regexp.MustCompile(`银发\s*(\d+)\s*号`),
	regexp.MustCompile(`银监办发[{（]\s*(\d+)\s*[}）]号`),
	regexp.MustCompile(`银监办发\s*(\d+)\s*号`),
	regexp.MustCompile(`银监发[{（]\s*(\d+)\s*[}）]号`),
	regexp.MustCompile(`银监发\s*(\d+)\s*号`),
	regexp.MustCompile(`银办发[{（]\s*(\d+)\s*[}）]号`),
	regexp.MustCompile(`银办发\s*(\d+)\s*号`),
	regexp.MustCompile(`(中国人民银行|海关总署)?公告[{（]\s*(\d+)\s*[}）]\s*第\s*(\d+)\s*号`),
	regexp.MustCompile(`中国人民银行令[{（]\s*(\d+)\s*[}）]\s*第\s*(\d+)\s*号`),
	regexp.MustCompile(`汇发\s*(\d+)\s*号`),
}

// 机构名称正则（旧格式）
var umsOrgNamePatterns = []*regexp.Regexp{
	regexp.MustCompile(`中国人民银行`),
	regexp.MustCompile(`中国银监会`),
	regexp.MustCompile(`国家外汇管理局`),
	regexp.MustCompile(`国务院办公厅`),
	regexp.MustCompile(`国家发展改革委`),
	regexp.MustCompile(`中国银监会办公厅`),
	regexp.MustCompile(`中国人民银行办公厅`),
	regexp.MustCompile(`银联商务`),
	regexp.MustCompile(`中国银联`),
	regexp.MustCompile(`拉卡拉`),
	regexp.MustCompile(`移卡`),
	regexp.MustCompile(`抖音支付`),
	regexp.MustCompile(`支付宝`),
	regexp.MustCompile(`浦发银行`),
	regexp.MustCompile(`交通银行`),
	regexp.MustCompile(`光大银行`),
	regexp.MustCompile(`桂林银行`),
	regexp.MustCompile(`快手`),
	regexp.MustCompile(`洞见科技`),
	regexp.MustCompile(`御风算法`),
	regexp.MustCompile(`大华捷通`),
	regexp.MustCompile(`一川`),
	regexp.MustCompile(`中金点睛`),
}

// 日期正则
var umsDatePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(\d{4})\s*年\s*第?\s*([\d一二三四五六七八九十]{1,2})\s*季度`),
	regexp.MustCompile(`(\d{4})\s*年\s*([\d一二三四五六七八九十]{1,2})\s*月`),
	regexp.MustCompile(`(\d{8})`),
	regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})`),
}

var umsExtRe = regexp.MustCompile(`\.(txt|pdf|docx?|xlsx|doc)$`)

// UMSMetadata is the extracted domain metadata of a document.
type UMSMetadata struct {
	DocType     string
	OrgName     string
	DocNumber   string
	DocDate     string
	ProductName string
}

// ExtractUMSMetadata extracts 银联商务 domain metadata from a file name.
// New format takes priority; legacy regexes are the fallback.
func ExtractUMSMetadata(fileName string) *UMSMetadata {
	if fileName == "" {
		return nil
	}
	if umsIsNewFormat(fileName) {
		return umsExtractNewFormat(fileName)
	}
	return umsExtractLegacyFormat(fileName)
}

func umsIsNewFormat(fileName string) bool {
	nameNoExt := umsExtRe.ReplaceAllString(fileName, "")
	parts := strings.Split(nameNoExt, "_")
	if len(parts) < 6 {
		return false
	}
	_, ok := umsTypePrefixMap[parts[0]]
	return ok
}

func umsExtractNewFormat(fileName string) *UMSMetadata {
	nameNoExt := umsExtRe.ReplaceAllString(fileName, "")
	parts := strings.Split(nameNoExt, "_")

	meta := &UMSMetadata{}
	meta.DocType = umsTypePrefixMap[parts[0]]
	if len(parts) > 1 {
		meta.DocDate = parts[1]
	}
	if len(parts) > 2 {
		meta.OrgName = parts[2]
	}
	if len(parts) > 3 {
		meta.ProductName = parts[3]
	}

	langIdx := len(parts) - 1
	versionIdx := langIdx - 1
	titleStart := 4
	titleEnd := versionIdx
	title := ""
	if titleEnd > titleStart {
		title = strings.Join(parts[titleStart:titleEnd], "_")
	} else if titleStart < len(parts) {
		title = parts[titleStart]
	}

	meta.DocNumber = umsExtractDocNumber(title)
	if d := umsExtractDateFromTitle(title); d != "" {
		meta.DocDate = d
	}
	return meta
}

func umsExtractLegacyFormat(fileName string) *UMSMetadata {
	clean := regexp.MustCompile(`^\d{3,4}-`).ReplaceAllString(fileName, "")
	clean = umsExtRe.ReplaceAllString(clean, "")

	meta := &UMSMetadata{}
	meta.DocNumber = umsExtractDocNumber(clean)
	for _, p := range umsOrgNamePatterns {
		if m := p.FindString(clean); m != "" {
			meta.OrgName = m
			break
		}
	}
	for _, p := range umsDatePatterns {
		if m := p.FindString(clean); m != "" {
			meta.DocDate = m
			break
		}
	}
	meta.DocType = umsClassifyLegacy(fileName)
	return meta
}

func umsClassifyLegacy(fileName string) string {
	clean := strings.ToLower(fileName)
	if strings.HasSuffix(clean, ".xlsx") {
		return "table"
	}
	for _, kw := range umsNewsKeywords {
		if strings.Contains(clean, kw) {
			return "news"
		}
	}
	cleanNoAcademy := strings.ReplaceAll(clean, "研究院", "")
	for _, kw := range umsReportKeywords {
		if strings.Contains(cleanNoAcademy, kw) {
			return "report"
		}
	}
	for _, kw := range umsManualKeywords {
		if strings.Contains(clean, kw) {
			return "manual"
		}
	}
	for _, kw := range umsRegulationKeywords {
		if strings.Contains(clean, kw) {
			return "regulation"
		}
	}
	return "general"
}

func umsExtractDocNumber(text string) string {
	wsRe := regexp.MustCompile(`\s+`)
	for _, p := range umsDocNumberPatterns {
		m := p.FindStringSubmatch(text)
		if m == nil {
			continue
		}
		groups := m[1:]
		matched := m[0]
		// 双数字文号（公告/令）：拼接两段数字
		if len(groups) >= 2 && groups[len(groups)-2] != "" && groups[len(groups)-1] != "" {
			a, b := groups[len(groups)-2], groups[len(groups)-1]
			for _, prefix := range []string{"银发", "银监办发", "银监发", "银办发", "汇发"} {
				if strings.Contains(matched, prefix) {
					return prefix + a + b + "号"
				}
			}
			return wsRe.ReplaceAllString(matched, "")
		}
		return wsRe.ReplaceAllString(matched, "")
	}
	return ""
}

func umsExtractDateFromTitle(title string) string {
	for _, p := range umsDatePatterns {
		if m := p.FindString(title); m != "" {
			return m
		}
	}
	if m := regexp.MustCompile(`(银发|银监发|银监办发|银办发|汇发)\s*\{?\s*(\d{4})`).FindStringSubmatch(title); m != nil {
		return m[2] + "0000"
	}
	if m := regexp.MustCompile(`〔(\d{4})〕`).FindStringSubmatch(title); m != nil {
		return m[1] + "0000"
	}
	return ""
}

// ExtractDocNumbersFromText returns all document numbers found in arbitrary
// user text (not just file names), in first-seen order. Internal whitespace is
// compacted so "银发〔 2024 〕5号" and "银发〔2024〕5号" are the same token.
func ExtractDocNumbersFromText(text string) []string {
	return umsFindAllUnique(umsDocNumberPatterns, text, true)
}

// ExtractOrgNamesFromText returns all known organization names found in text.
func ExtractOrgNamesFromText(text string) []string {
	return umsFindAllUnique(umsOrgNamePatterns, text, false)
}

// ExtractDatesFromText returns all date-like matches found in text.
func ExtractDatesFromText(text string) []string {
	return umsFindAllUnique(umsDatePatterns, text, true)
}

func umsFindAllUnique(patterns []*regexp.Regexp, text string, compact bool) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	seen := make(map[string]bool)
	out := make([]string, 0, 4)
	for _, pattern := range patterns {
		for _, match := range pattern.FindAllString(text, -1) {
			if compact {
				match = strings.Join(strings.Fields(match), "")
			} else {
				match = strings.TrimSpace(match)
			}
			if match == "" || seen[match] {
				continue
			}
			seen[match] = true
			out = append(out, match)
		}
	}
	return out
}
