# 文档索引

本目录存放 tapass 的设计规范与文档。**每类内容有唯一归属**，避免历史实施过程混入规范。

## 分类规则

| 内容性质 | 归属 | 说明 |
|----------|------|------|
| 格式规范、加密模型、平台约束 | `docs/v1/`、`docs/platforms.md` | 跨平台实现的契约；改动需同步测试向量并保证校验脚本通过 |
| 可执行校验资产 | `docs/v1/verify_format_v1.py`、`tools/vault/testdata/format_v1_vectors.json` | 由 CI 执行；路径被 CI 与多份文档引用，不要移动 |
| 面向 agent 的现行实现约定 | 仓库根与子模块的 `AGENTS.md` | 只写**当前**约定，不写历史 |
| 面向发布的变更条目 | `CHANGELOG.md` | 新功能、行为变更、缺陷修复 |
| 历史实施复盘 | `docs/archive/` | 早期架构演进与踩坑记录，**非现行约定** |

原则：

- 规范文档只描述当前契约，不写"曾经怎样"；历史过程进 `docs/archive/`，变更进 `CHANGELOG.md`
- 规范与实现不一致按缺陷处理（见根 `AGENTS.md` 的 Agent 读取指引）

## 目录

- `v1/` — V1 版本规范
  - `overview.md` — 版本目标、技术选型、文件格式概览
  - `data-structures.md` — 头部与数据段字节布局（契约）
  - `crypto-model.md` — Argon2id / HKDF / XChaCha20-Poly1305 与校验流程（契约）
  - `schema.json` — 上述布局的机器可读 JSON Schema
  - `verify_format_v1.py` — 独立校验脚本（纯 Python，不依赖参考实现）
- `platforms.md` — 运行平台矩阵与资源约束
- `archive/implementation-notes.md` — 历史实施复盘（归档）

## 建议阅读顺序

实现一个新平台客户端时：

1. `v1/overview.md` — 了解范围与技术选型
2. `v1/data-structures.md` — 字节布局
3. `v1/crypto-model.md` — 密钥派生与加密校验流程
4. `v1/schema.json` — 机器可读字段定义
5. `python3 docs/v1/verify_format_v1.py` — 用测试向量校验本地实现
