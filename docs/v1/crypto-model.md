# V1 加密模型设计

## 概述

V1 采用 Argon2id 密钥派生 + HKDF 子密钥拆分 + DEFLATE（原始 flate 流）压缩 + XChaCha20-Poly1305 AEAD 加密方案。

## 密钥派生

### Argon2id

- 输出长度：32 字节（Master Key）
- Salt：头部存储的 32 字节随机盐值
- 参数：由头部 Time Cost / Memory Cost / Parallelism 字段指定

### 参数约束与可解析性

Argon2id 参数（Time Cost / Memory Cost / Parallelism）存储在明文头部，任何设备打开时按声明参数派生密钥，
因此参数必须可被**精确执行**，否则不同实现实际使用的参数不同、派生出的密钥不同，表现为误导性的"密码错误"。

必须满足（否则拒绝解析，返回 invalid kdf params）：

- `Time Cost >= 1`
- `1 <= Parallelism <= 255`（argon2 的 threads 为 uint8，超出即无法表达）
- `Memory Cost >= 8 * Parallelism`，且 `Memory Cost` 是 `4 * Parallelism` 的整数倍
  （argon2 按 `4 * parallelism` 个 lane 划分并把内存向下对齐）

**不设参数上下限**：能否解析取决于本机可用内存（峰值内存 ≈ Memory Cost，另需运行时开销）。
内存不足时返回明确错误（需要 X MiB / 可用 Y MiB），不得 OOM 或 panic。
内存探测：Linux 读 `/proc/meminfo` 的 `MemAvailable`；其他平台暂不探测。详见 `docs/platforms.md`。

### KDF 参数变更

参数变更**必须重新派生密钥**（新 Salt + 新 Nonce + 按新参数重新派生 + 重新加密），
流程与改密相同（`Rekey`），且必须提供当前主密码。仅修改头部参数而不重新派生会导致 vault 永久无法打开。

### HKDF 子密钥拆分

Master Key 通过 HKDF-SHA256（RFC 5869）拆分为两路子密钥，**严格按以下 5 步执行**。
注意第 3 步的第二次 Extract 不可省略：若直接对第 1 步的 PRK 做两次 Expand
（即"一次 Extract + 按 info 两次 Expand"），会得到**完全不同的子密钥**，实现间无法互通。

| 步骤 | 操作 | 说明 |
|------|------|------|
| 1 | `prk1 = HKDF-Extract(salt = 头部 Salt, IKM = Master Key)` | salt 为头部 32 字节随机盐 |
| 2 | `okm64 = HKDF-Expand(prk1, info = ""（空）, L = 64)` | 中间密钥材料 |
| 3 | `prk2 = HKDF-Extract(salt = 32 字节全零, IKM = okm64)` | salt 为 hash 长度（32B）全零 |
| 4 | `HMAC Key = HKDF-Expand(prk2, info = "tapass-v1-hmac", L = 32)` | 用于 Header HMAC（主密码验证） |
| 5 | `Encrypt Key = HKDF-Expand(prk2, info = "tapass-v1-enc", L = 32)` | 用于 XChaCha20-Poly1305 |

- `info` 只参与 HKDF-Expand（Extract 不使用 info），因此第 4、5 步共用同一个 `prk2`
- 第 3 步的"32 字节全零 salt"等价于 Go `hkdf.New(sha256.New, okm64, nil, info)` 内部的 Extract 行为
- info 字符串按 ASCII 字节直接使用，不做编码转换

派生流程（编号同上）：

```
主密码 + Salt ──Argon2id(32B)──▶ Master Key
                                    │ ① Extract(salt = Salt)
                                    ▼
                                  prk1
                                    │ ② Expand(info = "", L = 64)
                                    ▼
                                  okm64
                                    │ ③ Extract(salt = 0x00 × 32)
                                    ▼
                                  prk2
                        ┌───────────┴───────────┐
     ④ Expand(info="tapass-v1-hmac")   ⑤ Expand(info="tapass-v1-enc")
                        ▼                       ▼
                HMAC Key (32B)          Encrypt Key (32B)
```

一致性校验：上述每一步的固定结果都记录在 `tools/vault/testdata/format_v1_vectors.json`，
并可由 `python3 docs/v1/verify_format_v1.py`（纯 Python 手写实现，不依赖参考实现代码）复算验证。

