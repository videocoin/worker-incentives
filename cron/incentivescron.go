package main

import (
	"encoding/base64"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-cmd/cmd"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

func main() {
	logger := log.New(os.Stdout, "incentivesreport: ", log.LstdFlags)
	cmdOptions := cmd.Options{
		Buffered:  false,
		Streaming: true,
	}
	if os.Getenv("Mode") == "Payout" {
		logger.Println("Incentives Jobs will run in Payout Mode")
		t := time.Now()
		month := t.Month()
		now := time.Date(t.Year(), month, 25, 0, 0, 0, 0, t.Location())
		lastmonth := now.AddDate(0, -1, 0)
		strnow := now.String()
		strlastmonth := lastmonth.String()

		workerJob := cmd.NewCmdOptions(cmdOptions, "/opt/incentives/job", "incentives", "WorkerIncentives.csv", "UptimeReport.txt", strlastmonth, strnow)
		jobChan := make(chan struct{})
		go func() {
			defer close(jobChan)
			for workerJob.Stdout != nil || workerJob.Stderr != nil {
				select {
				case line, open := <-workerJob.Stdout:
					if !open {
						workerJob.Stdout = nil
						continue
					}
					f, err := os.OpenFile("/opt/incentives/JobLogFile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
					if err != nil {
						log.Fatalf("error opening file: %v", err)
					}
					defer f.Close()

					log.SetOutput(f)
					log.Println(line)
					f.Close()

				case line, open := <-workerJob.Stderr:
					if !open {
						workerJob.Stderr = nil
						continue
					}
					log.Printf(line)
				}
			}
		}()
		<-workerJob.Start()
		<-jobChan

		incentivesJob := cmd.NewCmdOptions(cmdOptions, "/opt/incentives/incentives", "pay", "--input", "/WorkerIncentives.csv", "--output", "WorkerIncentivesReceipt.csv", "-c", "/vault/secrets/prodrun.json")
		incentivesChan := make(chan struct{})
		go func() {
			defer close(incentivesChan)
			for incentivesJob.Stdout != nil || incentivesJob.Stderr != nil {
				select {
				case line, open := <-incentivesJob.Stdout:
					if !open {
						incentivesJob.Stdout = nil
						continue
					}
					f, err := os.OpenFile("/opt/incentives/IncentivesLogFile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
					if err != nil {
						log.Fatalf("error opening file: %v", err)
					}
					defer f.Close()

					log.SetOutput(f)
					log.Println(line)
					f.Close()

				case line, open := <-incentivesJob.Stderr:
					if !open {
						incentivesJob.Stderr = nil
						continue
					}
					log.Printf(line)
				}
			}
		}()
		<-incentivesJob.Start()
		<-incentivesChan

		logger.Println("Incentives Jobs completed in Payout Mode and Files Created")
		logger.Println("Sending by Email the Files")
		sendEmail()

	} else {
		logger.Println("Worker Incentives Job will run in Report Mode")
		t := time.Now()
		month := t.Month()
		now := time.Date(t.Year(), month, 25, 0, 0, 0, 0, t.Location())
		lastmonth := now.AddDate(0, -1, 0)
		strnow := now.String()
		strlastmonth := lastmonth.String()

		workerJob := cmd.NewCmdOptions(cmdOptions, "/opt/incentives/job", "incentives", "WorkerIncentives.csv", "UptimeReport.txt", strlastmonth, strnow)
		jobChan := make(chan struct{})
		go func() {
			defer close(jobChan)
			for workerJob.Stdout != nil || workerJob.Stderr != nil {
				select {
				case line, open := <-workerJob.Stdout:
					if !open {
						workerJob.Stdout = nil
						continue
					}
					f, err := os.OpenFile("/opt/incentives/JobLogFile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
					if err != nil {
						log.Fatalf("error opening file: %v", err)
					}
					defer f.Close()

					log.SetOutput(f)
					log.Println(line)
					f.Close()

				case line, open := <-workerJob.Stderr:
					if !open {
						workerJob.Stderr = nil
						continue
					}
					log.Printf(line)
				}
			}
		}()
		<-workerJob.Start()
		<-jobChan

		logger.Println("Incentives Jobs completed in Report Mode and Files Created")
		logger.Println("Sending by Email the Files")
		sendEmail()
	}
	logger.Println("Incentives Report Complete")
}

func sendEmail() {
	logger := log.New(os.Stdout, "incentivesreport: ", log.LstdFlags)
	sendgripkeyfile, _ := ioutil.ReadFile("/vault/secrets/sendgrip.key")
	sendgripkey := strings.TrimRight(string(sendgripkeyfile), "\n")

	request := sendgrid.GetRequest(string(sendgripkey), "/v3/mail/send", "https://api.sendgrid.com")
	request.Method = "POST"
	if os.Getenv("Mode") == "Payout" {
		var Body = IncentivesEmailConfPayout()
		request.Body = Body
		response, err := sendgrid.API(request)
		if err != nil {
			logger.Println(err)
		} else {
			logger.Println(response.StatusCode)
			logger.Println(response.Headers)
		}
	} else {
		var Body = IncentivesEmailConf()
		request.Body = Body
		response, err := sendgrid.API(request)
		if err != nil {
			logger.Println(err)
		} else {
			logger.Println(response.StatusCode)
			logger.Println(response.Headers)
		}
	}
}

func IncentivesEmailConfPayout() []byte {
	address := "workers-incentives@liveplanet.net"
	name := "Incentives Reports"
	from := mail.NewEmail(name, address)
	subject := "Incentives - Monthly Payout"

	address = "workers-incentives@liveplanet.net"
	name = "Incentives Reports Google Groups"
	to := mail.NewEmail(name, address)

	content := mail.NewContent("text/plain", "Incentives - Monthly Payout")
	m := mail.NewV3MailInit(from, subject, to, content)

	address = "andres@liveplanet.net"
	name = "Andres Monje"
	email := mail.NewEmail(name, address)
	m.Personalizations[0].AddTos(email)

	logfile, _ := ioutil.ReadFile("/opt/incentives/WorkerIncentivesReceipt.csv")
	logfileEncoded := base64.StdEncoding.EncodeToString(logfile)
	csvfile, _ := ioutil.ReadFile("/WorkerIncentives.csv")
	csvfileEncoded := base64.StdEncoding.EncodeToString(csvfile)

	a := mail.NewAttachment()
	a.SetContent(logfileEncoded)
	a.SetFilename("WorkerIncentivesReceipt.csv")
	a.SetDisposition("attachment")
	a.SetContentID("WorkerIncentivesReceipt")
	m.AddAttachment(a)
	a2 := mail.NewAttachment()
	a2.SetContent(csvfileEncoded)
	a2.SetFilename("WorkerIncentives.csv")
	a2.SetDisposition("attachment")
	a2.SetContentID("WorkerIncentives")
	m.AddAttachment(a2)
	return mail.GetRequestBody(m)
}

func IncentivesEmailConf() []byte {
	address := "workers-incentives@liveplanet.net"
	name := "Incentives Reports"
	from := mail.NewEmail(name, address)
	subject := "Incentives Monthly Report"

	address = "workers-incentives@liveplanet.net"
	name = "Incentives Reports Google Groups"
	to := mail.NewEmail(name, address)

	content := mail.NewContent("text/plain", "Incentives Monthly Report")
	m := mail.NewV3MailInit(from, subject, to, content)

	address = "andres@liveplanet.net"
	name = "Andres Monje"
	email := mail.NewEmail(name, address)
	m.Personalizations[0].AddTos(email)

	logfile, _ := ioutil.ReadFile("/UptimeReport.txt")
	logfileEncoded := base64.StdEncoding.EncodeToString(logfile)
	csvfile, _ := ioutil.ReadFile("/WorkerIncentives.csv")
	csvfileEncoded := base64.StdEncoding.EncodeToString(csvfile)

	a := mail.NewAttachment()
	a.SetContent(logfileEncoded)
	a.SetFilename("UptimeReport.txt")
	a.SetDisposition("attachment")
	a.SetContentID("UptimeReport")
	m.AddAttachment(a)
	a2 := mail.NewAttachment()
	a2.SetContent(csvfileEncoded)
	a2.SetFilename("WorkerIncentives.csv")
	a2.SetDisposition("attachment")
	a2.SetContentID("WorkerIncentives")
	m.AddAttachment(a2)
	return mail.GetRequestBody(m)
}
