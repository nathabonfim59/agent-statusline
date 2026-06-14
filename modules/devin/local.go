package devin

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type LocalData struct {
	SessionID string
	Model     string
	CWD       string
	Title     string
}

type ModelConfigs map[string]int

func dataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "devin", "cli")
}

func loadLocalData() (*LocalData, ModelConfigs) {
	ld := &LocalData{}
	models := ModelConfigs{}

	dir := dataDir()

	// Read model configs from cache
	models = parseModelConfigs(filepath.Join(os.Getenv("HOME"), ".cache", "devin", "cli", "model_configs_v2.bin"))

	// Read active session from sessions.db
	dbPath := filepath.Join(dir, "sessions.db")
	db, err := sql.Open("sqlite3", dbPath+"?mode=ro")
	if err != nil {
		return ld, models
	}
	defer db.Close()

	row := db.QueryRow(`
		SELECT id, model, working_directory, COALESCE(title, '')
		FROM sessions
		WHERE hidden = 0
		ORDER BY last_activity_at DESC
		LIMIT 1
	`)
	if err := row.Scan(&ld.SessionID, &ld.Model, &ld.CWD, &ld.Title); err != nil {
		return ld, models
	}

	return ld, models
}

func parseModelConfigs(path string) ModelConfigs {
	models := ModelConfigs{}
	data, err := os.ReadFile(path)
	if err != nil {
		return models
	}

	pos := 0
	for pos < len(data) {
		tag, vb := readVarint(data, pos)
		if vb == 0 {
			break
		}
		pos += vb
		_ = tag >> 3
		wt := tag & 0x7
		if wt == 2 {
			length, lb := readVarint(data, pos)
			pos += lb
			if pos+length <= len(data) {
				sub := data[pos : pos+length]
				pos += length
				modelID, ctxSize := parseModelEntry(sub)
				if modelID != "" && ctxSize > 0 {
					models[modelID] = ctxSize
				}
			}
		} else if wt == 0 {
			_, vv := readVarint(data, pos)
			pos += vv
		} else if wt == 1 {
			pos += 8
		} else if wt == 5 {
			pos += 4
		} else {
			break
		}
	}
	return models
}

func parseModelEntry(data []byte) (modelID string, ctxSize int) {
	pos := 0
	for pos < len(data) {
		tag, vb := readVarint(data, pos)
		if vb == 0 {
			break
		}
		pos += vb
		fn := tag >> 3
		wt := tag & 0x7
		if wt == 0 {
			val, vv := readVarint(data, pos)
			pos += vv
			if fn == 18 {
				ctxSize = val
			}
		} else if wt == 2 {
			length, lb := readVarint(data, pos)
			pos += lb
			if fn == 22 && pos+length <= len(data) {
				modelID = string(data[pos : pos+length])
			}
			pos += length
		} else if wt == 1 {
			pos += 8
		} else if wt == 5 {
			pos += 4
		} else {
			break
		}
	}
	return
}

func readVarint(data []byte, pos int) (int, int) {
	value := 0
	shift := 0
	br := 0
	for pos+br < len(data) {
		b := data[pos+br]
		br++
		value |= int(b&0x7F) << shift
		if b&0x80 == 0 {
			return value, br
		}
		shift += 7
		if shift >= 64 {
			break
		}
	}
	return 0, 0
}