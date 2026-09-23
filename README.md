# tapass

tapass 密码管理器 — 安全、简洁、自托管。

> **⚠️ Vibe Coding 项目**
>
> 本项目采用 [Vibe Coding](https://en.wikipedia.org/wiki/Vibe_coding) 方式开发——代码主要由 AI 生成，人类负责方向和决策。虽然经过编译、测试和审查，但**请谨慎使用**：
>
> - **不要用于存储关键密码或高敏感信息**
> - 加密实现可能存在未被发现的安全缺陷
> - 数据格式仍可能发生破坏性变更
> - 请在充分理解风险的前提下自行评估是否使用

## 仓库结构

```
tapass/
├── CHANGELOG.md    # 变更日志（发布包附带）
├── docs/           # 设计文档
│   ├── platforms.md   # 运行平台与资源约束
│   └── v1/            # V1 版本设计
├── tools/          # CLI 工具集 + 核心加密库
└── tui/            # TUI 客户端
```

## 子项目

### tools — CLI 工具集

核心加密库（vault）和命令行工具：

- `tapass-cli` — 交互式 CLI（create/open/set/get/delete/list/raw/passwd/compact）
- `tapass-import` — KeePass XML/KDBX 导入工具

### tui — TUI 客户端

基于 Bubble Tea 的终端交互界面，通过 `go.mod` replace 引用 tools 的 vault 库。

## 设计文档

- [文档索引与分类规则](docs/README.md)
- [V1 版本概述](docs/v1/overview.md)
- [V1 数据结构](docs/v1/data-structures.md)
- [V1 加密模型](docs/v1/crypto-model.md)
- [V1 JSON Schema](docs/v1/schema.json)
- [运行平台与资源约束](docs/platforms.md)
- [变更日志](CHANGELOG.md)

## 格式一致性

- 测试向量：[`tools/vault/testdata/format_v1_vectors.json`](tools/vault/testdata/format_v1_vectors.json)
- 独立校验脚本（不依赖参考实现）：`python3 docs/v1/verify_format_v1.py`