## 数据处理流程

### 加密流程

```
明文数据段 → DEFLATE 压缩（原始 flate 流）→ XChaCha20-Poly1305 加密 → 密文
```

### 解密流程

```
密文 → XChaCha20-Poly1305 解密 → DEFLATE 解压（原始 flate 流）→ 明文数据段
```

- 压缩在加密之前执行（密文不可压缩）
- 压缩流为**原始 flate（RFC 1951）**，不带 zlib 头与校验尾（RFC 1950）：与 `compress/flate` 一致，不使用 zlib 封装
- 压缩流**不要求跨实现逐字节一致**（不同压缩器/压缩级别的输出不同）：只要求解压还原后与原始数据段字节一致。
  测试向量因此固定"解压后的数据段字节"，而不固定压缩流字节
- Compression ID = 0 时跳过压缩/解压步骤；= 1 时使用 DEFLATE
- Compression ID 为其他值时**拒绝解析**（返回 unsupported compression id），
  不得静默按"无压缩"处理（会解出错误的数据段内容）

## 加密方案

### 算法

XChaCha20-Poly1305 (AEAD)

### 参数

- Key：Encrypt Key (32B)，由 HKDF 拆分获得
- Nonce：头部存储的 24 字节随机值
- AAD：无（Header MAC + Header HMAC 已独立保护头部完整性）

### 输出

| 组成 | 大小 | 说明 |
|------|------|------|
| Ciphertext | 变长 | 加密后的主体数据 |
| Auth Tag | 16B | Poly1305 认证标签，附加在密文末尾 |

## 头部校验

### Header MAC

- 算法：SHA-256
- 输入：`header[0:80]`（Magic + Version + Salt + Nonce + Argon2 参数 + Compression ID + Reserved）
- 输出：32 字节摘要
- 用途：检测头部数据是否损坏或被篡改（无密钥保护）

**安全说明**：Header MAC 不覆盖自身和 Header HMAC 字段（偏移 80-143）。攻击者可篡改这两个字段而不被 MAC 检测。但 Header HMAC 依赖于 MAC 值，篡改 MAC 会导致 HMAC 验证失败。因此头部完整性由 MAC + HMAC 联合保证，单独篡改任一字段都会被检测。

### Header HMAC

- 算法：HMAC-SHA256
- 密钥：HMAC Key（由 HKDF 拆分获得）
- 消息：Header MAC（32 字节）
- 输出：32 字节摘要
- 用途：验证主密码是否正确

## 改密流程（更换主密码）

1. 使用旧主密码完成验证与解密流程，获得明文数据段
2. 生成新的 Salt（32 字节）和新的 Nonce（24 字节）
3. 新主密码 + 新 Salt + Argon2 参数 → Argon2id → 新 Master Key
4. 新 Master Key + 新 Salt → HKDF-SHA256 → 新 HMAC Key + 新 Encrypt Key
5. 新 Encrypt Key + 新 Nonce → XChaCha20-Poly1305 加密压缩后的数据段
6. 计算新 Header MAC：`SHA256(header[0:80])`（使用新 Salt + 新 Nonce）
7. 计算新 Header HMAC：`HMAC-SHA256(新 MAC, 新 HMAC_Key)`
8. 组装新文件头部 + 新加密体，写入临时文件
9. 验证临时文件可正确解密后，原子替换原文件

Nonce 必须重新生成，不得复用旧 Nonce。

## 完整验证与解密流程

1. 读取头部，验证 Magic == `TAPASS`，Version == 1
2. 计算并验证 `SHA256(header[0:80])` == Header MAC（头部完整性）
3. 主密码 + Salt + Argon2 参数 → Argon2id → Master Key (32B)
4. Master Key + Salt → HKDF-SHA256 → HMAC Key (32B) + Encrypt Key (32B)
5. 验证 `HMAC-SHA256(MAC, HMAC_Key)` == Header HMAC（主密码校验）
6. Encrypt Key + Nonce → XChaCha20-Poly1305 解密主体
7. Poly1305 Auth Tag 验证密文完整性
8. 根据 Compression ID 解压数据（0=跳过, 1=DEFLATE 解压）
