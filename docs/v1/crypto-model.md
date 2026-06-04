# V1 加密模型设计

## 概述

V1 采用 Argon2id 密钥派生 + HKDF 子密钥拆分 + zlib/DEFLATE 压缩 + XChaCha20-Poly1305 AEAD 加密方案。

## 密钥派生

### Argon2id

- 输出长度：32 字节（Master Key）
- Salt：头部存储的 32 字节随机盐值
- 参数：由头部 Time Cost / Memory Cost / Parallelism 字段指定

### HKDF 子密钥拆分

Master Key 通过 HKDF-SHA256 拆分为两路子密钥：

- HKDF-Extract：Salt 复用头部 Salt
- HKDF-Expand：输出 64 字节

| Info | 输出 | 用途 |
|------|------|------|
| `tapass-v1-hmac` | HMAC Key (32B) | Header HMAC 计算，验证主密码 |
| `tapass-v1-enc` | Encrypt Key (32B) | XChaCha20-Poly1305 加解密 |

### 派生流程

```
主密码 + Salt → Argon2id(32B) → Master Key
                                    │
                              HKDF-SHA256(Salt)
                                    │
                              HKDF-Expand(64B)
                              ┌────┴────┐
                              ▼         ▼
                    info="tapass-v1-hmac"  info="tapass-v1-enc"
                         │                     │
                         ▼                     ▼
                  HMAC Key (32B)        Encrypt Key (32B)
```

## 数据处理流程

### 加密流程

```
明文数据段 → zlib/DEFLATE 压缩 → XChaCha20-Poly1305 加密 → 密文
```

### 解密流程

```
密文 → XChaCha20-Poly1305 解密 → zlib/DEFLATE 解压 → 明文数据段
```

- 压缩在加密之前执行（密文不可压缩）
- Compression ID = 0 时跳过压缩/解压步骤

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
