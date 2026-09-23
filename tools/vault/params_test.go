package vault

import (
	"errors"
	"testing"
)

func TestValidateArgon2Params(t *testing.T) {
	cases := []struct {
		name    string
		params  Argon2Params
		wantErr bool
	}{
		{"default", DefaultArgon2Params, false},
		{"time cost 0", Argon2Params{TimeCost: 0, MemoryCost: 16384, Parallelism: 1}, true},
		{"parallelism 0", Argon2Params{TimeCost: 6, MemoryCost: 16384, Parallelism: 0}, true},
		{"parallelism 256 (uint8 overflow)", Argon2Params{TimeCost: 6, MemoryCost: 16384, Parallelism: 256}, true},
		{"parallelism 255 ok (lane 1020 KiB)", Argon2Params{TimeCost: 6, MemoryCost: 2040, Parallelism: 255}, false},
		{"memory below 8*parallelism", Argon2Params{TimeCost: 6, MemoryCost: 7, Parallelism: 1}, true},
		{"memory not lane aligned", Argon2Params{TimeCost: 6, MemoryCost: 16385, Parallelism: 1}, true},
		{"memory not aligned for p=2", Argon2Params{TimeCost: 6, MemoryCost: 1002, Parallelism: 2}, true},
		{"memory >= 4 TiB still structurally valid", Argon2Params{TimeCost: 1, MemoryCost: 0xFFFFFF00, Parallelism: 1}, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateArgon2Params(c.params)
			if c.wantErr && err == nil {
				t.Errorf("expected error for %+v", c.params)
			}
			if !c.wantErr && err != nil {
				t.Errorf("unexpected error for %+v: %v", c.params, err)
			}
			if c.wantErr && err != nil && !errors.Is(err, ErrInvalidKDFParams) {
				t.Errorf("expected ErrInvalidKDFParams, got %v", err)
			}
		})
	}
}

// 解析必须拒绝无法精确执行的参数，且不能 panic（历史缺陷：uint8 截断 → argon2 panic）。
func TestUnmarshalHeaderRejectsInvalidParams(t *testing.T) {
	tamper := []struct {
		name   string
		modify func(*Header)
	}{
		{"parallelism 0", func(h *Header) { h.Argon2.Parallelism = 0 }},
		{"parallelism 256", func(h *Header) { h.Argon2.Parallelism = 256 }},
		{"parallelism 512", func(h *Header) { h.Argon2.Parallelism = 512 }},
		{"time cost 0", func(h *Header) { h.Argon2.TimeCost = 0 }},
		{"memory not aligned", func(h *Header) { h.Argon2.MemoryCost = 16385 }},
	}

	for _, c := range tamper {
		t.Run(c.name, func(t *testing.T) {
			data, err := Create("password")
			if err != nil {
				t.Fatalf("Create failed: %v", err)
			}
			h, err := UnmarshalHeader(data[:HeaderSize])
			if err != nil {
				t.Fatal(err)
			}
			c.modify(h)
			h.computeAndSetMAC()
			hdrBytes, err := h.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			file := append(hdrBytes, data[HeaderSize:]...)

			// Open 必须返回错误，绝不允许 panic
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Open panicked instead of returning an error: %v", r)
				}
			}()
			if _, err := Open(file, "password"); err == nil {
				t.Fatalf("expected error, got nil")
			} else if !errors.Is(err, ErrInvalidKDFParams) {
				t.Fatalf("expected ErrInvalidKDFParams, got %v", err)
			}
		})
	}
}

// 参数超出本机能力时必须返回明确错误，而不是 OOM / panic。
func TestOpenRejectsParamsExceedingLocalMemory(t *testing.T) {
	data, err := Create("password")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	h, err := UnmarshalHeader(data[:HeaderSize])
	if err != nil {
		t.Fatal(err)
	}
	h.Argon2 = Argon2Params{TimeCost: 1, MemoryCost: 0xFFFFFF00, Parallelism: 1} // ~4 TiB
	h.computeAndSetMAC()
	hdrBytes, err := h.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	file := append(hdrBytes, data[HeaderSize:]...)

	if _, err := Open(file, "password"); !errors.Is(err, ErrInsufficientMemory) {
		t.Fatalf("expected ErrInsufficientMemory, got %v", err)
	}
}

