package orgedit

import (
	"crypto/rand"
	"fmt"
	"os"
	"time"
)

// Env は操作が外から受け取る値。テストで固定できるよう、時刻と ID の生
// 成をここに集める。
type Env struct {
	Now   time.Time
	NewID func() string
	// Home は ARCHIVE_FILE で ~ と略す home ディレクトリ。
	Home string
}

// DefaultEnv は実行時の Env を返す。
func DefaultEnv() Env {
	home, _ := os.UserHomeDir()
	return Env{Now: time.Now(), NewID: newUUID, Home: home}
}

// newUUID は org-id-new の既定(org-id-method が uuid)と同じく、小文
// 字の UUID v4 を返す。
func newUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// inactiveTimestamp は org の時刻付きの非アクティブなタイムスタンプ
// [2006-01-02 Mon 15:04] を返す。
func inactiveTimestamp(t time.Time) string {
	return t.Format("[2006-01-02 Mon 15:04]")
}
