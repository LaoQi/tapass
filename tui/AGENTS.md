# AGENTS.md — tapass-tui

tapass 密码管理器 TUI 客户端。

## 构建 & 测试

```bash
go build ./...                                    # 编译
go test ./internal/model/ -v                     # model 测试
go vet ./...                                      # 静态检查
go run ./cmd/tapass-tui/                          # 运行
```

## 包结构

```
cmd/tapass-tui/main.go     # 入口（可选数据库路径参数）
internal/
  model/                    # 数据层（DB + 工具函数）
    db.go                   # DB 核心：newDB/OpenDB/CreateDB(不写文件,dirty=true)/Save(清除dirty)/Query/QueryKeys/Get/Set/Delete/ChangePassword/Config/SetConfig/Dirty/OnChange/atomicWriteFile（已移除 SearchKeys）
    db_test.go              # DB 测试（含持久化测试）
    listing.go              # ListItem(Depth字段) + normalizePathPrefix/ParentPath/EntryPath
    listing_test.go
  tui/                      # Bubble Tea 视图层
    app.go                  # 主 Model + AppState(DB/DBPath) + page tea.Model 页面路由 + 窗口状态(StateWelcome/StateMainView/StateHelp/StateDBConfig) + 消息类型 + switchToMainView/updateMainView 辅助
    welcome.go              # 欢迎/打开/新建数据库（TAPASS ASCII art，使用 model.OpenDB/CreateDB）
    mainview.go             # 双栏布局 + vim导航 + TOTP tick管理 + dirty标记 + MainState(StateBrowse/StatePendingQuit) + searchActive + 三焦点 + 搜索过滤 + propagatePanelSize/propagatePanelFocus + updateLeft/updateRight 类型断言辅助
    panellist.go            # 列表面板（rawKeys存储原始key + buildItems聚合 + rebuildItems过滤聚合 + 分组/属性图标 + 搜索框 + Depth区分 + 滚动跟随 + resizeMsg/setFocusMsg 消息驱动）
    entrydetail.go          # 条目详情状态管理（detailState/detailMode + Update消息处理 + View路由分发 + IsTOTP/TOTPCode读取 + refresh/saveKV/resizeEditor + 密码生成器状态detailPassGen）
    detail_attrlist.go      # AttrListView — 属性列表渲染组件（Renderer接口）
    detail_empty.go         # EmptyDetailView — 空详情渲染组件（Renderer接口）
    detail_text.go          # TextDetailView — 文本属性渲染组件（Renderer接口）
    detail_totp.go          # TOTPDetailView — TOTP渲染组件（Renderer接口 + ComputeCode + parseOtpAuthURI + newHash）
    detail_editkv.go        # EditKVView — KV编辑渲染组件（Renderer接口）
    detail_passgen.go       # PassGenView — 密码生成器渲染组件（Renderer接口）
    passgen.go              # 密码生成器状态与逻辑（PassGenState/PassGenRules + 生成/游标/切换 + crypto/rand安全随机）
    renderer.go             # Renderer 接口(View() string) + wrapBorder 包级函数
    helpview.go             # 帮助视图（StateHelp 窗口状态，居中面板显示快捷键说明）
    dbconfig.go             # 数据库设置（改密，成功发 PasswordChangedMsg + resizeMsg 消息驱动）
    styles.go               # 共享样式（含 TOTP/进度条/dirty 标题/状态栏按键/复制成功/密码生成器样式）
```

## 依赖说明

- tui 通过 `go.mod` 的 `replace` 指令引用 `tools/vault`：
  ```
  replace github.com/LaoQi/tapass/tools => ../tools
  ```
- vault 包路径为 `github.com/LaoQi/tapass/tools/vault`（非 internal，允许跨模块访问）
- version 包路径为 `github.com/LaoQi/tapass/tools/version`（非 internal，允许跨模块访问）
- vault 源码只在 tools 中维护，tui 不包含 vault 副本
- vault API 变更会直接影响 tui 编译

## 二进制格式关键约束

- 头部 144 字节明文 + 变长密文体
- Magic（ASCII）大端序，其余多字节字段小端序
- 密钥派生：Argon2id(32B) → HKDF-SHA256 → HMAC Key(32B) + Encrypt Key(32B)
- HKDF info：`tapass-v1-hmac` / `tapass-v1-enc`
- 加密：XChaCha20-Poly1305，压缩在加密前执行（flate 裸 DEFLATE，非 zlib）
- 头部校验：Header MAC = SHA256(header[0:80])，Header HMAC = HMAC-SHA256(MAC, HMAC_Key)
- 数据段：追加式 KV，同 key 取最新时间戳，Type=0 表示删除
- Key 路径以 `/` 分隔，特殊属性名：PASSWD / SSH / TOTP

