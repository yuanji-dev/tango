package cmd

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/masakichi/tango/dict"
	"github.com/masakichi/tango/utils"
	"github.com/spf13/cobra"

	_ "modernc.org/sqlite"
)

var (
	appName    = "tango"
	appVersion = "1.2.0-dev"
	gitCommit  string
	buildDate  string
)

var (
	dataDir = utils.GetDataDir(appName)
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tango [単語|word]",
	Short: "Tango(単語) is a CLI Japanese dictionary tool",
	Long: `Tango(単語) is a CLI Japanese dictionary tool

It uses yomichan's dictionary files and works completely offline.
  `,
	Run: func(cmd *cobra.Command, args []string) {

		dict.InitDatabase(dataDir)

		// Define input word
		if len(args) > 0 {
			terms, err := dict.DefineWord(args[0])
			if err != nil {
				log.Fatal(err)
			}
			for _, t := range terms {
				fmt.Printf("%s(%s)\n[%s]\n", t.Expression, t.Reading, t.Dict)
				fmt.Println(strings.TrimSuffix(strings.Join(t.Glossary, "\n"), "\n") + "\n")
			}
			return
		}

		// Import dictionary
		if cmd.Flags().Changed("import") {
			dictPath, _ := cmd.Flags().GetString("import")
			dict.ImportDictDB(dictPath)
			return
		}

		// List all imported dictionaries
		if listFlag, _ := cmd.Flags().GetBool("list"); listFlag {
			dicts, err := dict.AllDicts()
			if err != nil {
				log.Fatal(err)
			}
			if len(dicts) == 0 {
				fmt.Println("There is no dictionary being imported yet.")
			}
			for _, d := range dicts {
				fmt.Printf("[%d] %s Format: %d, Revision: %s\n", d.ID, d.Title, d.Format, d.Revision)
			}
			return
		}

		cmd.Help()

	},
	Version: getVersion(),
}

func getVersion() string {
	version := appVersion
	if gitCommit != "" {
		version += "-" + gitCommit
	}
	if buildDate == "" {
		buildDate = "unknown"
	}
	osArch := runtime.GOOS + "/" + runtime.GOARCH
	return fmt.Sprintf("%s %s BuildDate=%s",
		version, osArch, buildDate)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("list", "l", false, "list all dictionaries")
	rootCmd.Flags().StringP("import", "i", "", "import yomichan's dictionary zip file")
}
