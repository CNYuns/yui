package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server" yaml:"server"`
	Auth     AuthConfig     `mapstructure:"auth" yaml:"auth"`
	Database DatabaseConfig `mapstructure:"database" yaml:"database"`
	Xray     XrayConfig     `mapstructure:"xray" yaml:"xray"`
	TLS      TLSConfig      `mapstructure:"tls" yaml:"tls"`
	Log      LogConfig      `mapstructure:"log" yaml:"log"`
}

type ServerConfig struct {
	Addr string `mapstructure:"addr" yaml:"addr"`
	Mode string `mapstructure:"mode" yaml:"mode"`
}

type AuthConfig struct {
	JWTSecret     string `mapstructure:"jwt_secret" yaml:"jwt_secret"`
	TokenTTLHours int    `mapstructure:"token_ttl_hours" yaml:"token_ttl_hours"`
}

type DatabaseConfig struct {
	Driver string `mapstructure:"driver" yaml:"driver"`
	DSN    string `mapstructure:"dsn" yaml:"dsn"`
}

type XrayConfig struct {
	BinaryPath string `mapstructure:"binary_path" yaml:"binary_path"`
	ConfigPath string `mapstructure:"config_path" yaml:"config_path"`
	AssetsPath string `mapstructure:"assets_path" yaml:"assets_path"`
}

type TLSConfig struct {
	AutoACME bool   `mapstructure:"auto_acme" yaml:"auto_acme"`
	CertPath string `mapstructure:"cert_path" yaml:"cert_path"`
	KeyPath  string `mapstructure:"key_path" yaml:"key_path"`
	Email    string `mapstructure:"email" yaml:"email"`
}

type LogConfig struct {
	Level  string `mapstructure:"level" yaml:"level"`
	Output string `mapstructure:"output" yaml:"output"`
}

var GlobalConfig *Config

// generateSecureSecret 生成安全的随机密钥
func generateSecureSecret(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		panic("无法生成安全随机数")
	}
	return hex.EncodeToString(bytes)
}

// isInsecureSecret 检查密钥是否不安全
func isInsecureSecret(secret string) bool {
	insecureDefaults := []string{
		"",
		"your-super-secret-key-change-me",
		"change-me",
		"secret",
		"jwt-secret",
	}
	for _, d := range insecureDefaults {
		if secret == d {
			return true
		}
	}
	return len(secret) < 32
}

// saveConfig 保存配置到文件
func saveConfig(cfg *Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func Load(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	viper.SetDefault("server.addr", ":8080")
	viper.SetDefault("server.mode", "release")
	viper.SetDefault("auth.token_ttl_hours", 24)
	viper.SetDefault("database.driver", "sqlite")
	viper.SetDefault("database.dsn", "xpanel.db")
	viper.SetDefault("xray.binary_path", "/usr/local/bin/xray")
	viper.SetDefault("xray.config_path", "/etc/xray/config.json")
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.output", "stdout")

	configExists := true
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			configExists = false
		} else {
			return nil, err
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// 检查并自动生成 JWT 密钥
	needSave := false
	if isInsecureSecret(cfg.Auth.JWTSecret) {
		cfg.Auth.JWTSecret = generateSecureSecret(32) // 生成64字符的十六进制密钥
		needSave = true
		fmt.Println("已自动生成安全的 JWT 密钥")
	}

	// 如果配置不存在或需要保存新密钥，则保存配置
	if !configExists || needSave {
		if err := saveConfig(&cfg, configPath); err != nil {
			fmt.Printf("警告: 无法保存配置文件: %v\n", err)
		} else {
			fmt.Printf("配置已保存到: %s\n", configPath)
		}
	}

	GlobalConfig = &cfg
	return &cfg, nil
}
