#!/usr/bin/env python3
"""
手动 MCP 导入检查脚本（非 unittest 测试）。

用法: python mcp/scripts/check_imports.py
"""

try:
    import mcp.server.stdio  # noqa: F401

    print("[OK] mcp.server.stdio 导入成功")
except ImportError as e:
    print(f"[FAIL] mcp.server.stdio 导入失败: {e}")

try:
    # mcp 2.x high-level server API (MCPServer, formerly FastMCP).
    from mcp.server import MCPServer

    print("[OK] MCPServer 从 mcp.server 导入成功")
except ImportError as e:
    print(f"[FAIL] MCPServer 导入失败: {e}")

try:
    from mcp.server.mcpserver.exceptions import ToolError  # noqa: F401

    print("[OK] ToolError 从 mcp.server.mcpserver.exceptions 导入成功")
except ImportError as e:
    print(f"[FAIL] ToolError 导入失败: {e}")

try:
    import mcp_types  # noqa: F401  # standalone protocol types in mcp 2.x

    print("[OK] mcp_types 导入成功")
except ImportError as e:
    print(f"[FAIL] mcp_types 导入失败: {e}")

# 检查 MCP 包结构
import mcp

print(f"\nMCP 包版本: {getattr(mcp, '__version__', '未知')}")
print(f"MCP 包路径: {mcp.__file__}")
