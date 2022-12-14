package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/adikari/safebox/v2/store"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

const doubleQuoteSpecialChars = "\\\n\r\"!$`"

var (
	exportFormat string
	outputFile   string
	keysToExport []string

	exportCmd = &cobra.Command{
		Use:     "export",
		Short:   "Exports all configuration to a file",
		RunE:    export,
		Example: `TODO: export command example`,
	}
)

func init() {
	exportCmd.Flags().StringVarP(&exportFormat, "format", "f", "json", "output format (json, yaml, dotenv)")
	exportCmd.Flags().StringVarP(&outputFile, "output-file", "o", "", "output file (default is standard output)")
	exportCmd.Flags().StringSliceVarP(&keysToExport, "key", "k", []string{}, "only export specified config (default is export all)")
	exportCmd.MarkFlagFilename("output-file")

	rootCmd.AddCommand(exportCmd)
}

func export(cmd *cobra.Command, args []string) error {
	config, err := loadConfig()

	if err != nil {
		return errors.Wrap(err, "failed to load config")
	}

	store, err := store.GetStore(config.Provider)

	if err != nil {
		return errors.Wrap(err, "failed to instantiate store")
	}

	toExport, err := configsToExport(config.All)

	if err != nil {
		return err
	}

	configs, err := store.GetMany(toExport)

	if err != nil {
		return errors.Wrap(err, "failed to get params")
	}

	file := os.Stdout
	if outputFile != "" {
		if file, err = os.OpenFile(outputFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
			return errors.Wrap(err, "Failed to open output file for writing")
		}
		defer file.Close()
		defer file.Sync()
	}
	w := bufio.NewWriter(file)
	defer w.Flush()

	params := map[string]string{}
	for _, c := range configs {
		params[c.Key()] = *c.Value
	}

	switch strings.ToLower(exportFormat) {
	case "json":
		err = exportAsJson(params, w)
	case "yaml":
		err = exportAsYaml(params, w)
	case "dotenv":
		err = exportAsEnvFile(params, w)
	default:
		err = errors.Errorf("unsupported export format: %s", exportFormat)
	}

	if err != nil {
		return errors.Wrap(err, "failed to export parameters")
	}

	return nil
}

func exportAsEnvFile(params map[string]string, w io.Writer) error {
	for _, k := range sortedKeys(params) {
		key := strings.ToUpper(k)
		key = strings.Replace(key, "-", "_", -1)
		w.Write([]byte(fmt.Sprintf(`%s="%s"`+"\n", key, doubleQuoteEscape(params[k]))))
	}
	return nil
}

func exportAsJson(params map[string]string, w io.Writer) error {
	d, err := json.MarshalIndent(params, "", "  ")
	if err != nil {
		return err
	}
	w.Write([]byte(d))
	return nil
}

func exportAsYaml(params map[string]string, w io.Writer) error {
	return yaml.NewEncoder(w).Encode(params)
}

func sortedKeys(params map[string]string) []string {
	keys := make([]string, len(params))
	i := 0
	for k := range params {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
}

func doubleQuoteEscape(line string) string {
	for _, c := range doubleQuoteSpecialChars {
		toReplace := "\\" + string(c)
		if c == '\n' {
			toReplace = `\n`
		}
		if c == '\r' {
			toReplace = `\r`
		}
		line = strings.Replace(line, string(c), toReplace, -1)
	}
	return line
}

func configsToExport(configs []store.ConfigInput) ([]store.ConfigInput, error) {
	if len(keysToExport) == 0 {
		return configs, nil
	}

	result := []store.ConfigInput{}

	for _, key := range keysToExport {
		var found bool

		for _, config := range configs {
			if config.Key() == key {
				found = true
				result = append(result, config)
				break
			}
		}

		if !found {
			return nil, errors.Errorf("key '%s' is not found in safebox config file", key)
		}
	}

	return result, nil
}