## 实现约定

- Key 路径格式：`/group/subgroup/entry/ATTR_NAME`，最后一段为属性名
- Node.ID 是路径标识符兼显示名，无 Name 字段
- 分组规则：有子节点自动推断为组
- 压缩使用 flate 裸 DEFLATE 流（非 zlib），与设计文档 "zlib/DEFLATE" 描述不同，为本项目明确选择
- Node.Path 统一以 `/` 开头，与 vault key 前缀一致
- 新建条目只创建路径前缀，用户在详情页添加属性
- 属性编辑先统一文本，PASSWD/TOTP/SSH 专属控件后续迭代
- 存储扩展通过实现 `store.Store` 接口（先 local，后续 WebDAV）→ **已删除 store 包，DB 统管持久化**
- Store.Save 需传入 path 参数（vault 不持有文件路径）→ **DB.Save() 自身持有 dbPath**
- vault 包不操作文件系统，文件 I/O 由 store 实现负责（local 使用原子写入）→ **DB 内部 atomicWriteFile 处理**
- TUI 组件均持有 width/height，通过 resizeMsg 消息响应 resize（不再使用 SetSize 方法）
- 共享样式定义在 `tui/styles.go`
- lipgloss v2 border 渲染存在 2 列内部开销：设置 `Width(w)` 时实际可用内容宽度为 `w-2`，因此 `wrapBorder` 设置 `Width(width-2)` 后实际可用宽度为 `width-4`
- 列表面板标题和详情面板标题均使用底部边框分隔符（NormalBorder bottom），渲染宽度需减 2 抵消内部开销
- 列表项 icon 显示宽度硬编码为 `iconDisplayWidth=3`（emoji 2列+空格 1列），label 最大宽度 = `width - 5 - iconDisplayWidth`
- `wrapLine` 使用 `runewidth.StringWidth` 按显示宽度切片，不按字节切片，确保中文/宽字符正确换行
- 左栏=当前级子项（Depth>0为分组📂，Depth=0为属性🔖），右栏=属性列表/属性详情
- 基于 `/` 前缀筛选条目，不构建树形结构（listing.go）
- vim 导航：h=上一级，j/k=上下，l=打开/聚焦右栏；Tab=切换左右栏
- 左栏选中属性(Depth=0)时 e/y/d 快捷键直接跳转右栏操作
- 详情面板 detailModeAttrList：显示属性名+修改时间列表；detailModeDetail：标题=属性key，第一栏=修改时间，第二栏=属性值
- TOTP 属性：解析 `otpauth://totp/` URI，支持 secret/digits/period/algorithm(SHA1/SHA256/SHA512)参数；计算逻辑在 TOTPDetailView.ComputeCode()，入口在 detail_totp.go
- Steam TOTP：digits=S 时使用字符表 `23456789BCDFGHJKMNPQRTVWXY` 生成5位字符码
- TOTP tick 通过 mainview 的 totpActive 标志 + Update 末尾检查启动；tick 到达时发送 refreshTOTPMsg 更新 TOTPDetailView
- 列表超长时限制显示高度，选中项自动滚动跟随
- dirty 标记：DB 内部维护 dirty 状态（单一真相源），Set/Delete/ChangePassword/SetConfig 后自动标记 dirty=true，Save 成功后清除；CreateDB 创建的 DB 初始 dirty=true（未落盘）；OpenDB 打开的 DB 初始 dirty=false（已从文件加载）；App 层通过 DB.Dirty() 读取状态发送 dirtyMsg 给 MainView
- 编辑/删除属性后发送 AttrChangedMsg，修改密码成功后发送 PasswordChangedMsg
- 退出确认：dirty 时按 q 进入 StatePendingQuit 状态，状态栏显示 `[y] save & quit  [n] quit without saving  [esc] cancel`
- 保存并退出：y → 发 SaveAndQuitMsg → 保存成功后 VaultSavedMsg{QuitAfter:true} → tea.Quit
- 不保存退出：n → 直接 tea.Quit
- Ctrl+S 仅保存：dirty 时保存，非 dirty 时静默无操作
- `c` 键打开设置：在 app 层直接响应，不转发到子面板，搜索框聚焦时屏蔽
- `?` 键打开帮助视图：切换到 StateHelp 窗口状态（非覆盖层），esc/?/q 返回主视图
- 删除功能在右栏面板开放，左栏选中属性时也可按 d 进入；删除仅能对属性（完整 key）操作
- 删除需二次确认：按 `d` 进入 detailConfirmDelete 状态，再按 `d`/`y` 确认，其他键取消
- DB 不暴露 vault：删除 `Vault()`/`Header()` 方法，外部禁止直接调用 vault
- DB 新增 `Config()`/`SetConfig()` 接口：获取与设置 Config（含 Argon2 参数），对外不暴露 Header
- `NewDB` 私有化为 `newDB`：外部通过 `OpenDB`/`CreateDB` 获取 DB 实例；CreateDB 不立即写文件，需手动 Save 落盘
- 状态栏 `[c] config` 始终显示，`[Ctrl+S] save` 仅 dirty 时显示
- 欢迎页居中排版，TAPASS ASCII art 紫色显示
- 右侧面板查看模式下按 `y` 复制属性值到剪贴板：TOTP 属性复制当前验证码，其他属性复制原始值
- 复制成功显示"已复制到剪贴板"提示（copySuccessStyle），1.5 秒后由 copyClearMsg 自动清除
- copyClearMsg 由 entrydetail 产生，mainview 层转发，不跳过子组件路由
- 切换条目/属性/状态时（SetEntryPath/SelectAttr/StartEdit/StartNew/SetAttrList/SetDetailMode）自动清除复制提示
- 密码生成器：编辑KV时按 Ctrl+G 进入 detailPassGen 状态；j/k 导航规则行，space 切换布尔项，+/- 调整长度，g 生成密码，enter 应用到值区域，esc 返回编辑
- 密码生成器使用 crypto/rand 安全随机，保证每类启用字符至少出现一次；支持排除易混淆字符（0Oo1lI）
- EntryDetailModel 持有 passGen PassGenState 字段，进入密码生成器时初始化，应用后回填 valueArea

