package config

import (
	"flag"
	"time"
)

const (
	DefaultListenAddr               string        = ":9081"
	DefaultProcessingTimeout        time.Duration = 60 * time.Second
	DefaultReadTimeout              time.Duration = 5 * time.Second
	DefaultReadHeaderTimeout        time.Duration = 5 * time.Second
	DefaultWriteTimeout             time.Duration = 10 * time.Second
	DefaultIdleTimeout              time.Duration = 30 * time.Second
	DefaultGracefullShutdownTimeout time.Duration = 5 * time.Second
	DefaultPsqlDSN                  string        = ""
	DefaultServerEncryptionKey      string        = "secret"
	DefaultAccrualAddress           string        = ""
)

type (
	GophermartConfig struct {
		ListenAddr               string
		ProcessingTimeout        time.Duration
		ReadTimeout              time.Duration
		ReadHeaderTimeout        time.Duration
		WriteTimeout             time.Duration
		IdleTimeout              time.Duration
		GracefullShutdownTimeout time.Duration
		PsqlDSN                  string
		EncryptionKey            string
		AccrualAddress           string
	}

	GophermartConfigInterface interface {
		UpdateFromEnvironment() error
		UpdateFromCLIArgs() error
		Validate() error
	}
)

func GetDefaultGophermartConfig() *GophermartConfig {
	return &GophermartConfig{
		ListenAddr:               DefaultListenAddr,
		ProcessingTimeout:        DefaultProcessingTimeout,
		ReadTimeout:              DefaultReadTimeout,
		ReadHeaderTimeout:        DefaultReadHeaderTimeout,
		WriteTimeout:             DefaultWriteTimeout,
		IdleTimeout:              DefaultIdleTimeout,
		GracefullShutdownTimeout: DefaultGracefullShutdownTimeout,
		PsqlDSN:                  DefaultPsqlDSN,
		EncryptionKey:            DefaultServerEncryptionKey,
		AccrualAddress:           DefaultAccrualAddress,
	}
}

func (c *GophermartConfig) Validate() error {
	if err := ValidateAddr(c.ListenAddr); err != nil {
		return err
	}
	return nil
}

func (c *GophermartConfig) UpdateFromEnvironment() error {
	c.ListenAddr = GetEnvOrFallback("RUN_ADDRESS", c.ListenAddr)
	c.PsqlDSN = GetEnvOrFallback("DATABASE_URI", c.PsqlDSN)
	c.AccrualAddress = GetEnvOrFallback("ACCRUAL_SYSTEM_ADDRESS", c.AccrualAddress)
	return nil
}

func (c *GophermartConfig) UpdateFromCLIArgs(flagSet *flag.FlagSet, args []string) error {
	flagSet.StringVar(&c.PsqlDSN, "d", c.PsqlDSN, "SQL database DSN string")
	flagSet.StringVar(&c.ListenAddr, "a", c.ListenAddr, "server listen address")
	flagSet.StringVar(&c.AccrualAddress, "r", c.AccrualAddress, "accrual address")

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	return nil
}

// Returns the instance of server configuration struct.
//
// Fills it's fileds by using CLI arguments and environments.
// ENV or CLI argument or the default values
func NewGophermartConfig(flagSet *flag.FlagSet, args []string) (*GophermartConfig, error) {
	cfg := GetDefaultGophermartConfig()

	if err := cfg.UpdateFromCLIArgs(flagSet, args); err != nil {
		return nil, err
	}

	if err := cfg.UpdateFromEnvironment(); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}
