# V1 数据结构设计

## 二进制文件布局

### 头部（明文，144 字节）

除 Magic 外全部采用小端序。

| 偏移 | 大小（字节） | 字段 | 类型 | 说明 |
|------|-------------|------|------|------|
| 0 | 6 | Magic | ASCII | `TAPASS` |
| 6 | 2 | Version | uint16 LE | 当前为 1 |
| 8 | 32 | Salt | bytes | Argon2id 盐值 |
| 40 | 24 | Nonce | bytes | XChaCha20-Poly1305 nonce |
| 64 | 4 | Time Cost | uint32 LE | Argon2id time_cost |
| 68 | 4 | Memory Cost | uint32 LE | Argon2id memory_cost |
| 72 | 4 | Parallelism | uint32 LE | Argon2id parallelism |
| 76 | 1 | Compression ID | uint8 | 0=无压缩, 1=DEFLATE |
| 77 | 3 | Reserved | bytes | 保留，置零 |
| 80 | 32 | Header MAC | bytes | `SHA256(header[0:80])` |
| 112 | 32 | Header HMAC | bytes | `HMAC-SHA256(MAC, HMAC_Key)` |

### 加密体（密文）

| 大小 | 字段 | 说明 |
|------|------|------|
| 变长 | Ciphertext | XChaCha20-Poly1305 加密的主体数据（压缩后） |
| 16 | Auth Tag | Poly1305 认证标签，附加在密文末尾 |

### 数据段（解压后的 Ciphertext 内容）

数据段由多条 KV 条目紧密排列组成，追加式写入。

#### 单条记录结构

| 大小（字节） | 字段 | 类型 | 说明 |
|-------------|------|------|------|
| 8 | Timestamp | uint64 LE | 毫秒级 Unix 时间戳 |
| 1 | Type | uint8 | 0=清空, 1=文本, 2=blob |
| 2 | Key Length | uint16 LE | key UTF-8 字节长度 |
| 变长 | Key | bytes | key 值，UTF-8 编码 |
| 4 | Value Length | uint32 LE | value 字节长度 |
| 变长 | Value | bytes | value 值（Type=0 时 Value Length=0，无 Value） |

条目之间紧密排列，无对齐、无填充。

#### 读取语义

- 同一 key 多次写入时，取时间戳最新的一条
- Type=0 表示该 key 已删除/清空，读取时视为不存在
- 全量扫描线性解析，无需索引

#### 垃圾回收（Compact）

追加式 KV 中，删除和更新会产生无效条目。V1 不强制规定 compact 策略，由实现自行决定，例如：

- 基于时间清理：仅保留最近 N 天内的有效条目
- 全量压缩：剔除所有历史版本和已删除条目，仅保留每个 key 的最新有效值

compact 操作需重写整个加密体（重新加密），流程同改密中的文件替换策略。

## 实现层 Key 处理规则

### Key 路径拆分

- Key 以 `/` 为分隔符拆分为路径组件
- 根为 `/`，最后一项为属性名，前面的组件为路径层级
- 路径深度灵活，无固定层级

示例：

| Key | 路径 | 属性名 |
|-----|------|--------|
| `/vault1/entry1/PASSWD` | `vault1/entry1` | `PASSWD` |
| `/vault1/folder1/entry2/PASSWD` | `vault1/folder1/entry2` | `PASSWD` |
| `/vault1/entry1/username` | `vault1/entry1` | `username` |

### 特殊属性名

以下属性名在实现层有特殊处理逻辑，属性名无保留限制，用户自定义属性可使用大写名称，若与特殊属性同名则享受对应处理。

| 属性名 | 特殊处理 |
|--------|---------|
| `PASSWD` | UI 密码专属控件（隐藏显示、一键复制等） |
| `SSH` | SSH 密钥对格式解析（公钥/私钥） |
| `TOTP` | 值存储 TOTP secret，实现层动态生成验证码 |

## 验证与解密流程

1. 读取头部，验证 Magic == `TAPASS`，Version == 1
2. 计算并验证 `SHA256(header[0:80])` == Header MAC（头部完整性）
3. Argon2id 派生 Master Key → HKDF 展开为 HMAC Key + Encrypt Key
4. 验证 `HMAC-SHA256(MAC, HMAC_Key)` == Header HMAC（主密码校验）
5. Encrypt Key + Nonce → XChaCha20-Poly1305 解密主体，Poly1305 验证密文完整性
6. 根据 Compression ID 解压数据（0=跳过, 1=DEFLATE 解压）

## 字节序

- Magic（ASCII）：大端序
- 其余所有多字节字段：小端序（Little-Endian）
