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

type Key struct {
	ID    int
	Key   string
	Memo  string
	Usage int
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

	_, err = km.loadKeys()
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

// 从数据库查询所有的数据，并且返回 Key数组

func (km *KeyManager) loadKeys() ([]Key, error) {
	rows, err := km.db.Query("SELECT id,key,memo,usage FROM keys")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []string
	var keyData []Key
	for rows.Next() {
		var k Key
		err = rows.Scan(&k.ID, &k.Key, &k.Memo, &k.Usage)
		if err != nil {
			return nil, err
		}
		keys = append(keys, k.Key)
		keyData = append(keyData, k)
	}

	km.mu.Lock()
	defer km.mu.Unlock()
	km.validKeys = keys
	return keyData, nil
}

func (km *KeyManager) CreateKey(memo, key string) error {

	_, err := km.db.Exec("INSERT INTO keys (key, memo ) VALUES (?, ?)", key, memo)
	if err != nil {
		return err
	}

	_, err2 := km.loadKeys()
	return err2
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

	_, err2 := km.loadKeys()
	return err2
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
func (km *KeyManager) AllocateKeyHandler(w http.ResponseWriter, r *http.Request) {
	key, err := km.AllocateKey()
	if err != nil {
		http.Error(w, "No key available", http.StatusServiceUnavailable)
		return
	}
	// 返回JSON，包含  code=0，key
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf(`{"code":0,"key":"%s"}`, key)))

}

func (km *KeyManager) CreateKeyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		r.ParseForm()
		key := r.FormValue("key")
		memo := r.FormValue("memo")
		err := km.CreateKey(memo, key)
		if err != nil {
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
	keyList, err := km.loadKeys()
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

	t := template.Must(template.New("keys").Parse(tmpl))
	t.Execute(w, keyList)
}
