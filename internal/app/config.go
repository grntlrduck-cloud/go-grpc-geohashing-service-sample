package app

import (
	"fmt"
	"os"
	"strings"

	"dario.cat/mergo"
	"gopkg.in/yaml.v3"
)

const (
	bootConfigPath = "config"
	bootFilePrefix = "boot"
)

type BootConfig struct {
	App     AppConfig     `yaml:"app"`
	Grpc    GrpcConfig    `yaml:"grpc"`
	Aws     AwsConfig     `yaml:"aws"`
	Logging LoggingConfig `yaml:"logging"`
}

type AppConfig struct {
	Name string `yaml:"name"`
}

type GrpcConfig struct {
	Server PortConfig `yaml:"server"`
	Proxy  PortConfig `yaml:"proxy"`
	Ssl    SslConfig  `yaml:"ssl"`
}

type PortConfig struct {
	Port int32 `yaml:"port"`
}

type SslConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertPath string `yaml:"cert_path"`
	KeyPath  string `yaml:"key_path"`
	CaPath   string `yaml:"ca_path"`
}

type AwsConfig struct {
	Config   BasicConfig    `yaml:"config"`
	DynamoDb DynamoDbConfig `yaml:"dynamodb"`
}

type BasicConfig struct {
	Region  string `yaml:"region"`
	Account string `yaml:"account"`
}

type DynamoDbConfig struct {
	PoiTableName     string           `yaml:"poi_table_name"`
	EndpointOverride EndpointOverride `yaml:"endpoint_override"`
	CreateInitTable  bool             `yaml:"create_init_table"`
}

type EndpointOverride struct {
	Enabled bool   `yaml:"enabled"`
	Host    string `yaml:"host"`
	Port    string `yaml:"port"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
}

func LoadBootConfig() (*BootConfig, error) {
	defaultBootFile := fmt.Sprintf("%s/%s.yaml", bootConfigPath, bootFilePrefix)
	defaultBootConfig, err := loadExpandProfile(defaultBootFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load default profile: %w", err)
	}

	bootProfileActive := strings.ToLower(os.Getenv("BOOT_PROFILE_ACTIVE"))
	if bootProfileActive != "" {
		profileBootFile := fmt.Sprintf(
			"%s/%s-%s.yaml",
			bootConfigPath,
			bootFilePrefix,
			bootProfileActive,
		)
		profileBootConfig, err := loadExpandProfile(profileBootFile)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to load profile %s: %w",
				bootProfileActive,
				err,
			)
		}
		if err := mergo.Merge(profileBootConfig, defaultBootConfig); err != nil {
			return nil, fmt.Errorf(
				"failed to merge config %s with default: %w",
				bootProfileActive,
				err,
			)
		}
		return profileBootConfig, nil
	}
	return defaultBootConfig, nil
}

func loadExpandProfile(filePath string) (*BootConfig, error) {
	bootFile, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load boot file %s: %w", filePath, err)
	}
	replaced := os.ExpandEnv(string(bootFile))
	var boot BootConfig
	err = yaml.Unmarshal([]byte(replaced), &boot)
	if err != nil {
		return nil, fmt.Errorf("failed to Unmarshal %s: %w", filePath, err)
	}
	return &boot, nil
}
