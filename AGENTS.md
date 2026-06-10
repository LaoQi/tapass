# AGENTS.md — tapass

tapass 密码管理器单一仓库。

## 项目结构

```
tapass/
├── docs/           # 设计文档
│   └── v1/         # V1 版本设计
├── tools/          # CLI 工具集（独立 Go 模块）
│   ├── cmd/tapass-cli/        # 交互式 CLI 工具
│   ├── cmd/tapass-import/     # KeePass 导入
│   ├── vault/                # 核心加密库（非 internal，供 tui 跨模块引用）
│   ├── version/              # 共享版本信息包（供 tui 跨模块引用）
│   └── internal/
│       └── importer/         # KeePass 导入逻辑
└── tui/            # TUI 客户端（独立 Go 模块）
    ├── cmd/tapass-tui/
    └── internal/
        ├── model/            # 数据层（DB + 工具函数）
        └── tui/              # Bubble Tea 视图层
```

## 子项目

- **tools** — CLI 工具 + 核心加密库（vault），详见 `tools/AGENTS.md`
- **tui** — TUI 客户端，通过 replace 引用 tools 的 vault，详见 `tui/AGENTS.md`

## 设计文档

- V1 版本设计文档：`docs/v1/`
- 数据结构定义：`docs/v1/data-structures.md`
- 数据结构 JSON Schema：`docs/v1/schema.json`
- 加密模型：`docs/v1/crypto-model.md`
- 版本概述：`docs/v1/overview.md`

## Agent 读取指引

1. 读取数据结构时优先解析 `docs/v1/schema.json`，若为空则回退读取 `docs/v1/data-structures.md`
2. 不得在占位文件中填充自行推测的内容
3. 数据结构由项目负责人规划，Agent 不自行发挥
