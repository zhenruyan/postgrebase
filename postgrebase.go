package postgrebase

import (
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"github.com/zhenruyan/postgrebase/cmd"
	"github.com/zhenruyan/postgrebase/core"
	"github.com/zhenruyan/postgrebase/tools/list"
	"github.com/spf13/cobra"
)

var _ core.App = (*PostgreBase)(nil)

// Version of PostgreBase
var Version = "(untracked)"

// appWrapper serves as a private core.App instance wrapper.
type appWrapper struct {
	core.App
}

// PostgreBase defines a PostgreBase app launcher.
//
// It implements [core.App] via embedding and all of the app interface methods
// could be accessed directly through the instance (eg. PostgreBase.DataDir()).
type PostgreBase struct {
	*appWrapper

	debugFlag         bool
	dataDirFlag       string
	dataLogFlag       string
	dataDataFlag      string
	redisFlag         string
	encryptionEnvFlag string
	hideStartBanner   bool

	// RootCmd is the main console command
	RootCmd *cobra.Command
}

// Config is the PostgreBase initialization config struct.
type Config struct {
	// optional default values for the console flags
	DefaultDebug         bool
	DefaultDataDir       string // if not set, it will fallback to "./pb_data"
	DefaultDataDsn       string // if not set, it will fallback to "postgresql://<username>:<password>@<host>:<port>/<database>?sslmode=verify-full"
	RedisDsn             string //redis://<user>:<pass>@localhost:6379/<db>
	DefaultEncryptionEnv string

	// hide the default console server info on app startup
	HideStartBanner bool

	// optional DB configurations
	DataMaxOpenConns int // default to core.DefaultDataMaxOpenConns
	DataMaxIdleConns int // default to core.DefaultDataMaxIdleConns
	LogsMaxOpenConns int // default to core.DefaultLogsMaxOpenConns
	LogsMaxIdleConns int // default to core.DefaultLogsMaxIdleConns
}

// New creates a new PostgreBase instance with the default configuration.
// Use [NewWithConfig()] if you want to provide a custom configuration.
//
// Note that the application will not be initialized/bootstrapped yet,
// aka. DB connections, migrations, app settings, etc. will not be accessible.
// Everything will be initialized when [Start()] is executed.
// If you want to initialize the application before calling [Start()],
// then you'll have to manually call [Bootstrap()].
func New() *PostgreBase {
	_, isUsingGoRun := inspectRuntime()

	return NewWithConfig(Config{
		DefaultDebug: isUsingGoRun,
	})
}

// NewWithConfig creates a new PostgreBase instance with the provided config.
//
// Note that the application will not be initialized/bootstrapped yet,
// aka. DB connections, migrations, app settings, etc. will not be accessible.
// Everything will be initialized when [Start()] is executed.
// If you want to initialize the application before calling [Start()],
// then you'll have to manually call [Bootstrap()].
func NewWithConfig(config Config) *PostgreBase {
	// initialize a default data directory based on the executable baseDir
	if config.DefaultDataDir == "" {
		baseDir, _ := inspectRuntime()
		config.DefaultDataDir = filepath.Join(baseDir, "pb_data")
	}
	if config.DefaultDataDsn == "" {
		config.DefaultDataDsn = "postgresql://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable"
	}
	if config.RedisDsn == "" {
		config.RedisDsn = ""
	}
	if config.DefaultDataDir == "" {
		baseDir, _ := inspectRuntime()
		config.DefaultDataDir = filepath.Join(baseDir, "pb_data")
	}

	pb := &PostgreBase{
		RootCmd: &cobra.Command{
			Use:     filepath.Base(os.Args[0]),
			Short:   "PostgreBase — AI-Native No-Code API Platform",
			Version: Version,
			FParseErrWhitelist: cobra.FParseErrWhitelist{
				UnknownFlags: true,
			},
			// no need to provide the default cobra completion command
			CompletionOptions: cobra.CompletionOptions{
				DisableDefaultCmd: true,
			},
		},
		debugFlag:         config.DefaultDebug,
		dataDirFlag:       config.DefaultDataDir,
		redisFlag:         config.RedisDsn,
		dataDataFlag:      config.DefaultDataDsn,
		encryptionEnvFlag: config.DefaultEncryptionEnv,
		hideStartBanner:   config.HideStartBanner,
	}

	// parse base flags
	// (errors are ignored, since the full flags parsing happens on Execute())
	pb.eagerParseFlags(&config)

	// initialize the app instance
	pb.appWrapper = &appWrapper{core.NewBaseApp(core.BaseAppConfig{
		DataDir:          pb.dataDirFlag,
		DataDsn:          pb.dataDataFlag,
		RedisDsn:         pb.redisFlag,
		EncryptionEnv:    pb.encryptionEnvFlag,
		IsDebug:          pb.debugFlag,
		DataMaxOpenConns: config.DataMaxOpenConns,
		DataMaxIdleConns: config.DataMaxIdleConns,
		LogsMaxOpenConns: config.LogsMaxOpenConns,
		LogsMaxIdleConns: config.LogsMaxIdleConns,
	})}

	// hide the default help command (allow only `--help` flag)
	pb.RootCmd.SetHelpCommand(&cobra.Command{Hidden: true})

	return pb
}

