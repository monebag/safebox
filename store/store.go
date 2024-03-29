package store

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/adikari/safebox/v2/aws"
	"github.com/adikari/safebox/v2/util"
	a "github.com/aws/aws-sdk-go/aws"
)

type Config struct {
	Name     *string
	Value    *string
	Modified time.Time
	Created  time.Time
	Version  string
	Type     string
	DataType string
}

type ConfigInput struct {
	Name        string
	Value       string
	Secret      bool
	Description string
}

var (
	ConfigNotFoundError = errors.New("config not found")
)

type Store interface {
	PutMany(input []ConfigInput) error
	Get(input ConfigInput) (*Config, error)
	GetMany(inputs []ConfigInput) ([]Config, error)
	GetByPath(path string) ([]Config, error)
	DeleteMany(inputs []ConfigInput) error
}

type StoreConfig struct {
	Provider string
	Region   string
	FilePath string
}

func GetStore(cfg StoreConfig) (Store, error) {
	switch cfg.Provider {
	case util.SsmProvider:
		return NewSSMStore(aws.NewSession(a.Config{Region: &cfg.Region}))
	case util.SecretsManagerProvider:
		return NewSecretsManagerStore(aws.NewSession(a.Config{Region: &cfg.Region}))
	case util.GpgProvider:
		return NewGpgStore(GpgStoreOptions{Path: cfg.FilePath})
	default:
		return nil, fmt.Errorf("invalid provider `%s`", cfg.Provider)
	}
}

func (c *Config) Key() string {
	parts := strings.Split(*c.Name, "/")
	return parts[len(parts)-1]
}

func (c *ConfigInput) Key() string {
	parts := strings.Split(c.Name, "/")
	return parts[len(parts)-1]
}

func (c *Config) Path() string {
	parts := strings.Split(*c.Name, "/")
	return strings.Join(parts[0:len(parts)-1], "/")
}
