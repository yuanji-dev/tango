package dict

import (
	"archive/zip"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
)

var TangoDB *sql.DB

// https://github.com/FooSoft/yomichan-import/blob/5fe039e5f66ccad397f97a44a9f406a5a68a9438/common.go
//type dbTerm struct {
//	Expression     string
//	Reading        string
//	DefinitionTags []string
//	Rules          []string
//	Score          int
//	Glossary       []string
//	Sequence       int
//	TermTags       []string
//}
type Term struct {
	Expression string
	Reading    string
	Glossaries []string
	Dict       string
}

type Dict struct {
	ID       int
	Title    string `json:"title"`
	Format   int    `json:"format"`
	Revision string `json:"revision"`
}

func (t *Term) UnmarshalJSON(data []byte) error {
	var v []interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	t.Expression = v[0].(string)
	t.Reading = v[1].(string)
	//t.Glossaries = v[5].([]string)
	t.Glossaries = func(gs []interface{}) []string {
		result := []string{}
		for _, g := range gs {
			result = append(result, g.(string))
		}
		return result
	}(v[5].([]interface{}))
	return nil
}

func InitDictDB() {
	TangoDB.Exec(`CREATE TABLE IF NOT EXISTS terms (id INTEGER PRIMARY KEY, expression TEXT, reading TEXT, glossaries TEXT, dict_id INTEGER)`)
	TangoDB.Exec(`CREATE INDEX IF NOT EXISTS idx_expr ON terms (expression)`)
	TangoDB.Exec(`CREATE INDEX IF NOT EXISTS idx_read ON terms (reading)`)
	TangoDB.Exec(`CREATE TABLE IF NOT EXISTS dicts (id INTEGER PRIMARY KEY, title TEXT, format INTEGER, revision TEXT)`)
}

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
	row := TangoDB.QueryRow(`SELECT title FROM dicts WHERE title = ?`, dict.Title)
	row.Scan(&dictTitle)
	if dictTitle != "" {
		log.Fatal("dict already exist")
	}

	tx, _ := TangoDB.Begin()

	rv, err := tx.Exec(
		`INSERT INTO dicts (title, format, revision) VALUES (?,?,?)`,
		dict.Title, dict.Format, dict.Revision,
	)
	if err != nil {
		log.Fatal(err)
	}
	dictID, _ := rv.LastInsertId()

	termList := []Term{}
	for _, f := range r.File {
		if !strings.HasPrefix(f.Name, "term_bank_") {
			continue
		}
		rc, _ := f.Open()
		content, _ := ioutil.ReadAll(rc)

		var v []Term
		json.Unmarshal(content, &v)
		termList = append(termList, v...)
		rc.Close()
	}
	canCommit := true
	for _, t := range termList {
		glossaries, _ := json.Marshal(t.Glossaries)
		_, err := tx.Exec(
			`INSERT INTO terms (expression, reading, glossaries, dict_id) VALUES (?,?,?,?)`,
			t.Expression, t.Reading, string(glossaries), dictID,
		)
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

func GetAllDicts() ([]Dict, error) {
	rows, err := TangoDB.Query(
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

func DefineWord(w string) ([]Term, error) {
	rows, err := TangoDB.Query(
		`SELECT expression, reading, glossaries, d.title
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
		var glossaries string
		rows.Scan(&t.Expression, &t.Reading, &glossaries, &t.Dict)
		json.Unmarshal([]byte(glossaries), &t.Glossaries)
		result = append(result, t)
	}
	return result, nil
}
