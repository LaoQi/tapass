# AGENTS.md — tapass-tools

tapass CLI 工具集 + 核心加密库。

## 构建 & 测试

```bash
go build ./...                                    # 编译
go test ./vault/ -v -timeout 120s             # vault 测试（Argon2id 慢，需 timeout）
go test ./internal/importer/ -v -timeout 120s  # importer 测试
go vet ./...                                      # 静态检查
go run ./cmd/tapass/                              # 运行 CLI
go run ./cmd/tapass-import/                       # 运行导入工具
```

## 包结构

```
cmd/tapass/main.go           # CLI CRUD 工具入口
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

## 实现约定

- Argon2 默认参数：time=6, memory=16384 (16 MiB), parallelism=1
- 压缩使用 flate 裸 DEFLATE 流（非 zlib），与设计文档 "zlib/DEFLATE" 描述不同，为本项目明确选择
- write() 每次写入重新生成 Nonce
- 文件写入使用临时文件 + 原子替换策略
- SubKeys.Zero() 安全清零密钥

## CLI 用法

```
tapass create <file> <password>
tapass set    <file> <password> <key> <value>
tapass get    <file> <password> <key>
tapass delete <file> <password> <key>
tapass list   <file> <password>

tapass-import <keepass.xml> <output.tap>
```
