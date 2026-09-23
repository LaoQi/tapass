#!/usr/bin/env python3
"""tapass v1 格式独立校验（不依赖参考实现代码）。

用规范文档 + 测试向量手写 HKDF-SHA256 与 XChaCha20-Poly1305，
按 docs/v1/data-structures.md 的偏移解析头部与记录，验证：
  1. HKDF 5 步派生链（prk1 → okm64 → prk2 → HMAC Key / Encrypt Key）
  2. 头部字段偏移/字节序/MAC 规则/HMAC 规则
  3. 数据段记录布局（timestamp/type/key_length/key/value_length/value）
  4. XChaCha20-Poly1305 密文体（AAD 为空）
用法: python3 docs/v1/verify_format_v1.py [向量 JSON 路径]
"""
import binascii
import hashlib
import hmac
import json
import struct
import sys

MASK = 0xFFFFFFFF
SIGMA = (0x61707865, 0x3320646E, 0x79622D32, 0x6B206574)


def rotl(v, n):
    return ((v << n) | (v >> (32 - n))) & MASK


def qround(s, a, b, c, d):
    s[a] = (s[a] + s[b]) & MASK; s[d] = rotl(s[d] ^ s[a], 16)
    s[c] = (s[c] + s[d]) & MASK; s[b] = rotl(s[b] ^ s[c], 12)
    s[a] = (s[a] + s[b]) & MASK; s[d] = rotl(s[d] ^ s[a], 8)
    s[c] = (s[c] + s[d]) & MASK; s[b] = rotl(s[b] ^ s[c], 7)


def chacha20_block(key, counter, nonce12):
    state = list(SIGMA) + list(struct.unpack("<8I", key)) + [counter] + list(struct.unpack("<3I", nonce12))
    work = list(state)
    for _ in range(10):
        qround(work, 0, 4, 8, 12); qround(work, 1, 5, 9, 13)
        qround(work, 2, 6, 10, 14); qround(work, 3, 7, 11, 15)
        qround(work, 0, 5, 10, 15); qround(work, 1, 6, 11, 12)
        qround(work, 2, 7, 8, 13); qround(work, 3, 4, 9, 14)
    return struct.pack("<16I", *[(work[i] + state[i]) & MASK for i in range(16)])


def hchacha20(key, nonce16):
    state = list(SIGMA) + list(struct.unpack("<8I", key)) + list(struct.unpack("<4I", nonce16))
    for _ in range(10):
        qround(state, 0, 4, 8, 12); qround(state, 1, 5, 9, 13)
        qround(state, 2, 6, 10, 14); qround(state, 3, 7, 11, 15)
        qround(state, 0, 5, 10, 15); qround(state, 1, 6, 11, 12)
        qround(state, 2, 7, 8, 13); qround(state, 3, 4, 9, 14)
    return struct.pack("<8I", state[0], state[1], state[2], state[3],
                       state[12], state[13], state[14], state[15])


