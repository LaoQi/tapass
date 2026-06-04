# tapass

tapass 密码管理器 — 安全、简洁、自托管。

## 仓库结构

```
tapass/
├── docs/           # 设计文档
│   └── v1/         # V1 版本设计
├── tools/          # CLI 工具集 + 核心加密库
└── tui/            # TUI 客户端
```

## 子项目

### tools — CLI 工具集

核心加密库（vault）和命令行工具：

- `tapass-cli` — 交互式 CLI（create/open/set/get/delete/list/raw/passwd/compact）
- `tapass-import` — KeePass XML 导入工具

### tui — TUI 客户端

基于 Bubble Tea 的终端交互界面，通过 `go.mod` replace 引用 tools 的 vault 库。

## 设计文档

- [V1 版本概述](docs/v1/overview.md)
- [V1 数据结构](docs/v1/data-structures.md)
- [V1 加密模型](docs/v1/crypto-model.md)
- [V1 JSON Schema](docs/v1/schema.json)
