package dict

import (
	"archive/zip"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var db *sql.DB

// Term is type used for definition of a word
// https://github.com/FooSoft/yomichan-import/blob/5fe039e5f66ccad397f97a44a9f406a5a68a9438/common.go
type Term struct {
	Expression     string
	Reading        string
	DefinitionTags string
	Rules          string
	Score          int
	Glossary       []string
	Sequence       int
	TermTags       string
	Dict           string
}

type rawTermList [][]interface{}

// Dict stores metadata of a yomichan dictionary
type Dict struct {
	ID        int
	Title     string `json:"title"`
	Format    int    `json:"format"`
	Revision  string `json:"revision"`
	Sequenced bool   `json:"sequenced"`
}

// InitDatabase sets up the global db object
func InitDatabase(dataDir string) error {
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		os.MkdirAll(dataDir, os.ModePerm)
	}
	databasePath := filepath.Join(dataDir, "tango.db")
	var err error
	db, err = sql.Open("sqlite", databasePath)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stat(databasePath); os.IsNotExist(err) {
		initDictDB()
	}
	return db.Ping()
}

func initDictDB() {
	db.Exec(`CREATE TABLE IF NOT EXISTS terms 
          (id INTEGER PRIMARY KEY,
           expression TEXT,
           reading TEXT,
           definition_tags TEXT,
           rules TEXT,
           score INTEGER,
           glossary TEXT,
           sequence INTEGER,
           term_tags INTEGER,
           dict_id INTEGER)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_expr ON terms (expression)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_read ON terms (reading)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS dicts (id INTEGER PRIMARY KEY, title TEXT, format INTEGER, revision TEXT, sequenced BOOLEAN)`)
}

// ImportDictDB import a yomichan zip format dictionary into sqlite database
func ImportDictDB(dictName string) {
	r, err := zip.OpenReader(dictName)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()
	var dict Dict
	for _, f := range r.File {
		if f.Name == "index.json" {
			rc, _ := f.Open()
			content, _ := ioutil.ReadAll(rc)
			json.Unmarshal(content, &dict)
			rc.Close()
		}
	}
	if &dict == nil {
		log.Fatal("bad dict")
	}

	var dictTitle string
	row := db.QueryRow(`SELECT title FROM dicts WHERE title = ?`, dict.Title)
	row.Scan(&dictTitle)
	if dictTitle != "" {
		log.Fatal("dict already exist")
	}

	tx, _ := db.Begin()

	rv, err := tx.Exec(
		`INSERT INTO dicts (title, format, revision, sequenced) VALUES (?,?,?,?)`,
		dict.Title, dict.Format, dict.Revision, dict.Sequenced,
	)
	if err != nil {
		log.Fatal(err)
	}
	dictID, _ := rv.LastInsertId()

	termList := rawTermList{}
	for _, f := range r.File {
		if !strings.HasPrefix(f.Name, "term_bank_") {
			continue
		}
		rc, _ := f.Open()
		content, _ := ioutil.ReadAll(rc)

		var v rawTermList
		json.Unmarshal(content, &v)
		termList = append(termList, v...)
		rc.Close()
	}
	canCommit := true
	for _, t := range termList {
		//glossaries, _ := json.Marshal(t.Glossaries)
		t[5], err = json.Marshal(t[5])
		_, err := tx.Exec(
			`INSERT INTO terms (expression, reading, definition_tags, rules, score, glossary, sequence, term_tags, dict_id) VALUES (?,?,?,?,?,json(?),?,?,?)`,
			append(t, dictID)...)
		if err != nil {
			fmt.Println(err)
			canCommit = false
			break
		}
	}
	if canCommit {
		tx.Commit()
		fmt.Println("import done")
	} else {
		tx.Rollback()
	}
}

// AllDicts returns all imported dictionaries
func AllDicts() ([]Dict, error) {
	rows, err := db.Query(
		`SELECT id, title, format, revision
		 FROM dicts
		 ORDER BY id ASC`,
	)
	if err != nil {
		return nil, err
	}
	var result []Dict
	for rows.Next() {
		var d Dict
		rows.Scan(&d.ID, &d.Title, &d.Format, &d.Revision)
		result = append(result, d)
	}
	return result, nil
}

// DefineWord searchs definition by expression or reading
func DefineWord(w string) ([]Term, error) {
	rows, err := db.Query(
		`SELECT expression, reading, definition_tags, rules, score, glossary, sequence, term_tags, d.title
		 FROM terms t
		 LEFT JOIN dicts d on t.dict_id = d.id
		 WHERE expression = ? OR reading = ?`,
		w, w,
	)
	if err != nil {
		return nil, err
	}
	var result []Term
	for rows.Next() {
		var t Term
		var _glossary []byte
		rows.Scan(&t.Expression, &t.Reading, &t.DefinitionTags, &t.Rules, &t.Score, &_glossary, &t.Sequence, &t.TermTags, &t.Dict)
		if err := json.Unmarshal(_glossary, &t.Glossary); err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, nil
}
