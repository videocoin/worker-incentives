package incentives

import (
	"context"	
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"
	"crypto/ecdsa"
	"golang.org/x/oauth2"
	"github.com/ethereum/go-ethereum/ethclient"
	erpc "github.com/ethereum/go-ethereum/rpc"	
	stackdriver "github.com/TV4/logrus-stackdriver-formatter"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	//"github.com/ethereum/go-ethereum/common"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
	//"golang.org/x/oauth2/jwt"
)

type Auth struct {
	AuthURL  string
	ClientID string
}

type Config struct {
	WorkerChainURL string `default:"https://symphony.dev.videocoin.net/"`
	Auth *Auth	
	KeyFile         string `default:"keyfile.json"`
	KeyPasswordFile string `default:"pwfile.json"`
	CredentialsFile string `default:"credentials.json"`
	ClientID        string `default:"47928468404-chckab1fperdmo0bno5on8kvth6kha4m.apps.googleusercontent.com"`
	LogLevel        string `default:"debug"`
	JobTimeout           time.Duration  `default:"60m"`
	Address              string         `default:"0.0.0.0:8000"`
	WriteTimeout         time.Duration  `default:"10s"`
	ReadTimeout          time.Duration  `default:"10s"`
	MaxHeaderBytes       int            `default:"4096"`
	InputFileName string `default:"test.csv"`
	OutputFileName string `default:"test_receipt.csv"`
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
	senderPrivKey *ecdsa.PrivateKey
	tokenSrc      oauth2.TokenSource
	client        *ethclient.Client
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

func Dial(ctx context.Context, url string, tokenSrc oauth2.TokenSource) (*ethclient.Client, error) {

	r, err := erpc.DialHTTPWithClient(url, oauth2.NewClient(ctx, tokenSrc))
	if err != nil {
		return nil, err
	}

	client := ethclient.NewClient(r)
	return client, nil
}

func transactor(keyfile, password string) (*bind.TransactOpts, *ecdsa.PrivateKey, error) {
	f, err := os.Open(keyfile)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, nil, err
	}
	key, err := keystore.DecryptKey(data, password)
	if err != nil {
		return nil, nil, err
	}
	return bind.NewKeyedTransactor(key.PrivateKey), key.PrivateKey, nil
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

	// Retrieving Access Token using Service Account by Google's OAuth2 package for Golang
	credsBytes, err := ioutil.ReadFile(conf.CredentialsFile)
	if err != nil {
		return nil, err
	}
	app.tokenSrc, err = google.JWTAccessTokenSourceFromJSON(credsBytes, conf.ClientID)
	if err != nil {
		return nil, err
	}
	c := context.Background()
	wc, err := Dial(c, conf.WorkerChainURL, app.tokenSrc)
	if err != nil {
		return nil, err
	}

	app.client = wc

	pw, err := readPassword(conf.KeyPasswordFile)
	if err != nil {
		return nil, err
	}
	app.opts, app.senderPrivKey, err = transactor(conf.KeyFile, pw)
	if err != nil {
		return nil, err
	}
	return app, nil
}