// Start starts the application, aka. registers the default system
// commands (serve, migrate, version) and executes pb.RootCmd.
func (pb *PostgreBase) Start() error {
	// register system commands
	pb.RootCmd.AddCommand(cmd.NewAdminCommand(pb))
	pb.RootCmd.AddCommand(cmd.NewServeCommand(pb, !pb.hideStartBanner))
	pb.RootCmd.AddCommand(cmd.NewMCPCommand(pb, Version))

	return pb.Execute()
}

// Execute initializes the application (if not already) and executes
// the pb.RootCmd with graceful shutdown support.
//
// This method differs from pb.Start() by not registering the default
// system commands!
func (pb *PostgreBase) Execute() error {
	if !pb.skipBootstrap() {
		if err := pb.Bootstrap(); err != nil {
			return err
		}
	}

	done := make(chan bool, 1)

	// listen for interrupt signal to gracefully shutdown the application
	go func() {
		sigch := make(chan os.Signal, 1)
		signal.Notify(sigch, os.Interrupt, syscall.SIGTERM)
		<-sigch
		done <- true
	}()

	// execute the root command
	go func() {
		if err := pb.RootCmd.Execute(); err != nil {
			// @todo replace with db log once generalized logs are added
			// and maybe consider reorganizing the code to return os.Exit(1)
			// (note may need to update the existing commands to not silence errors)
			color.Red(err.Error())
		}

		done <- true
	}()

	<-done

	// trigger app cleanups
	return pb.OnTerminate().Trigger(&core.TerminateEvent{
		App: pb,
	})
}

// eagerParseFlags parses the global app flags before calling pb.RootCmd.Execute().
// so we can have all PostgreBase flags ready for use on initialization.
func (pb *PostgreBase) eagerParseFlags(config *Config) error {
	pb.RootCmd.PersistentFlags().StringVar(
		&pb.dataDirFlag,
		"dir",
		config.DefaultDataDir,
		"the PostgreBase data directory",
	)

	pb.RootCmd.PersistentFlags().StringVar(
		&pb.dataDataFlag,
		"dataDsn",
		config.DefaultDataDsn,
		"store data dsn (postgres://... OR mysql://... OR sqlite:///path/to/data.db)",
	)

	pb.RootCmd.PersistentFlags().StringVar(
		&pb.redisFlag,
		"redisDsn",
		config.RedisDsn,
		"Cache data Redis dsn(default  redis://<user>:<pass>@localhost:6379/<db>  redis://localhost:6379/0)",
	)

	pb.RootCmd.PersistentFlags().StringVar(
		&pb.encryptionEnvFlag,
		"encryptionEnv",
		config.DefaultEncryptionEnv,
		"the env variable whose value of 32 characters will be used \nas encryption key for the app settings (default none)",
	)

	pb.RootCmd.PersistentFlags().BoolVar(
		&pb.debugFlag,
		"debug",
		config.DefaultDebug,
		"enable debug mode, aka. showing more detailed logs",
	)

	return pb.RootCmd.ParseFlags(os.Args[1:])
}

// skipBootstrap eagerly checks if the app should skip the bootstrap process:
// - already bootstrapped
// - is unknown command
// - is the default help command
// - is the default version command
//
// https://github.com/pocketbase/pocketbase/issues/404
// https://github.com/pocketbase/pocketbase/discussions/1267
func (pb *PostgreBase) skipBootstrap() bool {
	flags := []string{
		"-h",
		"--help",
		"-v",
		"--version",
	}

	if pb.IsBootstrapped() {
		return true // already bootstrapped
	}

	cmd, _, err := pb.RootCmd.Find(os.Args[1:])
	if err != nil {
		return true // unknown command
	}

	for _, arg := range os.Args {
		if !list.ExistInSlice(arg, flags) {
			continue
		}

		// ensure that there is no user defined flag with the same name/shorthand
		trimmed := strings.TrimLeft(arg, "-")
		if len(trimmed) > 1 && cmd.Flags().Lookup(trimmed) == nil {
			return true
		}
		if len(trimmed) == 1 && cmd.Flags().ShorthandLookup(trimmed) == nil {
			return true
		}
	}

	return false
}

// inspectRuntime tries to find the base executable directory and how it was run.
func inspectRuntime() (baseDir string, withGoRun bool) {
	if strings.HasPrefix(os.Args[0], os.TempDir()) {
		// probably ran with go run
		withGoRun = true
		baseDir, _ = os.Getwd()
	} else {
		// probably ran with go build
		withGoRun = false
		baseDir = filepath.Dir(os.Args[0])
	}
	return
}
