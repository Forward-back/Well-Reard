package tools

// maxFunctionNameLength is the maximum length for a tool/function name
// imposed by the OpenAI API.
const maxFunctionNameLength = 64

// Tool names constants
const (
	// Capability-scoped MCP discovery and invocation; not tenant-selectable builtins.
	ToolDiscoverMCPTools    = "discover_mcp_tools"
	ToolCallMCPTool         = "call_mcp_tool"
	ToolThinking            = "thinking"
	ToolTodoWrite           = "todo_write"
	ToolGrepChunks          = "grep_chunks"
	ToolKnowledgeSearch     = "knowledge_search"
	ToolListKnowledgeChunks = "list_knowledge_chunks"
	ToolGetDocumentInfo     = "get_document_info"
	ToolSearchConversations = "search_conversations"
	ToolSearchMemory        = "search_memory"
	ToolDatabaseQuery       = "database_query"
	ToolDataAnalysis        = "data_analysis"
	ToolDataSchema          = "data_schema"
	ToolWebSearch           = "web_search"
	ToolWebFetch            = "web_fetch"
	// Unified reading: workspace access follows sandbox file capability;
	// skill resources follow SkillsEnabled and do not require a sandbox.
	ToolReadFile = "read_file"
	// Sandbox filesystem tools (only available when the sandbox backend
	// supports per-session files — Cube, E2B, Docker). list/read inspect
	// the session workspace; write creates text files so generated
	// scripts do not have to travel through a shell_exec heredoc; edit
	// patches an existing file without regenerating it.
	//
	// Deliberately absent from AvailableToolDefinitions and
	// DefaultAllowedTools, like search_memory and web_search: the sandbox
	// switch already decides whether a run has a workspace at all, and
	// registerSandboxFileTools registers these from that capability rather
	// than from the allowlist. A checkbox would have been a lie — clearing
	// it changed nothing.
	ToolListSandboxFiles = "list_sandbox_files"
	ToolWriteSandboxFile = "write_sandbox_file"
	ToolEditSandboxFile  = "edit_sandbox_file"
	// ToolWriteSkillFile / ToolEditSkillFile write the skill tree under
	// /opt/gaze/tenant/skills rather than /workspace, and exist only for
	// the built-in skill installer. They are scoped to the one skill being
	// installed; see internal/agent/tools/skill_file.go.
	//
	// Deliberately absent from AvailableToolDefinitions: these write the
	// shared snapshot image, so they are granted by install mode alone and
	// must not become selectable on a tenant-editable agent config.
	ToolWriteSkillFile = "write_skill_file"
	ToolEditSkillFile  = "edit_skill_file"
	// ToolShellExec lets the LLM execute ad-hoc shell commands inside the
	// current session's sandbox (dependency installs, environment probing).
	// Registered only when the resolved backend advertises the session shell
	// capability (Cube, E2B, Docker). The command never runs on the Gaze host.
	//
	// Also absent from AvailableToolDefinitions: registerSandboxShellIfAllowed
	// keys it on SkillsEnabled (or install mode), so the shell follows the
	// skills switch and not a per-agent tool checkbox.
	ToolShellExec = "shell_exec"
	// Gaze-related tools (only available when gaze KBs are in scope)
	ToolGazeReadPage      = "gaze_read_page"
	ToolGazeWritePage     = "gaze_write_page"
	ToolGazeReplaceText   = "gaze_replace_text"
	ToolGazeRenamePage    = "gaze_rename_page"
	ToolGazeDeletePage    = "gaze_delete_page"
	ToolGazeSearch        = "gaze_search"
	ToolGazeReadSourceDoc = "gaze_read_source_doc"
)

