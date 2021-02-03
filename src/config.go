package incentives

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	stackdriver "github.com/TV4/logrus-stackdriver-formatter"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/sheets/v4"
)

type Config struct {
	EthereumURL     string `default:"https://goerli.infura.io/v3/977d81ca036540f48c4ab19f1927dcd0"`
	KeyFile         string `default:"keyfile.json"`
	KeyPasswordFile string `default:"pwfile.json"`
	InputTypeCsv    bool	`default:"true"`
	// Required only if spread sheet is eabled
	CredentialsFile string `default:"credentials.json"`
	LogLevel        string `default:"debug"`
	// All columns are expected to be one after the other.
	// TxColumn will be updated by the app.
	WorkerColumn     string        `default:"A"`
	AmountColumn     string        `default:"B"`
	TxColumn         string        `default:"C"`
	StartRow         int           `default:"2"`
	BatchSize        int           `default:"50"`
	HistoryRetention time.Duration `default:"24h"`
	// JobTimeout is a single timeout for paying to all pending users.
	// Job will run asynchronously.
	JobTimeout           time.Duration  `default:"60m"`
	Address              string         `default:"0.0.0.0:8000"`
	WriteTimeout         time.Duration  `default:"10s"`
	ReadTimeout          time.Duration  `default:"10s"`
	MaxHeaderBytes       int            `default:"4096"`
	ERC20Address         common.Address `default:"0x462630b53AaDfB80AC9c0ee7F51a7e6eAd5c0e92"`
	BatchContractAddress common.Address `default:"0x10ae5201F0fd548E3E25763288778958420188C0"`
	DryRun               bool           `default:"false"`
	// for videocoin erc20 (e.g. prod environment use increaseApproval)
	// for test env use increaseAllowance
	IncreaseAllowanceMethod string `default:"increaseAllowance"`

	// SpreadsheetID is used only for executions using job subcommand
	//SpreadsheetID string `default:"1cEA4VMcm6muATcezXVbnPIen_rndBlMJqbklP6HSiTc"`
	SpreadsheetID string `default:"test.csv"`
}

func FromEnv() (conf Config) {
	if err := envconfig.Process("INCENTIVES", &conf); err != nil {
		panic("failed to process envconfig " + err.Error())
	}
	return
}

const (
	stateIdle = iota + 1
	statePending
)

type App struct {
	conf          Config
	log           *logrus.Entry
	opts          *bind.TransactOpts
	sheetsService *sheets.Service
	oauthConf     *jwt.Config

	// application allows only one task to run concurrently
	// until the task is interrupted by timeout - there is no way
	// to start the next task
	mu sync.Mutex
	id int
	// state of the app
	// 0 - no pending tasks
	// 1 - pending task
	// 2 - task completed
	state uint64
	// err will be set only if request was completed with error!
	results map[int]*result
}

func (app *App) Config() Config {
	return app.conf
}

func (app *App) Log() *logrus.Entry {
	return app.log
}

type result struct {
	timestamp time.Time
	err       error
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

func NewApp(conf Config) (*App, error) {
	app := &App{conf: conf, results: map[int]*result{}}

	logger := logrus.New()
	logLevel, err := logrus.ParseLevel(conf.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %s", conf.LogLevel)
	}
	logger.SetLevel(logLevel)
	logger.SetFormatter(stackdriver.NewFormatter(
		stackdriver.WithService("incentives"),
	))
	app.log = logrus.NewEntry(logger)

	if !conf.InputTypeCsv {
		credsBytes, err := ioutil.ReadFile(conf.CredentialsFile)
		if err != nil {
			return nil, err
		}
		app.oauthConf, err = google.JWTConfigFromJSON(credsBytes, "https://www.googleapis.com/auth/spreadsheets")
		if err != nil {
			return nil, err
		}
	}
	pw, err := readPassword(conf.KeyPasswordFile)
	if err != nil {
		return nil, err
	}
	app.opts, err = transactor(conf.KeyFile, pw)
	if err != nil {
		return nil, err
	}
	return app, nil
}
