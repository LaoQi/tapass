# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [dev]

### Added

- 条目详情编辑器内的密码生成器（`Ctrl+G`）：长度/字符类别/排除易混淆字符，`crypto/rand` 安全随机，
  保证每类启用字符至少出现一次，`enter` 应用到值区域
- `docs/platforms.md`：运行平台矩阵与资源约束（含可用内存 <32 MiB 的嵌入式设备场景）
- KDF 参数能力判断（Linux 读 `/proc/meminfo` 的 `MemAvailable`），不设参数上下限
- `tui/internal/tui` 测试文件：右栏意图路由（mainview_test.go）、错误展示（app_test.go）、
  新建覆盖确认与路径校验（welcome_test.go）回归
- `tui/internal/model/bench_test.go`：导航/查询基准（防性能回归）
- CI 门禁（`.github/workflows/ci.yml`）：gofmt / vet / test + 格式一致性独立校验 + windows、arm64 交叉编译冒烟
- 格式一致性资产：
  - `tools/vault/testdata/format_v1_vectors.json`：固定 salt/nonce/参数的测试向量
    （master key、HKDF 各步输出、子密钥、144 字节头部、数据段字节、AEAD 密文体字节）
  - `vault/vector_test.go`：参考实现按向量逐项校验 + 端到端组装/解析
  - `docs/v1/verify_format_v1.py`：纯 Python 手写 HKDF-SHA256 与 XChaCha20-Poly1305 的独立校验脚本
    （19 项断言全部通过，跨实现可照此验证）

### Changed

- 规范修订（本项目价值在于设计与算法一致性，规范必须可被其他平台照实现）：
  - `crypto-model.md` HKDF 拆分改为精确 5 步描述（旧描述缺少第二次 Extract 的说明；
    照旧描述字面实现会派生出完全不同的子密钥 —— 已用脚本复算对比确认）
  - 压缩流明确"不要求跨实现逐字节一致"，只要求解压还原后数据段一致
  - `data-structures.md` 补"同一时间戳取文件中靠后一条"的确定性解析规则
- `DB` 增加解析视图缓存（`resolvedIndex`/`invalidateResolved`）：`Query`/`QueryKeys`/`Get` 不再每次调用
  全量 `ResolveLatest`。此前 TUI 每次按键是 O(n²)，2000 条时单次导航约 117 ms；现在约 0.16 ms
  （`Get` 1.13 ms → 4.5 µs）
- 设计文档与实现对齐：压缩为原始 flate 流（RFC 1951），非 zlib 封装

### Fixed

- KDF 参数变更导致 vault 永久无法打开：新增 `vault.Rekey`（新 salt/nonce + 重新派生 + 重新加密 + 自校验），
  删除 `DB.SetConfig`；`Vault` 记录密钥派生上下文，`MarshalBinary` 在参数与密钥不一致时返回 `ErrKDFParamsChanged` 拒绝写出
- 解析伪造/损坏头部参数导致进程 panic（`argon2: parallelism degree too low` / `number of rounds too small`）：
  解析时校验参数可精确执行，非法参数返回错误而非崩溃
- `DeriveMasterKey` 的 `uint8(parallelism)` 隐式截断：改为显式校验参数
- 参数超出本机内存时被 OOM kill：解析/派生/建库前按本机可用内存判断能力，不足返回 `ErrInsufficientMemory`
- `zeroBytes` 对空切片 panic
- TUI `DB.OnChange` 返回的退订函数永不生效（用 `&l == &fn` 比较循环变量副本地址，
  且 Go 无法比较函数值）：监听器无法移除。改为按 id 定位，退订幂等
- vault `Compact()` 结果顺序随机（遍历 map）：幸存条目顺序不确定，破坏同一时间戳记录的先后关系。
  改为按幸存条目在原切片中的位置排序（确定性）
- vault `Sort()` 改为稳定排序：同一时间戳（同毫秒多次写入 / 导入的历史时间戳）的条目
  不得被重排，否则"同 key 同时间戳"记录可能解析出先写入的旧值
- KeePass 导入：同组同名条目被静默合并成一条（字段混杂、数据丢失）——
  实测后一条会完全覆盖前一条的属性。导入时按路径去重，同名追加 " (2)"、" (3)"…
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
  选中分组时右栏显示空详情而非属性列表。已拆分显式意图消息（`showAttrListMsg`/`showAttrDetailMsg`/`clearDetailMsg`）

## [v0.1.1] - 2026-06-11

### Added

- 搜索过滤（`/` 键进入）：基于当前前缀、大小写不敏感，过滤作用于原始 key 后再聚合为列表项
- 帮助覆盖层（`?` 键）

### Changed

- 发布包（zip）开始附带 `CHANGELOG.md`（`make dist`）

### Removed

- 移除 `DB.SearchKeys()` 全局搜索方法

## [v0.1.0] - 2026-06-10

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
  - 数据库设置（改密）
- 共享 version 包 + Makefile 交叉编译 + `--version` 参数
- GitHub Actions Release workflow（tag 触发，自动构建 Linux/Windows 并发布）
