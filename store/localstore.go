package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var _ Store = &LocalStore{}

type LocalStore struct {
	filename string
	dir      string
	path     string
	stage    string
}

type LocalStoreConfig struct {
	Directory string
	Filename  string
	Stage     string
}

func NewLocalStore(config LocalStoreConfig) (*LocalStore, error) {
	if config.Directory == "" {
		return nil, errors.New("invalid parameter: directory is required")
	}

	if config.Filename == "" {
		return nil, errors.New("invalid parameter: filename is required")
	}

	dir := filepath.Clean(config.Directory)

	filename := fmt.Sprintf("%s-%s", config.Stage, config.Filename)
	if config.Stage == "" {
		filename = fmt.Sprintf("%s", config.Filename)
	}

	store := &LocalStore{
		filename: config.Filename,
		dir:      dir,
		path:     filepath.Join(dir, filename),
		stage:    config.Stage,
	}

	if _, err := os.Stat(dir); err == nil {
		return store, nil
	}

	err := os.MkdirAll(dir, 0755)

	if err != nil {
		return nil, err
	}

	return store, nil
}

func (s *LocalStore) PutMany(input []ConfigInput) error {
	updates := []Config{}

	for _, c := range input {
		t := "String"

		if c.Secret == true {
			t = "SecureString"
		}

		now := time.Now()

		name := c.Name
		value := c.Value

		updates = append(updates, Config{
			Name:     &name,
			Value:    &value,
			Version:  "1",
			Type:     t,
			Created:  now,
			Modified: now,
		})
	}

	existing, _ := s.read()
	for _, e := range existing {
		found, i := find(*e.Name, updates)
		if found != nil {
			v, _ := strconv.Atoi(e.Version)
			found.Version = strconv.Itoa(v + 1)
			updates[i] = *found
		} else {
			updates = append(updates, e)
		}
	}

	return s.write(updates)
}

func (s *LocalStore) Put(input ConfigInput) error {
	return s.PutMany([]ConfigInput{input})
}

func (s *LocalStore) DeleteMany(input []ConfigInput) error {
	existing, _ := s.read()
	updates := []Config{}

	for _, e := range existing {
		found := false
		for _, i := range input {
			if i.Name == *e.Name {
				found = true
				break
			}
		}

		if !found {
			updates = append(updates, e)
		}
	}

	if err := s.write(updates); err != nil {
		return err
	}

	return nil
}

func (s *LocalStore) GetMany(input []ConfigInput) ([]Config, error) {
	if len(input) <= 0 {
		return []Config{}, nil
	}

	existing, err := s.read()

	if err != nil {
		return nil, nil
	}

	configs := []Config{}

	for _, i := range input {
		if found, _ := find(i.Name, existing); found != nil {
			configs = append(configs, *found)
		}
	}

	return configs, nil
}

func (s *LocalStore) Get(input ConfigInput) (*Config, error) {
	if configs, _ := s.GetMany([]ConfigInput{input}); configs != nil && len(configs) > 0 {
		return &configs[0], nil
	}

	return nil, nil
}

func (s *LocalStore) GetByPath(path string) ([]Config, error) {
	existing, _ := s.read()
	result := []Config{}

	for _, e := range existing {
		if strings.HasPrefix(*e.Name, path) {
			result = append(result, e)
		}
	}

	return result, nil
}

// Read a record from json file
func (s *LocalStore) read() ([]Config, error) {
	if _, err := stat(s.path); err != nil {
		return []Config{}, err
	}

	b, err := ioutil.ReadFile(s.path)

	if err != nil {
		return []Config{}, err
	}

	configs := []Config{}

	err = json.Unmarshal(b, &configs)

	if err != nil {
		return nil, errors.New("failed to parse data in database")
	}

	return configs, nil
}

func (s *LocalStore) write(configs []Config) error {
	b, err := json.MarshalIndent(configs, "", "\t")

	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(s.path, b, 0644); err != nil {
		return err
	}

	return nil
}

func find(id string, all []Config) (*Config, int) {
	for i, c := range all {
		if *c.Name == id {
			return &c, i
		}
	}

	return nil, -1
}

func stat(path string) (fi os.FileInfo, err error) {
	if fi, err = os.Stat(path); os.IsNotExist(err) {
		fi, err = os.Stat(path)
	}

	return
}
