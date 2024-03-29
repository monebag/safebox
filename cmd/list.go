package cmd

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/adikari/safebox/v2/config"
	"github.com/adikari/safebox/v2/store"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// runCmd represents the exec command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Lists all the configs available",
	RunE:  list,
}

var (
	sortByModified bool
	sortByVersion  bool
)

func init() {
	listCmd.Flags().BoolVarP(&sortByModified, "modified", "m", false, "sort by modified time")
	listCmd.Flags().BoolVarP(&sortByVersion, "version", "v", false, "sort by version")

	rootCmd.AddCommand(listCmd)
}

func list(_ *cobra.Command, _ []string) error {
	config, err := loadConfig()

	if err != nil {
		return errors.Wrap(err, "failed to load config")
	}

	store, err := store.GetStore(store.StoreConfig{
		Provider: config.Provider,
		Region:   config.Region,
		FilePath: config.Filepath,
	})

	if err != nil {
		return errors.Wrap(err, "failed to instantiate store")
	}

	configs, err := store.GetMany(config.All)

	if err != nil {
		return errors.Wrap(err, "failed to list params")
	}

	if sortByVersion {
		sort.Sort(ByVersion(configs))
	} else if sortByModified {
		sort.Sort(ByModified(configs))
	} else {
		sort.Sort(ByName(configs))
	}

	printList(configs, config)

	return nil
}

func printList(configs []store.Config, cfg *config.Config) {
	if len(configs) <= 0 {

		PrintSummary(Summary{
			Message: "Total parameters = 0",
			Config:  *cfg,
		})

		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, '\t', 0)

	fmt.Fprint(w, "Name\tValue\tType\tVersion\tLastModified")
	fmt.Fprintln(w, "")

	for _, config := range configs {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s",
			*config.Name,
			*config.Value,
			config.Type,
			config.Version,
			config.Modified.Local().Format(TimeFormat),
		)

		fmt.Fprintln(w, "")
	}
	fmt.Fprintln(w, "---")

	PrintSummary(Summary{
		Message: fmt.Sprintf("Total parameters = %d", len(configs)),
		Config:  *cfg,
	})

	w.Flush()
}

type ByName []store.Config

func (a ByName) Len() int           { return len(a) }
func (a ByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByName) Less(i, j int) bool { return *a[i].Name < *a[j].Name }

type ByVersion []store.Config

func (a ByVersion) Len() int           { return len(a) }
func (a ByVersion) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByVersion) Less(i, j int) bool { return a[i].Version < a[j].Version }

type ByModified []store.Config

func (a ByModified) Len() int           { return len(a) }
func (a ByModified) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByModified) Less(i, j int) bool { return a[i].Modified.Before(a[j].Modified) }
