package updates

import (
	"fmt"
	"os"

	"github.com/asaidimu/updater"

	"github.com/asaidimu/hestia/core/runtime"
)

// HandleStartup consumes a --perform-update launch produced by ApplyUpdate
// (waiting for the old process, swapping the executable, clearing the pending
// update), then removes leftover staged binaries. It runs inside Setup, before
// migrations and before the CLI arg parse. Returns normally on a regular launch.
//
// The Store is deliberately nil here — the DB (and the settings collection)
// isn't open yet; a stale pending row is cleared post-boot by Store.Reconcile.
func HandleStartup(cfg *runtime.SelfUpdateConfig, conf *runtime.Config) error {
	if cfg == nil {
		return nil
	}
	dataDir := cfg.DataDir
	if dataDir == "" {
		dataDir = conf.DataDir
	}
	exe := cfg.ExecutablePath
	if exe == "" {
		var err error
		exe, err = os.Executable()
		if err != nil {
			return fmt.Errorf("resolve executable: %w", err)
		}
	}
	u, err := updater.New(cfg.Provider, updater.Config{
		Version:          conf.Version,
		DataDir:          dataDir,
		ExecutablePath:   exe,
		ForwardArguments: cfg.ForwardArguments,
	})
	if err != nil {
		return fmt.Errorf("init updater: %w", err)
	}
	if u.HandleUpdateMode() {
		fmt.Println("updated; resuming normal operation")
		// HandleUpdateMode only restores the original argv when ForwardArguments
		// is set. Otherwise the update-mode flags (--perform-update,
		// --original-path=, --pid=) stay in os.Args and would be parsed by the
		// CLI interface at Start, aborting the freshly-swapped process. Drop
		// them so the new process starts as if launched normally.
		if len(os.Args) > 1 && os.Args[1] == "--perform-update" {
			os.Args = os.Args[:1]
		}
	}
	if err := u.Cleanup(); err != nil {
		return fmt.Errorf("cleanup staged update: %w", err)
	}
	return nil
}