// AvailableTool defines a simple tool metadata used by settings APIs.
type AvailableTool struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// AvailableToolDefinitions returns the list of tools exposed to the UI.
// Keep this in sync with registered tools in this package.
func AvailableToolDefinitions() []AvailableTool {
	return []AvailableTool{
		{Name: ToolThinking, Label: "思考", Description: "动态和反思性的问题解决思考工具"},
		{Name: ToolTodoWrite, Label: "制定计划", Description: "创建结构化的研究计划"},
		{Name: ToolGrepChunks, Label: "关键词搜索", Description: "快速定位包含特定关键词的文档和分块"},
		{Name: ToolKnowledgeSearch, Label: "语义搜索", Description: "理解问题并查找语义相关内容"},
		{Name: ToolListKnowledgeChunks, Label: "查看文档分块", Description: "获取文档完整分块内容"},
		{Name: ToolGetDocumentInfo, Label: "获取文档信息", Description: "查看文档元数据"},
		{
			Name:        ToolSearchConversations,
			Label:       "回顾历史对话",
			Description: "在用户自己的历史会话中查找之前聊过的内容",
		},
		{Name: ToolDatabaseQuery, Label: "查询数据库", Description: "查询数据库中的信息"},
		{Name: ToolDataAnalysis, Label: "数据分析", Description: "理解数据文件并进行数据分析"},
		{Name: ToolDataSchema, Label: "查看数据元信息", Description: "获取表格文件的元信息"},
		{Name: ToolGazeReadPage, Label: "读取Gaze页面", Description: "读取指定的Gaze页面内容"},
		{Name: ToolGazeSearch, Label: "搜索Gaze", Description: "在Gaze中搜索页面"},
		{Name: ToolGazeReadSourceDoc, Label: "精读源文档", Description: "使用知识点深入阅读特定原始文档"},
		{Name: ToolGazeWritePage, Label: "创建/覆盖Gaze", Description: "创建新页面或完全覆盖已有页面"},
		{Name: ToolGazeReplaceText, Label: "局部替换Gaze", Description: "替换Gaze页面中的特定文本"},
		{Name: ToolGazeRenamePage, Label: "重命名Gaze", Description: "重命名Gaze页面并自动更新关联链接"},
		{Name: ToolGazeDeletePage, Label: "删除Gaze", Description: "删除Gaze页面并自动清理关联死链"},
	}
}

// DefaultAllowedTools returns the default allowed tools list.
func DefaultAllowedTools() []string {
	return []string{
		ToolKnowledgeSearch,
		ToolGrepChunks,
		ToolListKnowledgeChunks,
		ToolGetDocumentInfo,
		// Looking up what this user asked before is only ever a read of their
		// own history, and it is what lets "上次你给我的那个配置" resolve at all
		// without stuffing every past conversation into the context window.
		ToolSearchConversations,
		// ToolSearchMemory is deliberately absent here and from
		// AvailableToolDefinitions. Like web_search it is not chosen from this
		// list at all: registerTools injects it whenever the workspace, the
		// user and the agent all allow memory, and strips it whenever they do
		// not. Adding it here would let a stale allowlist decide something the
		// memory switches already decide.
		// Graph, SQL, data analysis and explicit planning are opt-in. Existing
		// agents keep their explicit allowlists; domain presets select extras.
	}
}

// Retired tool identifiers are retained only for decoding existing histories
// and ignoring obsolete allowlist entries. They have no implementation or
// registration path and are never offered in model tool schemas.
const (
	LegacyToolExecuteSkillScript = "execute_skill_script"
	LegacyToolReadSkill          = "read_skill"
	LegacyToolReadSandboxFile    = "read_sandbox_file"
)

// RetiredToolReplacement tells the model how to replace a removed tool.
// Empty when name was never a Gaze tool.
func RetiredToolReplacement(name string) string {
	switch name {
	case LegacyToolExecuteSkillScript:
		return "execute_skill_script is no longer available; use shell_exec(skill_name=..., command=...) to run skill scripts"
	case LegacyToolReadSkill:
		return `read_skill is no longer available; use read_file(path="skill://<name>/<file_path or SKILL.md>")`
	case LegacyToolReadSandboxFile:
		return "read_sandbox_file is no longer available; use read_file(path=...)"
	default:
		return ""
	}
}