def chacha20_xor(data, key, nonce12, counter=1):
    out = bytearray()
    for off in range(0, len(data), 64):
        ks = chacha20_block(key, counter + off // 64, nonce12)
        chunk = data[off:off + 64]
        out += bytes(a ^ b for a, b in zip(chunk, ks))
    return bytes(out)


def poly1305(msg, key):
    r = int.from_bytes(key[:16], "little") & 0x0ffffffc0ffffffc0ffffffc0fffffff
    s = int.from_bytes(key[16:32], "little")
    p = (1 << 130) - 5
    acc = 0
    for off in range(0, len(msg), 16):
        block = msg[off:off + 16]
        acc = ((acc + int.from_bytes(block + b"\x01", "little")) * r) % p
    return ((acc + s) % (1 << 128)).to_bytes(16, "little")


def pad16(b):
    return b if len(b) % 16 == 0 else b + b"\x00" * (16 - len(b) % 16)


def xchacha20_poly1305_encrypt(key, nonce24, plaintext, aad=b""):
    subkey = hchacha20(key, nonce24[:16])
    nonce12 = b"\x00" * 4 + nonce24[16:24]
    otk = chacha20_block(subkey, 0, nonce12)[:32]
    ct = chacha20_xor(plaintext, subkey, nonce12, counter=1)
    mac_data = pad16(aad) + pad16(ct) + struct.pack("<QQ", len(aad), len(ct))
    return ct + poly1305(mac_data, otk)


def hkdf_extract(salt, ikm):
    if not salt:
        salt = b"\x00" * 32
    return hmac.new(salt, ikm, hashlib.sha256).digest()


def hkdf_expand(prk, info, length):
    out, t, i = b"", b"", 1
    while len(out) < length:
        t = hmac.new(prk, t + info + bytes([i]), hashlib.sha256).digest()
        out += t
        i += 1
    return out[:length]


def parse_header(b):
    assert len(b) == 144, f"header 必须 144 字节，实际 {len(b)}"
    return {
        "magic": b[0:6],
        "version": struct.unpack_from("<H", b, 6)[0],
        "salt": b[8:40],
        "nonce": b[40:64],
        "time_cost": struct.unpack_from("<I", b, 64)[0],
        "memory_cost": struct.unpack_from("<I", b, 68)[0],
        "parallelism": struct.unpack_from("<I", b, 72)[0],
        "compression_id": b[76],
        "reserved": b[77:80],
        "mac": b[80:112],
        "hmac": b[112:144],
        "mac_input": b[0:80],
    }


def parse_records(b):
    recs, off = [], 0
    while off < len(b):
        ts, = struct.unpack_from("<Q", b, off); off += 8
        typ = b[off]; off += 1
        klen, = struct.unpack_from("<H", b, off); off += 2
        key = b[off:off + klen]; off += klen
        vlen, = struct.unpack_from("<I", b, off); off += 4
        val = b[off:off + vlen]; off += vlen
        recs.append({"timestamp": ts, "type": typ, "key": key.decode("utf-8"), "value": val, "value_length": vlen})
    return recs


def main():
    path = sys.argv[1] if len(sys.argv) > 1 else "tools/vault/testdata/format_v1_vectors.json"
    v = json.load(open(path))
    inp, exp = v["inputs"], v["expected"]
    checks = []

    def check(name, ok, detail=""):
        checks.append((name, ok, detail))

    salt = bytes.fromhex(inp["salt_hex"])
    nonce = bytes.fromhex(inp["nonce_hex"])
    master = bytes.fromhex(exp["master_key_hex"])

    # 1) HKDF 派生链（Argon2id 无纯 Python 实现，以向量中的 master key 为输入）
    okm64 = hkdf_expand(hkdf_extract(salt, master), b"", 64)
    check("HKDF okm64 = Expand(Extract(salt, master), \"\", 64)", okm64.hex() == exp["hkdf_okm64_hex"])
    prk2 = hkdf_extract(b"\x00" * 32, okm64)
    check("HKDF prk2 = Extract(32 字节零, okm64)", prk2.hex() == exp["hkdf_prk2_hex"])
    hmac_key = hkdf_expand(prk2, b"tapass-v1-hmac", 32)
    enc_key = hkdf_expand(prk2, b"tapass-v1-enc", 32)
    check("HMAC Key = Expand(prk2, \"tapass-v1-hmac\", 32)", hmac_key.hex() == exp["hmac_key_hex"])
    check("Encrypt Key = Expand(prk2, \"tapass-v1-enc\", 32)", enc_key.hex() == exp["encrypt_key_hex"])

    # 2) 头部按规范偏移解析 + MAC/HMAC 规则
    hdr_bytes = bytes.fromhex(exp["header_hex"])
    h = parse_header(hdr_bytes)
    check("header 为 144 字节", len(hdr_bytes) == 144)
    check("magic = TAPASS", h["magic"] == b"TAPASS")
    check("version = 1", h["version"] == 1)
    check("salt/nonce 与输入一致", h["salt"] == salt and h["nonce"] == nonce)
    check("argon2 参数与输入一致",
          (h["time_cost"], h["memory_cost"], h["parallelism"]) ==
          (inp["argon2"]["time_cost"], inp["argon2"]["memory_cost_kib"], inp["argon2"]["parallelism"]))
    check("compression_id = 1 (DEFLATE)", h["compression_id"] == 1)
    check("reserved 置零", h["reserved"] == b"\x00\x00\x00")
    check("Header MAC = SHA256(header[0:80])", hashlib.sha256(h["mac_input"]).digest() == h["mac"])
    check("Header HMAC = HMAC-SHA256(MAC, hmac_key)",
          hmac.new(hmac_key, h["mac"], hashlib.sha256).digest() == h["hmac"])

    # 3) 数据段记录布局
    recs = parse_records(bytes.fromhex(exp["data_segment_hex"]))
    want = inp["entries"]
    check("记录条数与向量一致", len(recs) == len(want), f"{len(recs)} vs {len(want)}")
    for r, w in zip(recs, want):
        exp_val = bytes.fromhex(w.get("value_hex", "")) if "value_hex" in w else w.get("value_utf8", "").encode()
        check(f"记录 {w['key']} 布局",
              (r["timestamp"], r["type"], r["key"], r["value"]) == (w["timestamp"], w["type"], w["key"], exp_val))
    check("type=0 记录 value_length = 0", all(r["value_length"] == 0 for r in recs if r["type"] == 0))

    # 4) XChaCha20-Poly1305 密文体（AAD 空）
    data_segment = bytes.fromhex(exp["data_segment_hex"])
    ct = xchacha20_poly1305_encrypt(enc_key, nonce, data_segment)
    check("XChaCha20-Poly1305 密文体（AAD 空，含 tag）", ct.hex() == exp["aead_ciphertext_hex"])

    failed = [c for c in checks if not c[1]]
    for name, ok, detail in checks:
        print(f"{'PASS' if ok else 'FAIL'}  {name}" + (f"  [{detail}]" if detail and not ok else ""))
    print(f"\n{len(checks) - len(failed)}/{len(checks)} 通过")
    if failed:
        sys.exit(1)
    print("独立校验（HKDF / 头部 / 记录布局 / AEAD）全部通过")


if __name__ == "__main__":
    main()
