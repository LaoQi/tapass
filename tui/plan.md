# TUI 接收器混用重构

## 问题

tui 包中 10 个类型存在指针接收器和值接收器混用，违反 Go 一致性原则。

## 已完成

### 阶段一：清除死代码 ✅

删除 6 个无调用方法：
- MainViewModel: `SetCurrentPrefix`, `CurrentPrefix`, `SelectedEntryPath`, `RestoreSelection`
- PanelListModel: `SearchActive`, `SetDB`

### 阶段二：事件化 setter 方法 ✅

将父组件的命令式 setter 调用改为消息驱动，在子组件 Update 中处理。

**PanelListModel 消息类型**：`searchEnterMsg`, `searchExitMsg`, `searchFocusMsg`, `searchBlurMsg`, `moveUpMsg`, `moveDownMsg`, `setPrefixMsg`, `refreshMsg`, `dbEventMsg`

**EntryDetailModel 消息类型**：`syncRightMsg`, `startNewMsg`, `refreshTOTPMsg`

**MainViewModel 消息类型**：`dirtyMsg`, `cancelQuitMsg`, `mainViewModelEventMsg`

**HelpViewModel 消息类型**：`helpToggleMsg`, `helpCloseMsg`, `setHelpSizeMsg`

**WelcomeModel 消息类型**：`initialPathMsg`, `setWelcomeSizeMsg`

**已删除的公开方法**：
- PanelListModel: `ApplySearchFilter`, `SetPrefix`, `Refresh`, `HandleEvent`, `MoveUp`, `MoveDown`, `EnterSearch`, `ExitSearch`, `FocusSearchInput`, `BlurSearchInput`, `SearchInput`, `SetSearchInput`
- EntryDetailModel: `SetEntryPath`, `SelectAttr`, `SetAttrList`, `SetDetailMode`, `StartEdit`, `StartNew`, `RefreshTOTP`, `HandleEvent`
- MainViewModel: `CancelQuit`, `HandleDBEvent`, `RefreshAll`, `SetDirty`, `performSearch`
- HelpViewModel: `Toggle`, `Close`, `SetSize`
- WelcomeModel: `SetSize`, `SetInitialPath`

**保留的 getter**：
- MainViewModel: `MainState`
- PanelListModel: `Prefix`, `SelectedItem`, `ItemCount`
- EntryDetailModel: `State`, `HasSelectedEntry`, `EntryPath`
- HelpViewModel: `Active`

### 阶段三：统一接收器 ✅

所有类型统一为值接收器，零混用。

修改状态的方法改为返回修改后的 Model，调用方用 `m = m.method()` 接收。

## 未完成（后续迭代）

以下子组件方法仍为指针接收器，由 EntryDetailModel 内部直接调用：

| 类型 | 方法 | 说明 |
|------|------|------|
| EditorModel | StartEdit, StartNew, SetSize, resize, KeyValue, Value, EditKey, Mode, SetError, Blur | 编辑器子组件 |
| ConfirmModel | SetPendingKey, PendingKey, Clear | 确认删除子组件 |
| AttrListModel | SetAttrs | 属性列表子组件 |
| AttrViewModel | SetCopySuccess | 属性视图子组件 |
| TOTPModel | SetEntry, Update, Code, update | TOTP 子组件 |

这些子组件可进一步事件化，但收益较低（仅在 EntryDetailModel 内部使用）。
