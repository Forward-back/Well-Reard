# Frontend design sources

设计源文件，不进入生产构建。运行资源仍在 `frontend/src/assets/`。

## UI source

`ui-source/icons/` 为手绘/矢量源文件，已统一英文命名：

| 原文件 | 现文件 |
|---|---|
| 对话.png | chat.png |
| 检索文档.png | retrieve-document.png |
| 组装.png | assemble.png |
| 检索完成.png | retrieve-done.png |
| 画笔.png | brush.png |
| 机器人（手绘）.png | bot-handdrawn.png |
| 大脑（指代llm）.png | brain-llm.png |
| 共享知识库（侧边图标）.png | shared-kb.png |
| 知识库（侧边栏图标）.png | knowledge-base.png |
| 分类.png / 标签.png | category.png / tag.png |
| 加载.png / 最近.png | loading.png / recent.png |
| 协作.png / 治理.png | collaboration.png / governance.png |
| 搜索.png / 智能体.png | search.png / agent.png |
| 注册.png / 编辑.png | register.png / edit.png |
| 设置.svg / 权限.svg | settings.svg / permissions.svg |

`brand/` 保存未进入运行时的手绘品牌图。`fonts/` 保存字体源压缩包；实际字体与 OFL 许可已解压到 `frontend/src/assets/fonts/`，分发时以该目录为准。

## 图标矢量化

```bash
python frontend/scripts/vectorize_icon.py \
  frontend/design/ui-source/icons/chat.png --json
```

生成结果写入聊天检索流程图标；修改源图后同步更新 `frontend/src/views/chat/components/traceSourceIcons.ts` 的头部说明。