// 写入侧兜底：头部参数与密钥派生上下文不一致时拒绝序列化。
func TestMarshalBinaryDetectsParamTamper(t *testing.T) {
	data, err := Create("password")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	v, err := Open(data, "password")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := v.MarshalBinary(); err != nil {
		t.Fatalf("baseline MarshalBinary failed: %v", err)
	}

	v.Hdr.Argon2.TimeCost++
	if _, err := v.MarshalBinary(); !errors.Is(err, ErrKDFParamsChanged) {
		t.Fatalf("expected ErrKDFParamsChanged, got %v", err)
	}
	// 校验不会破坏内存态：参数改回后可继续写出
	v.Hdr.Argon2.TimeCost--
	if _, err := v.MarshalBinary(); err != nil {
		t.Fatalf("MarshalBinary after rollback failed: %v", err)
	}
}

// 改参数只能通过 Rekey：新参数/新密码生效、条目不丢、上下文一致。
func TestRekey(t *testing.T) {
	data, err := Create("old-pass")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	v, err := Open(data, "old-pass")
	if err != nil {
		t.Fatal(err)
	}
	v.Set("/grp/entry/PASSWD", []byte("secret"))

	newParams := Argon2Params{TimeCost: 2, MemoryCost: 8192, Parallelism: 1}
	newData, err := v.Rekey("old-pass", "new-pass", newParams)
	if err != nil {
		t.Fatalf("Rekey failed: %v", err)
	}

	// 旧密码失效，新密码 + 新参数可用
	if _, err := Open(newData, "old-pass"); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("expected ErrWrongPassword for old password, got %v", err)
	}
	reopened, err := Open(newData, "new-pass")
	if err != nil {
		t.Fatalf("Open with new password failed: %v", err)
	}
	if got := reopened.Params(); got != newParams {
		t.Errorf("params not applied: got %+v want %+v", got, newParams)
	}
	val, ok := reopened.Get("/grp/entry/PASSWD")
	if !ok || string(val) != "secret" {
		t.Errorf("entry lost after rekey: ok=%v value=%q", ok, val)
	}

	// Rekey 后上下文一致，可直接继续写出
	if _, err := reopened.MarshalBinary(); err != nil {
		t.Errorf("MarshalBinary after rekey failed: %v", err)
	}
	if got := v.DerivedParams(); got != newParams {
		t.Errorf("derived context not updated: got %+v want %+v", got, newParams)
	}

	// 非法参数直接拒绝，且不改动库
	if _, err := v.Rekey("new-pass", "new-pass", Argon2Params{TimeCost: 0, MemoryCost: 8192, Parallelism: 1}); !errors.Is(err, ErrInvalidKDFParams) {
		t.Errorf("expected ErrInvalidKDFParams, got %v", err)
	}
}

// 嵌入式设备场景：不设参数上下限，能否解析由本机可用内存决定。
func TestCheckArgon2ResourceOnConstrainedDevice(t *testing.T) {
	const mib = 1 << 20
	cases := []struct {
		name      string
		params    Argon2Params
		avail     uint64
		wantError bool
	}{
		// 可用内存 <32 MiB 的设备（目标平台之一）
		{"20 MiB avail, default 16 MiB params", paramsDefault(), 20 * mib, true},
		{"24 MiB avail, default 16 MiB params (16+8=24)", paramsDefault(), 24 * mib, false},
		{"20 MiB avail, 8 MiB params", params8MiB(), 20 * mib, false},
		{"8 MiB avail, 8 MiB params", params8MiB(), 8 * mib, true},
		// PC 场景：高参数照常通过
		{"8 GiB avail, 1 GiB params", Argon2Params{TimeCost: 4, MemoryCost: 1 << 20, Parallelism: 1}, 8 << 30, false},
		// 无上限：参数极大时由能力判定拒绝
		{"8 GiB avail, 4 TiB params", Argon2Params{TimeCost: 1, MemoryCost: 0xFFFFFF00, Parallelism: 1}, 8 << 30, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := ValidateArgon2Params(c.params); err != nil {
				t.Fatalf("params should be structurally valid: %v", err)
			}
			err := checkArgon2Resource(c.params, c.avail)
			if c.wantError && !errors.Is(err, ErrInsufficientMemory) {
				t.Errorf("expected ErrInsufficientMemory, got %v", err)
			}
			if !c.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// paramsDefault 返回默认（16 MiB）参数
func paramsDefault() Argon2Params { return DefaultArgon2Params }

// params8MiB 返回适配可用内存 <32 MiB 设备的 8 MiB 参数
func params8MiB() Argon2Params {
	return Argon2Params{TimeCost: 3, MemoryCost: 8192, Parallelism: 1}
}

// 运行环境内存探测（Linux 有效；其他平台允许未知）
func TestAvailableMemoryBytes(t *testing.T) {
	avail, ok := availableMemoryBytes()
	if !ok {
		t.Skip("memory probing unavailable on this platform")
	}
	if avail == 0 {
		t.Fatal("expected non-zero available memory")
	}
	t.Logf("available memory: %d MiB", avail>>20)
}
