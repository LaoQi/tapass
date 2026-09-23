package vault

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var (
	// ErrInvalidKDFParams 参数无法被本实现精确执行（含非法取值）。
	ErrInvalidKDFParams = errors.New("invalid kdf params")
	// ErrKDFParamsChanged 头部 KDF 参数与当前密钥的派生参数不一致（未经 Rekey 就改动）。
	ErrKDFParamsChanged = errors.New("kdf params changed without rekey")
	// ErrInsufficientMemory 本机当前没有足够内存按该参数解析。
	ErrInsufficientMemory = errors.New("insufficient local memory for kdf params")
	// ErrWrongPassword 主密码校验失败。
	ErrWrongPassword = errors.New("wrong password")
)

// argon2（x/crypto/argon2）把内存划分为 syncPoints*parallelism 个 lane：
// memory 会先向下对齐到该倍数，且不低于 2*syncPoints*parallelism（KiB）。
// 因此只有 memory 是 4*parallelism 的整数倍且不小于 8*parallelism 时，
// 声明值才会被精确执行；否则实际生效值与头部声明不同，跨实现派生出不同密钥
// （表现为误导性的"密码错误"），故直接拒绝解析。
const (
	argon2SyncPoints   = 4
	argon2LaneMultiple = argon2SyncPoints // memory 对齐倍数 = 4*parallelism
)

// kdfMemoryHeadroom 为 argon2 之外的开销预留的余量：Go 运行时堆基线、
// 主密码 / 盐 / 密钥与解析缓冲。判定"本机能力"时叠加在 argon2 峰值之上。
const kdfMemoryHeadroom = 8 << 20

// ValidateArgon2Params 校验参数能否被本实现精确执行。与本地资源无关。
func ValidateArgon2Params(p Argon2Params) error {
	if p.TimeCost < 1 {
		return fmt.Errorf("%w: time cost must be >= 1, got %d", ErrInvalidKDFParams, p.TimeCost)
	}
	if p.Parallelism < 1 || p.Parallelism > 255 {
		return fmt.Errorf("%w: parallelism must be in 1..255, got %d (argon2 threads is uint8)",
			ErrInvalidKDFParams, p.Parallelism)
	}

	lane := uint64(p.Parallelism) * argon2LaneMultiple
	memory := uint64(p.MemoryCost)
	if memory < 2*lane {
		return fmt.Errorf("%w: memory must be >= %d KiB for parallelism %d, got %d KiB",
			ErrInvalidKDFParams, 2*lane, p.Parallelism, p.MemoryCost)
	}
	if memory%lane != 0 {
		return fmt.Errorf("%w: memory must be a multiple of %d KiB for parallelism %d, got %d KiB",
			ErrInvalidKDFParams, lane, p.Parallelism, p.MemoryCost)
	}
	return nil
}

// CheckArgon2Resource 判断本机当前是否有能力按该参数完成派生。
// 不设参数上限：能否解析由本机可用内存决定。无法探测内存的平台不阻拦。
func CheckArgon2Resource(p Argon2Params) error {
	avail, ok := availableMemoryBytes()
	if !ok {
		return nil
	}
	return checkArgon2Resource(p, avail)
}

// checkArgon2Resource 按已知可用内存做判定，便于测试（不依赖运行环境）。
func checkArgon2Resource(p Argon2Params, availBytes uint64) error {
	need := uint64(p.MemoryCost) * 1024
	if need <= availBytes && availBytes-need >= kdfMemoryHeadroom {
		return nil
	}
	return fmt.Errorf("%w: argon2 needs %d MiB (memory %d KiB) + %d MiB overhead, local available %d MiB",
		ErrInsufficientMemory, need>>20, p.MemoryCost, kdfMemoryHeadroom>>20, availBytes>>20)
}

// availableMemoryBytes 返回本机当前可用于分配的字节数；ok=false 表示无法探测。
func availableMemoryBytes() (uint64, bool) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, false
	}

	var memAvailable, memFree, buffers, cached uint64
	var haveAvailable, haveFree bool
	for _, line := range strings.Split(string(data), "\n") {
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		kb, err := parseMeminfoKB(value)
		if err != nil {
			continue
		}
		switch key {
		case "MemAvailable":
			memAvailable, haveAvailable = kb, true
		case "MemFree":
			memFree, haveFree = kb, true
		case "Buffers":
			buffers = kb
		case "Cached":
			cached = kb
		}
	}

	switch {
	case haveAvailable:
		return memAvailable, true
	case haveFree:
		// 老内核无 MemAvailable（< 2.6.27）：退化为 MemFree + Buffers + Cached
		return memFree + buffers + cached, true
	default:
		return 0, false
	}
}

// parseMeminfoKB 解析 /proc/meminfo 中形如 "  12345 kB" 的值，返回字节数。
func parseMeminfoKB(value string) (uint64, error) {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return 0, fmt.Errorf("empty meminfo value")
	}
	kb, err := strconv.ParseUint(fields[0], 10, 64)
	if err != nil {
		return 0, err
	}
	return kb * 1024, nil
}
