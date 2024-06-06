package keymanager

import (
	"database/sql"
	"fmt"
	"net/http"
	"sync"
	"text/template"

	_ "modernc.org/sqlite"
)

type KeyManager struct {
	db        *sql.DB
	mu        sync.Mutex
	validKeys []string
	index     int
}

func NewKeyManager(dbPath string) (*KeyManager, error) {
	db, err := sql.Open("sqlite", dbPath) // 使用 "sqlite" 驱动
	if err != nil {
		return nil, err
	}

	km := &KeyManager{
		db:        db,
		validKeys: []string{},
		index:     -1,
	}

	err = km.initDB()
	if err != nil {
		return nil, err
	}

	err = km.loadKeys()
	if err != nil {
		return nil, err
	}

	return km, nil
}

func (km *KeyManager) initDB() error {
	_, err := km.db.Exec(`CREATE TABLE IF NOT EXISTS keys (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		key TEXT UNIQUE,
		memo TEXT,
		usage INTEGER DEFAULT 0
	)`)
	return err
}

func (km *KeyManager) loadKeys() error {
	rows, err := km.db.Query("SELECT key FROM keys")
	if err != nil {
		return err
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return err
		}
		keys = append(keys, key)
	}

	km.mu.Lock()
	defer km.mu.Unlock()
	km.validKeys = keys
	return nil
}

func (km *KeyManager) CreateKey(memo, key string) error {

	_, err := km.db.Exec("INSERT INTO keys (key, memo ) VALUES (?, ?)", key, memo)
	if err != nil {
		return err
	}

	return km.loadKeys()
}

func (km *KeyManager) GetAllKeys() ([]string, error) {
	km.mu.Lock()
	defer km.mu.Unlock()
	return km.validKeys, nil
}

func (km *KeyManager) DeleteKey(key string) error {

	_, err := km.db.Exec("DELETE FROM keys WHERE key = ?", key)
	if err != nil {
		return err
	}

	return km.loadKeys()
}

func (km *KeyManager) AllocateKey() (string, error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	if len(km.validKeys) == 0 {
		return "", sql.ErrNoRows
	}

	km.index++
	if km.index >= len(km.validKeys) {
		km.index = 0
	}

	return km.validKeys[km.index], nil
}

// 增加使用数量
func (km *KeyManager) IncreaseUsage(key string, usage int) (string, error) {
	_, err := km.db.Exec("UPDATE keys SET usage = usage + ? WHERE key = ?", usage, key)
	if err != nil {
		return "", err
	}
	return key, nil
}

func (km *KeyManager) CreateKeyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		r.ParseForm()
		key := r.FormValue("key")
		memo := r.FormValue("memo")
		err := km.CreateKey(memo, key)
		if err != nil {
			fmt.Print("床啊进错误", err)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	tmpl := `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Create Key</title>
		</head>
		<body>
			<h1>Create Key</h1>
			<form method="POST" action="/create_key">
				Key: <input type="text" name="key"><br>
				Memo: <input type="text" name="memo"><br>
				<input type="submit" value="Create">
			</form>
		</body>
		</html>
	`
	fmt.Fprint(w, tmpl)
}
func (km *KeyManager) RemoveKeyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		r.ParseForm()
		key := r.FormValue("key")
		err := km.DeleteKey(key)
		if err != nil {
			http.Error(w, "无法删除", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	http.Error(w, "无效操作", http.StatusInternalServerError)
}
func (km *KeyManager) ListKeysHandler(w http.ResponseWriter, r *http.Request) {
	keys, err := km.GetAllKeys()
	if err != nil {
		http.Error(w, "Failed to get keys", http.StatusInternalServerError)
		return
	}

	tmpl := `
		<!DOCTYPE html>
		<html>
		<head>
			<title>List Keys</title>
		</head>
		<h3>所有KEY</h3>
		<h4>图片生成模型:30</h4>
		<h4>文本生成模型:20</h4>
		<a href="/create_key">创建新KEY</a>

		<body>
			<h1>All Keys</h1>
			<table border="1">
				<tr>
					<th>Key</th>
					<th>备注</th>
					<th>用量</th>
					<th>操作</th>
				</tr>
				{{range .}}
				<tr>
					<td>{{.Key}}</td>
					<td>{{.Memo}}</td>
					<td>{{.Usage}}</td>
					<td>
						<form method="POST" action="/remove_key" style="display:inline;">
							<input type="hidden" name="key" value="{{.Key}}">
							<input type="submit" value="删除">
						</form>
					</td>
				</tr>
				{{end}}
			</table>
			<br>
		</body>
		</html>
	`

	type Key struct {
		ID    int
		Key   string
		Memo  string
		Usage int
	}
	var keyList []Key
	for _, key := range keys {
		keyList = append(keyList, Key{Key: key})
	}
	t := template.Must(template.New("keys").Parse(tmpl))
	t.Execute(w, keyList)
}
