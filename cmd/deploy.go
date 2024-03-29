package cmd

import (
	"fmt"

	"github.com/adikari/safebox/v2/store"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	removeOrphans bool
	prompt        string

	deployCmd = &cobra.Command{
		Use:   "deploy",
		Short: "Deploys all configurations specified in config file",
		RunE:  deploy,
	}
)

func init() {
	rootCmd.AddCommand(deployCmd)
	deployCmd.Flags().BoolVarP(&removeOrphans, "remove-orphans", "r", false, "remove orphan configurations")
	deployCmd.Flags().StringVarP(&prompt, "prompt", "p", "", "prompt for configurations (missing or all)")
}

func deploy(_ *cobra.Command, _ []string) error {
	config, err := loadConfig()

	if prompt != "" && prompt != "all" && prompt != "missing" {
		return errors.New("value for prompt must be \"all\" or \"missing\"")
	}

	if err != nil {
		return errors.Wrap(err, "failed to load config")
	}

	st, err := store.GetStore(store.StoreConfig{
		Provider: config.Provider,
		Region:   config.Region,
		FilePath: config.Filepath,
	})

	if err != nil {
		return errors.Wrap(err, "failed to instantiate store")
	}

	all, err := st.GetMany(config.All)

	if err != nil {
		return errors.Wrap(err, "failed to read existing params")
	}

	missing := getMissing(config.Secrets, all)

	if len(missing) > 0 && prompt == "" {
		return errors.New("config values missing. run deploy with \"--prompt\" flag")
	}

	configsToDeploy := []store.ConfigInput{}

	// prompt for missing secrets
	if prompt == "missing" {
		for _, c := range missing {
			if c.Value == "" {
				configsToDeploy = append(configsToDeploy, promptConfig(c))
			}
		}
	}

	// prompt for all secrets and provide existing value as default
	if prompt == "all" {
		for _, c := range config.Secrets {
			var existingValue string
			for _, a := range all {
				if c.Name == *a.Name {
					existingValue = *a.Value
					c.Value = *a.Value
				}
			}

			userInput := promptConfig(c)

			if userInput.Value != existingValue {
				configsToDeploy = append(configsToDeploy, userInput)
			}
		}
	}

	// filter configs with changed values
	for _, c := range config.Configs {
		found := false
		for _, a := range all {
			if c.Name == *a.Name {
				found = true

				if c.Value != *a.Value {
					configsToDeploy = append(configsToDeploy, c)
				}
				break
			}
		}

		if !found {
			configsToDeploy = append(configsToDeploy, c)
		}
	}

	err = st.PutMany(configsToDeploy)

	if err != nil {
		return errors.Wrap(err, "failed to write params")
	}

	if removeOrphans {
		orphans, err := doRemoveOrphans(st, config.Prefix, config.All)
		if err != nil {
			fmt.Printf("%s\n", errors.Wrap(err, "Error: failed to remove orphan"))
		}

		fmt.Printf("orphans removed = %d.\n", len(orphans))
	}

	if len(config.Generate) > 0 {
		for _, t := range config.Generate {
			err := exportToFile(ExportParams{
				config: config,
				format: t.Type,
				output: t.Path,
			})

			if err != nil {
				fmt.Printf("Error: failed to generate file type = %s, output = %s\n", t.Type, t.Path)
				fmt.Printf("       %s\n", err)
				continue
			}

			fmt.Printf("wrote file -> %s\n", t.Path)
		}
	}

	PrintSummary(Summary{
		Message: fmt.Sprintf("%s = %d", "new configs", len(configsToDeploy)),
		Config:  *config,
	})

	return nil
}

func doRemoveOrphans(st store.Store, prefix string, all []store.ConfigInput) ([]store.ConfigInput, error) {
	var orphans []store.ConfigInput
	params, err := st.GetByPath(prefix)

	if err != nil {
		return nil, err
	}

	for _, param := range params {
		exists := false

		for _, config := range all {
			if config.Name == *param.Name {
				exists = true
				break
			}
		}

		if !exists {
			orphans = append(orphans, store.ConfigInput{Name: *param.Name})
		}
	}

	if err = st.DeleteMany(orphans); err != nil {
		return nil, err
	}

	return orphans, nil
}

func promptConfig(config store.ConfigInput) store.ConfigInput {
	validate := func(input string) error {
		if len(input) < 1 {
			return fmt.Errorf("%s must not be empty", config.Name)
		}
		return nil
	}

	prompt := promptui.Prompt{
		Label:    config.Key(),
		Validate: validate,
		Default:  config.Value,
	}

	result, _ := prompt.Run()

	config.Value = result

	return config
}

func getMissing(a []store.ConfigInput, b []store.Config) []store.ConfigInput {
	mb := make(map[string]struct{}, len(b))

	for _, x := range b {
		mb[*x.Name] = struct{}{}
	}

	var diff []store.ConfigInput
	for _, x := range a {
		if _, found := mb[x.Name]; !found {
			diff = append(diff, x)
		}
	}

	return diff
}
