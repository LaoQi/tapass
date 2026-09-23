# 历史实施复盘（归档）

> **归档文档，非现行约定。** 这里记录 **v0.1.0 之前**的早期架构演进与踩坑过程。
> 该时期尚无 `CHANGELOG.md`，因此这些变更没有对应的发布条目。
> 现行行为以 `docs/v1/`、`docs/platforms.md` 与各 `AGENTS.md` 为准。

## 移除 store 包：持久化交由 model.DB 统管

**背景**：TUI 数据层最初设计了 `store.Store` 接口（先实现本地文件，规划 WebDAV 等后端），
由 store 负责文件 I/O，`DB` 只做内存状态与查询。

**问题**：

- `Store.Save` 需要调用方传入路径，而 vault 包不持有文件路径，路径信息要在 `DB` 与视图层之间来回传递。
- 实际上没有出现第二个后端实现，接口抽象只增加了一层间接与额外的状态同步点。
- dirty 状态与落盘时机分散，难以判断"是否已保存"。

**结论**：删除 `store` 包，`DB` 自身持有 dbPath，内部用 `atomicWriteFile`（临时文件 + rename）落盘；
vault 包保持不操作文件系统、所有 I/O 通过 `[]byte` 传递的边界。后续若需要 WebDAV 等后端，
再在 `DB` 之上按需抽象（见 `tui/AGENTS.md` 的持久化约定）。

## 相关记录

以下演进发生在 v0.1.1 之后（dev 期间），已记入 `CHANGELOG.md` 的 `[dev]` 段，此处不重复：

- 组件改为消息驱动（pane 层不再暴露 `SetSize`/`SetFocused`）
- 详情渲染拆分为无状态 Renderer 组件
- App 层统一 `page tea.Model` 路由、dirty 状态下沉到 `DB`
- 帮助界面从覆盖层改为独立窗口状态 `StateHelp`
- 模块路径迁移 `github.com/tapass` → `github.com/LaoQi/tapass`
