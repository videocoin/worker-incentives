package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	stackdriver "github.com/TV4/logrus-stackdriver-formatter"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/ethclient"
	erpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/api/idtoken"
)

type Config struct {
	WorkerChainURL  string
	KeyFile         string
	Password        string
	CredentialsFile string
	ClientID        string
	LogLevel        string
	JobTimeout      time.Duration `default:"60m"`
}

func transactor(keyfile, password string) (*bind.TransactOpts, error) {
	f, err := os.Open(keyfile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	key, err := keystore.DecryptKey(data, password)
	if err != nil {
		return nil, err
	}
	return bind.NewKeyedTransactor(key.PrivateKey), nil
}

func readPassword(passwordFile string) (string, error) {
	if len(passwordFile) == 0 {
		return "", nil
	}
	password, err := ioutil.ReadFile(passwordFile)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(password), "\n"), nil
}

type Context struct {
	context.Context

	Config Config
	Path   string

	Logger *logrus.Entry

	Transactor *bind.TransactOpts

	client *ethclient.Client
	opts   *bind.TransactOpts
}

func (c Context) Dial(url string, clientOption idtoken.ClientOption) (*ethclient.Client, error) {

	ts, err := idtoken.NewClient(c, c.Config.ClientID, clientOption)
	if err != nil {
		return nil, err
	}

	r, err := erpc.DialHTTPWithClient(url, ts)
	if err != nil {
		return nil, err
	}

	client := ethclient.NewClient(r)
	return client, nil
}

func (c *Context) Incentives() (eng Engine, err error) {

	logger := logrus.New()
	logLevel, err := logrus.ParseLevel(c.Config.LogLevel)
	if err != nil {
		return eng, fmt.Errorf("invalid log level: %s", c.Config.LogLevel)
	}
	logger.SetLevel(logLevel)
	logger.SetFormatter(stackdriver.NewFormatter(
		stackdriver.WithService("incentives"),
	))
	c.Logger = logrus.NewEntry(logger)

	clientOption := idtoken.WithCredentialsFile(c.Config.CredentialsFile)
	c.Logger.WithFields(logrus.Fields{"client_id": c.Config.ClientID}).Debug("JWTAccessTokenSourceFromJSON processed successfully")

	wc, err := c.Dial(c.Config.WorkerChainURL, clientOption)
	if err != nil {
		return eng, err
	}
	c.Logger.WithFields(logrus.Fields{"WorkerChainURL": c.Config.WorkerChainURL}).Debug("Dial processed successfully")
	c.client = wc

	c.opts, err = transactor(c.Config.KeyFile, c.Config.Password)
	if err != nil {
		return eng, err
	}
	return NewEngine(c.Logger), nil
}

func GetContext(path string) (Context, error) {
	conf := Config{}
	if len(path) > 0 {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return Context{}, fmt.Errorf("can't read config at %s: %w", path, err)
		}
		if err := json.Unmarshal(data, &conf); err != nil {
			return Context{}, err
		}
	}
	if err := envconfig.Process("INCENTIVES", &conf); err != nil {
		return Context{}, fmt.Errorf("failed to parse envconfig: %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, syscall.SIGINT)
	go func() {
		<-sigint
		cancel()
	}()

	return ToContext(ctx, conf, path)
}

func ToContext(ctx context.Context, conf Config, path string) (Context, error) {
	path, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return Context{}, err
	}
	appctx := Context{
		Context: ctx,
		Config:  conf,
		Path:    path,
	}
	logger := logrus.New()
	logLevel, err := logrus.ParseLevel(conf.LogLevel)
	if err != nil {
		return appctx, err
	}
	logger.SetLevel(logLevel)
	appctx.Logger = logrus.NewEntry(logger)

	var txn *bind.TransactOpts
	if len(conf.KeyFile) > 0 {
		keypath := conf.KeyFile
		if !filepath.IsAbs(keypath) {
			keypath = filepath.Join(path, keypath)
		}
		password := conf.Password
		txn, err = transactor(keypath, password)
		if err != nil {
			return appctx, err
		}
	} else {
		appctx.Logger.Warn("key file is not provided.")
	}

	appctx.Transactor = txn
	return appctx, nil
}

func PayCommand(config *string) *cobra.Command {
	execOpts := ExecuteOpts{}
	var (
		output string
		input  string
	)
	cmd := &cobra.Command{
		Use:     "pay",
		Aliases: []string{"p"},
		Short:   "Incentives to workers.",
		RunE: func(cmd *cobra.Command, args []string) error {

			if len(input) > 0 {
				execOpts.inputfile = input
			} else {
				return fmt.Errorf("--input=%v is not suppied", input)
			}
			if len(output) > 0 {
				execOpts.outputfile = output
			} else {
				return fmt.Errorf("--output=%v is not suppied", input)
			}

			ctx, err := GetContext(*config)
			if err != nil {
				return err
			}

			engine, err := ctx.Incentives()
			if err != nil {
				return err
			}

			execOpts.client = ctx.client
			execOpts.opts = ctx.opts
			err = engine.Execute(ctx, &execOpts)
			if err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&input, "input", "", "supply csv payment file")
	cmd.Flags().StringVar(&output, "output", "", "save transaction csv report to the file")

	return cmd
}

func RootCommand() *cobra.Command {
	config := ""
	cmd := &cobra.Command{
		Use:   "worker-incentives [sub]",
		Short: "worker-incentives is a command line utility to distribute worker incentives.",
	}
	cmd.AddCommand(PayCommand(&config))
	cmd.AddCommand(VersionCommand())
	cmd.PersistentFlags().StringVarP(&config, "config", "c", "", "Configuraton for incentives command.")
	return cmd
}
