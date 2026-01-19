package xconfig

import (
    "fmt"
    "sync"
    "sync/atomic"
)

var (
    globalOnce sync.Once
    globalCfg  atomic.Value // stores *Config
    globalErr  atomic.Value // stores error
)

// InitGlobal initializes the global config singleton.
// Call this once at program startup after parsing CLI flags.
func InitGlobal(c *Config) {
    if c == nil {
        panic("xconfig: InitGlobal with nil config")
    }
    globalOnce.Do(func() {
        globalCfg.Store(c)
    })
}

// InitGlobalFromCLI parses config from CLI command then stores as global.
// Helpful if you want a single call in main().
// Note: requires you already built the CLI command and flags.
func InitGlobalFromCLI(cmd any) (*Config, error) {
    // We avoid importing cli here to keep xconfig decoupled.
    // If you prefer, you can remove this helper and call cfg.Parse(c) in main() yourself.
    return nil, fmt.Errorf("xconfig: InitGlobalFromCLI not implemented; call cfg.Parse(cliCmd) then InitGlobal(cfg)")
}

// Get returns the global config if initialized, otherwise nil.
func Get() *Config {
    v := globalCfg.Load()
    if v == nil {
        return nil
    }
    return v.(*Config)
}

// Must returns the global config or panics if not initialized.
func Must() *Config {
    cfg := Get()
    if cfg == nil {
        panic("xconfig: global config not initialized; call xconfig.InitGlobal(cfg) at startup")
    }
    return cfg
}
