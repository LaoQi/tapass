package importer

import (
	"fmt"
	"os"
)

// ensureOutputAbsent 在导入前检查输出文件：已存在则拒绝覆盖（避免误删已有 vault）。
func ensureOutputAbsent(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("output file already exists: %s (remove it or choose another path)", path)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat output file: %w", err)
	}
	return nil
}

// writeVaultFile 原子写入：先写临时文件再改名，避免写入中断留下半截 vault。
func writeVaultFile(path string, data []byte) error {
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("write tmp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename tmp file: %w", err)
	}
	return nil
}
