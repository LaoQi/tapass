package model

import (
	"fmt"
	"testing"

	"github.com/LaoQi/tapass/tools/vault"
)

// benchDB 构造 n 条属性的 DB（不经 Argon2，直接内存构造）。
func benchDB(n int) *DB {
	entries := make(map[string]vault.Entry, n)
	for i := 0; i < n; i++ {
		key := fmt.Sprintf("/grp%02d/entry%03d/PASSWD", i%20, i)
		entries[key] = vault.Entry{
			Key:       key,
			Type:      vault.TypeText,
			Value:     []byte("secret"),
			Timestamp: uint64(i + 1),
		}
	}
	return newTestDB(entries)
}

// BenchmarkNavigateGroup 模拟一次 j/k 导航：查询某分组下的属性并逐个读取
// （对应 mainview.queryAttributes 的调用路径）。
func BenchmarkNavigateGroup(b *testing.B) {
	db := benchDB(2000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		keys := db.QueryKeys("/grp00")
		for _, k := range keys {
			if _, ok := db.Get(k); !ok {
				b.Fatal("missing entry")
			}
		}
	}
}

func BenchmarkGet(b *testing.B) {
	db := benchDB(2000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok := db.Get("/grp03/entry003/PASSWD"); !ok {
			b.Fatal("missing entry")
		}
	}
}

func BenchmarkQueryKeys(b *testing.B) {
	db := benchDB(2000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.QueryKeys("/grp00")
	}
}
