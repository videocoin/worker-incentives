package incentives

import (
	"context"	
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"
	"crypto/ecdsa"
	"golang.org/x/oauth2"
	"github.com/ethereum/go-ethereum/ethclient"
	erpc "github.com/ethereum/go-ethereum/rpc"	
	stackdriver "github.com/TV4/logrus-stackdriver-formatter"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/idtoken"
)

type Config struct {
	WorkerChainURL string `default:"https://symphony.dev.videocoin.net/"`
	KeyFile         string `default:"keyfile.json"`
	KeyPasswordFile string `default:"pwfile.json"`
	CredentialsFile string `default:"credentials.json"`
	ClientID        string `default:"47928468404-hfuqhrb6lhtv9sem30rkjc1djcrlpt4v.apps.googleusercontent.com"`
	LogLevel        string `default:"debug"`
	JobTimeout           time.Duration  `default:"60m"`
	InputFileName string `default:"test.csv"`
	OutputFileName string `default:"test_receipt.csv"`
}

func FromEnv() (conf Config) {
	if err := envconfig.Process("INCENTIVES", &conf); err != nil {
		panic("failed to process envconfig " + err.Error())
	}
	fmt.Println("KeyFile:", conf.KeyFile)
	return
}

type App struct {
	conf          Config
	log           *logrus.Entry
	opts          *bind.TransactOpts
	senderPrivKey *ecdsa.PrivateKey
	tokenSrc      oauth2.TokenSource
	client        *ethclient.Client
	id int
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

func (app *App)Dial(ctx context.Context, url string, clientOption idtoken.ClientOption) (*ethclient.Client, error) {

	ts, err := idtoken.NewClient(ctx, app.Config().ClientID, clientOption)
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
	/*
	credsBytes, err := ioutil.ReadFile(conf.CredentialsFile)
	if err != nil {
		return nil, err
	}
	app.tokenSrc, err = google.JWTAccessTokenSourceFromJSON(credsBytes, conf.ClientID)
	if err != nil {
		return nil, err
	}
	*/
	clientOption := idtoken.WithCredentialsFile(conf.CredentialsFile)
	app.Log().WithFields(logrus.Fields{"client_id": conf.ClientID}).Debug("JWTAccessTokenSourceFromJSON processed successfully")
	c := context.Background()
	wc, err := app.Dial(c, conf.WorkerChainURL, clientOption)
	if err != nil {
		return nil, err
	}
	app.Log().WithFields(logrus.Fields{"WorkerChainURL": conf.WorkerChainURL}).Debug("Dial processed successfully")
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
