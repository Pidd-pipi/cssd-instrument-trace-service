package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewBacksUpCorruptFileAndStartsEmpty(t *testing.T) {
	file := filepath.Join(t.TempDir(), "store.json")
	if err := os.WriteFile(file, []byte(`{"packs":`), 0o600); err != nil {
		t.Fatal(err)
	}

	st, err := New(file)
	if err != nil {
		t.Fatalf("损坏文件应降级为空库启动: %v", err)
	}
	if st.CountPacks() != 0 {
		t.Fatalf("损坏文件降级后 packs 应为空，实际 %d", st.CountPacks())
	}
	if _, err := os.Stat(file + ".bak"); err != nil {
		t.Fatalf("应生成备份文件 store.json.bak: %v", err)
	}
}

func TestAtomicWriteFileCreatesDirectory(t *testing.T) {
	file := filepath.Join(t.TempDir(), "nested", "store.json")
	if err := atomicWriteFile(file, []byte(`{"ok":true}`)); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"ok":true}` {
		t.Fatalf("文件内容不一致: %s", data)
	}
}
