package main

import (
	"database/sql"
	"flag"
	"fmt"
	"github.com/masakichi/tango/dict"
	"github.com/masakichi/tango/utils"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	appName = "tango"
	version = "1.0.0"
)

var (
	dataDir      = utils.GetDataDir(appName)
	databasePath = filepath.Join(dataDir, "tango.db")
)

func initDatabase() {
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		os.MkdirAll(dataDir, os.ModePerm)
	}
	db, err := sql.Open("sqlite3", databasePath)
	if err != nil {
		log.Fatal(err)
	} else {
		dict.TangoDB = db
	}
	if _, err := os.Stat(databasePath); os.IsNotExist(err) {
		dict.InitDictDB()
	}
}

func printTerms() {
	terms, err := dict.DefineWord(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	for _, t := range terms {
		fmt.Printf("%s(%s)\n[%s]\n", t.Expression, t.Reading, t.Dict)
		fmt.Println(strings.TrimSuffix(strings.Join(t.Glossaries, "\n"), "\n") + "\n")
	}
}

func printDicts() {
	dicts, err := dict.GetAllDicts()
	if err != nil {
		log.Fatal(err)
	}
	for _, d := range dicts {
		fmt.Printf("[%d] %s Format: %d, Revision: %s\n", d.ID, d.Title, d.Format, d.Revision)
	}
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("plz give me a word")
	}

	initDatabase()

	if !strings.HasPrefix(os.Args[1], "-") {
		printTerms()
		return
	}

	listFlag := flag.Bool("list", false, "list all dictionaries")
	importFlag := flag.String("import", "", "import yomichan's dictionary zip file")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s: A CLI Japanese Dictionary Tool\n", appName)
		fmt.Fprintf(os.Stderr, "Usage: %s %s\n\n", os.Args[0], "単語")
		fmt.Fprintf(os.Stderr, "Other Commands of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if *importFlag != "" {
		dict.ImportDictDB(*importFlag)
		return
	}

	if *listFlag {
		printDicts()
		return
	}
}
