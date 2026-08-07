package db

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL embed.FS

func Open(dbPath string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	fresh := false
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fresh = true
	}

	// _pragma=foreign_keys(1) 对 modernc 不一定生效，打开后再 PRAGMA
	dsn := dbPath + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(4)
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := ensureSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	if fresh {
		if err := seedDictionaries(db); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("seed dictionaries: %w", err)
		}
	} else {
		// 已有库也确保字典存在（空库或旧文件）
		if err := seedDictionariesIfEmpty(db); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	return db, nil
}

func ensureSchema(db *sql.DB) error {
	raw, err := schemaSQL.ReadFile("schema.sql")
	if err != nil {
		return err
	}
	if _, err := db.Exec(string(raw)); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}

func seedDictionariesIfEmpty(db *sql.DB) error {
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM egg_groups`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	return seedDictionaries(db)
}

func seedDictionaries(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	eggs := []string{
		"无法孵蛋", "巨灵", "两栖", "昆虫", "天空",
		"动物", "妖精", "植物", "拟人", "软体",
		"大地", "魔力", "海洋", "巨龙", "机械",
	}
	for i, name := range eggs {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO egg_groups (id, name) VALUES (?, ?)`, i+1, name); err != nil {
			return err
		}
	}

	natures := [][3]string{
		{"沉默", "生命", "物攻"}, {"平和", "生命", "魔攻"}, {"忧郁", "生命", "物防"}, {"粗心", "生命", "魔防"}, {"踏实", "生命", "速度"},
		{"逞强", "物攻", "生命"}, {"固执", "物攻", "魔攻"}, {"大胆", "物攻", "物防"}, {"调皮", "物攻", "魔防"}, {"勇敢", "物攻", "速度"},
		{"理性", "魔攻", "生命"}, {"聪明", "魔攻", "物攻"}, {"专注", "魔攻", "物防"}, {"偏执", "魔攻", "魔防"}, {"冷静", "魔攻", "速度"},
		{"坦率", "物防", "生命"}, {"稳重", "物防", "物攻"}, {"天真", "物防", "魔攻"}, {"懒散", "物防", "魔防"}, {"悠闲", "物防", "速度"},
		{"焦虑", "魔防", "生命"}, {"警惕", "魔防", "物攻"}, {"害羞", "魔防", "魔攻"}, {"温顺", "魔防", "物防"}, {"慎重", "魔防", "速度"},
		{"热情", "速度", "生命"}, {"胆小", "速度", "物攻"}, {"开朗", "速度", "魔攻"}, {"急躁", "速度", "物防"}, {"莽撞", "速度", "魔防"},
	}
	for i, row := range natures {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO natures (id, name, boost, penalty) VALUES (?,?,?,?)`, i+1, row[0], row[1], row[2]); err != nil {
			return err
		}
	}

	medals := [][2]string{
		{"体型", "大块头"}, {"体型", "小不点"}, {"声音", "婉转音"}, {"声音", "粗嗓门"},
	}
	for _, m := range medals {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO medals (type, name) VALUES (?, ?)`, m[0], m[1]); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ResolveDBPath：优先 DB_PATH；否则按常见布局查找/创建。
func ResolveDBPath() string {
	if v := strings.TrimSpace(os.Getenv("DB_PATH")); v != "" {
		return v
	}
	exe, err := os.Executable()
	if err != nil {
		return filepath.Join("data", "rkw_hatcher.db")
	}
	exe, _ = filepath.EvalSymlinks(exe)
	dir := filepath.Dir(exe)
	lowExe := strings.ToLower(exe)
	lowDir := strings.ToLower(dir)

	candidates := []string{
		filepath.Join(dir, "rkw_hatcher.db"),
		filepath.Join(dir, "data", "rkw_hatcher.db"),
		filepath.Join("data", "rkw_hatcher.db"),
	}
	if strings.EqualFold(filepath.Base(dir), "bin") {
		candidates = append([]string{filepath.Join(filepath.Dir(dir), "data", "rkw_hatcher.db")}, candidates...)
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			abs, e := filepath.Abs(c)
			if e == nil {
				return abs
			}
			return c
		}
	}

	if strings.Contains(lowExe, "go-build") || strings.Contains(lowDir, string(filepath.Separator)+"temp"+string(filepath.Separator)) || strings.Contains(lowDir, `\temp\`) {
		p, _ := filepath.Abs(filepath.Join("data", "rkw_hatcher.db"))
		return p
	}
	if strings.EqualFold(filepath.Base(dir), "bin") {
		p, _ := filepath.Abs(filepath.Join(filepath.Dir(dir), "data", "rkw_hatcher.db"))
		return p
	}
	return filepath.Join(dir, "rkw_hatcher.db")
}