## 组件架构约定

- 所有子组件统一实现 `tea.Model` 接口：`Update() (tea.Model, tea.Cmd)` + `View() tea.View`
- 渲染组件实现 `Renderer` 接口：`View() string`（定义在 renderer.go），纯渲染无状态，通过指针接收者 Set 方法注入数据（Set 无返回值）
- 渲染组件：AttrListView / EmptyDetailView / TextDetailView / TOTPDetailView / EditKVView / PassGenView
- 渲染组件使用方式：`v := &XxxView{}; v.SetXxx(...); content = v.View()`
- EntryDetailModel 持有状态和 Update 逻辑，View() 根据状态创建 Renderer 指针实例并注入数据渲染，最后统一 wrapBorder 包装
- App 层使用 `page tea.Model` 统一路由，不再为每个页面持有独立字段
- `AppState` 集中管理 DB/DBPath，dirty 状态由 DB 内部维护，App 层通过 DB.Dirty() 读取
- 子组件不再暴露 SetSize/SetFocused 方法，改用 `resizeMsg`/`setFocusMsg` 消息驱动
- `updateLeft`/`updateLeftCmd`/`updateRight`/`updateRightCmd` 辅助函数处理 tea.Model → 具体类型的断言
- Help 从覆盖层模式改为独立窗口状态（StateHelp），不再持有 active 标志
- `propagatePanelSize()`/`propagatePanelFocus()` 在 mainview 中集中分发 resize/focus 消息给子面板
- `wrapBorder` 为包级函数（renderer.go），供 EntryDetailModel.View() 统一调用

## 搜索过滤机制

- 搜索是浏览模式上的过滤层，非独立状态：`searchActive bool` + `StateBrowse`
- 三焦点模型：`focusSearch`/`focusLeft`/`focusRight`，Tab 循环 `(focus + 1) % 3`
- `/` 键进入搜索，搜索框聚焦时仅 `esc`/`tab`/`enter` 特殊路由，其余转发给 searchInput
- 搜索范围：当前前缀下的原始 key 全路径匹配（`strings.Contains(strings.ToLower(key), lowerQuery)`），不区分大小写
- 空查询显示当前前缀下所有条目
- `esc` 统一退出搜索 + 清空过滤 + 焦点回到左面板
- `enter`（搜索框聚焦）：焦点移到左面板
- `enter`（列表聚焦）：等同于浏览模式
- 导航子分组（`h`/`l`）时保持搜索，`SetPrefix` 触发 `doQuery` 重新填充 rawKeys 后自动 `rebuildItems`
- 数据流：`DB.QueryKeys(prefix)` → `rawKeys` → 过滤 → `buildItems(filteredKeys)` → `items`
- 前缀变更 → `doQuery` 重新查 DB 填充 `rawKeys` + `rebuildItems`
- 搜索词变更 → 仅 `rebuildItems`，不查 DB
- `panellist.searchMode` 仅用于 View 渲染（搜索框、FullPath 显示、空匹配提示）
- `mainview.searchActive` 是搜索状态的权威标识
