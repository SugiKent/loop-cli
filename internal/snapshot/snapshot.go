// Package snapshot は前回の取得結果（Card 群と保存時刻）を JSON で往復する。
// 中身は GitHub から作り直せる派生データだけで、認証情報を含まない（D-002）。
package snapshot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/SugiKent/sugi-loop/internal/model"
)

// Snapshot は前回の取得結果。At はその取得が完了した（保存した）時刻。
type Snapshot struct {
	Cards []model.Card
	At    time.Time
}

// DefaultPath はスナップショットの既定パス $HOME/.cache/sugi-loop/snapshot.json を返す。
// os.UserCacheDir は macOS で ~/Library/Caches になり D-002 のパスと一致しないので使わない。
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "sugi-loop", "snapshot.json"), nil
}

// Save は path に Snapshot を書く。ディレクトリが無ければ作る。
func Save(path string, s Snapshot) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// Load は path の Snapshot を読む。無い・読めない・壊れているはどれもエラーで、区別しない。
// ファイルは消さない（次の取得成功で上書きされる）。
func Load(path string) (Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Snapshot{}, err
	}
	var s Snapshot
	if err := json.Unmarshal(data, &s); err != nil {
		return Snapshot{}, err
	}
	return s, nil
}
