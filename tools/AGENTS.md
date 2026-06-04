# AGENTS.md — tapass-tools

tapass CLI 工具集 + 核心加密库。

## 构建 & 测试

```bash
go build ./...                                    # 编译
go test ./vault/ -v -timeout 120s             # vault 测试（Argon2id 慢，需 timeout）
go test ./internal/importer/ -v -timeout 120s  # importer 测试
go vet ./...                                      # 静态检查
go run ./cmd/tapass-cli/                            # 运行交互式 CLI
go run ./cmd/tapass-import/                         # 运行导入工具
```

## 包结构

```
cmd/tapass-cli/main.go       # 交互式 CLI 工具入口
cmd/tapass-import/main.go    # KeePass XML 导入入口
vault/                       # 核心加密库（非 internal，供 tui 跨模块引用）
  crypto.go                  # Argon2id + HKDF + XChaCha20-Poly1305
  compress.go                # flate 裸 DEFLATE
  entry.go                   # KV 条目序列化
  header.go                  # 144 字节头部
  vault.go                   # Vault CRUD + ChangePassword + Compact
  vault_test.go
internal/
  importer/                  # KeePass XML 导入
    keepass.go
    keepass_test.go
```

## 二进制格式关键约束

- 头部 144 字节明文 + 变长密文体
- Magic（ASCII）大端序，其余多字节字段小端序
- 密钥派生：Argon2id(32B) → HKDF-SHA256 → HMAC Key(32B) + Encrypt Key(32B)
- HKDF info：`tapass-v1-hmac` / `tapass-v1-enc`
- 加密：XChaCha20-Poly1305，压缩在加密前执行（flate 裸 DEFLATE，非 zlib）
- 头部校验：Header MAC = SHA256(header[0:80])，Header HMAC = HMAC-SHA256(MAC, HMAC_Key)
- 数据段：追加式 KV，同 key 取最新时间戳，Type=0 表示删除
- Key 路径以 `/` 分隔，特殊属性名：PASSWD / SSH / TOTP

## vault API

vault 包不直接操作文件系统，所有 I/O 通过 `[]byte` 传递：

| 函数 | 签名 | 说明 |
|------|------|------|
| `Create` | `(password string) ([]byte, error)` | 创建空 vault，返回序列化字节 |
| `Open` | `(data []byte, password string) (*Vault, error)` | 从字节反序列化并解密 |
| `MarshalBinary` | `(*Vault) ([]byte, error)` | 序列化为字节（重新生成 Nonce） |
| `ChangePassword` | `(old, new string) ([]byte, error)` | 改密，返回新 vault 字节 |
| `Set/SetBlob/Delete` | 纯内存操作 | 不自动持久化 |
| `Sort` | `()` | 排序 Entries |
| `Compact` | `()` | 压缩 Entries（纯内存） |

- 文件读写由调用方负责（CLI、importer、TUI store 各自实现原子写入）

## 实现约定

- Argon2 默认参数：time=6, memory=16384 (16 MiB), parallelism=1
- 压缩使用 flate 裸 DEFLATE 流（非 zlib），与设计文档 "zlib/DEFLATE" 描述不同，为本项目明确选择
- MarshalBinary 每次调用重新生成 Nonce
- SubKeys.Zero() 安全清零密钥

## CLI 用法

tapass-cli 为交互式终端工具，启动时指定 vault 文件，密码通过终端提示输入（不回显）：

```
tapass-cli <vault-file>
```

### 交互命令

| 命令 | 说明 |
|------|------|
| `create` | 创建新 vault（两次密码确认） |
| `open` | 打开/切换 vault 文件 |
| `set <key> <value>` | 设置条目 |
| `get <key>` | 获取条目值 |
| `delete <key>` | 删除条目 |
| `list` | 列出所有有效条目 |
| `raw` | 显示所有原始 Entry（含已删除条目，显示序号/时间戳/类型/Key/Value） |
| `passwd` | 修改密码 |
| `compact` | 压缩 vault（清除历史/已删除条目） |
| `help` | 显示帮助 |
| `quit` / `exit` | 退出 |

### 交互特性

- Tab 补全：第一级补全命令名，第二级对 get/delete/set 补全已有 key
- 上下箭头浏览命令历史（仅内存）
- 支持 UTF-8 多字节字符输入
- 密码输入使用 term.ReadPassword()，不回显
- 终端 raw 模式，所有输出使用 \r\n 换行

### raw 输出格式

```
#N  <timestamp>  <type(N)>  <key>  <value>
```

- type: clear(0) / text(1) / blob(2)
- clear 条目 value 显示 `-`
- blob 条目 value 显示 `[blob N bytes]`

### 导入工具

```
tapass-import <keepass.xml> <output.tap>
```
