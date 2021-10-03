package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/masakichi/tango/dict"
	"github.com/masakichi/tango/utils"
	_ "modernc.org/sqlite"
)

var (
	appName   = "tango"
	version   = "1.2.0-dev"
	gitCommit string
	buildDate string
)

var (
	dataDir      = utils.GetDataDir(appName)
	databasePath = filepath.Join(dataDir, "tango.db")
)

func initDatabase() *sql.DB {
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		os.MkdirAll(dataDir, os.ModePerm)
	}
	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stat(databasePath); os.IsNotExist(err) {
		dict.InitDictDB(db)
	}
	return db
}

func main() {

	listFlag := flag.Bool("list", false, "list all dictionaries")
	importFlag := flag.String("import", "", "import yomichan's dictionary zip file")
	versionFlag := flag.Bool("version", false, "print tango app version")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s: A CLI Japanese Dictionary Tool\n", appName)
		fmt.Fprintf(os.Stderr, "Usage: %s %s\n\n", os.Args[0], "単語")
		fmt.Fprintf(os.Stderr, "Other Commands of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if len(os.Args) < 2 {
		flag.Usage()
		return
	}

	db := initDatabase()
	defer db.Close()

	if !strings.HasPrefix(os.Args[1], "-") {
		dict.PrintTerms(db)
		return
	}

	if *importFlag != "" {
		dict.ImportDictDB(db, *importFlag)
		return
	}

	if *listFlag {
		dict.PrintDicts(db)
		return
	}

	if *versionFlag {
		if gitCommit != "" {
			version += "-" + gitCommit
		}
		if buildDate == "" {
			buildDate = "unknown"
		}
		osArch := runtime.GOOS + "/" + runtime.GOARCH
		versionString := fmt.Sprintf("%s %s %s BuildDate=%s",
			appName, version, osArch, buildDate)
		fmt.Println(versionString)
	}
}
