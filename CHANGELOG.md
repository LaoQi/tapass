# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [dev]

### Fixed

- KDF 参数变更导致 vault 永久无法打开：新增 `vault.Rekey`（新 salt/nonce + 重新派生 + 重新加密 + 自校验），
  删除 `DB.SetConfig`；`Vault` 记录密钥派生上下文，`MarshalBinary` 在参数与密钥不一致时返回 `ErrKDFParamsChanged` 拒绝写出
- 解析伪造/损坏头部参数导致进程 panic（`argon2: parallelism degree too low` / `number of rounds too small`）：
  解析时校验参数可精确执行，非法参数返回错误而非崩溃
- `DeriveMasterKey` 的 `uint8(parallelism)` 隐式截断：改为显式校验参数
- 参数超出本机内存时被 OOM kill：解析/派生/建库前按本机可用内存判断能力，不足返回 `ErrInsufficientMemory`
- `zeroBytes` 对空切片 panic
- 新建/导入时目标文件已存在会被直接覆盖（数据丢失）：
  TUI 新建 vault 增加覆盖确认（`WelcomeConfirmOverwrite`，`y` 覆盖 / `n`、`esc` 返回），
  并校验空路径与目录路径；CLI `create` 默认拒绝覆盖（`create -f` 显式强制）；
  导入工具拒绝覆盖已有输出文件，且写出改为临时文件 + rename 原子替换
- 未知 Compression ID 被静默当作"无压缩"，格式扩展/文件损坏时会解出错乱数据：
  压缩标识在 `NewHeader`/`UnmarshalHeader`/`Open`/`MarshalBinary` 均显式校验，未知值返回 `ErrUnsupportedCompression`
- TUI 错误静默：`AppModel.err` 只赋值从不渲染，保存失败对用户完全不可见。`ErrorMsg` 现在统一投递到
  主视图状态栏（`[!] <错误>  [Ctrl+S] retry  [q] quit`，10 秒后自动清除），保存失败时保留 `[未保存]` 标记
- TUI 复制假成功：忽略了 `clipboard.WriteAll` 的错误、无条件显示"已复制到剪贴板"，
  现改为显示"复制失败: <原因>"（5 秒后清除）
- TUI 右栏属性列表不可达：`syncRightMsg` 的 `SetDetailMode` 分支覆盖了 `detailModeAttrList`，
  选中分组时右栏显示空详情而非属性列表。已拆分显式意图消息（`showAttrListMsg`/`showAttrDetailMsg`/`clearDetailMsg`），
  并补 tui 视图层回归测试

### Added

- `docs/platforms.md`：运行平台矩阵与资源约束（含可用内存 <32 MiB 的嵌入式设备场景）
- `tui/internal/tui` 测试文件：右栏意图路由（mainview_test.go）、错误展示（app_test.go）、
  新建覆盖确认与路径校验（welcome_test.go）回归
- KDF 参数能力判断（Linux 读 `/proc/meminfo` 的 `MemAvailable`），不设参数上下限

### Changed

- 重构搜索为基于原始 key 的过滤机制：面板存储 `rawKeys`，搜索过滤作用于原始 key 后再聚合生成列表项

### Removed

- 移除 `DB.SearchKeys()` 全局搜索方法

## [v0.1.0] - 2025-06-11

### Added

- vault 核心加密库：Argon2id + HKDF + XChaCha20-Poly1305
- vault 二进制格式：144 字节明文头部 + 变长密文体
- tapass-cli 交互式终端工具，支持 Tab 补全和 UTF-8
- tapass-import KeePass XML/KDBX 导入工具
- tapass-tui TUI 客户端
  - 双栏布局：左侧分组/属性列表 + 右侧详情/编辑
  - vim 导航（h/j/k/l）+ Tab 焦点切换
  - TOTP / Steam TOTP 验证码生成
  - 属性值复制到剪贴板
  - dirty 标记 + 保存/退出确认
  - 搜索过滤（`/` 键进入，基于当前前缀）
  - 数据库设置（改密）
  - 帮助覆盖层（`?` 键）
- 共享 version 包 + Makefile 交叉编译 + `--version` 参数
- GitHub Actions Release workflow（tag 触发，自动构建 Linux/Windows 并发布）
