package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"reflect"
	"sort"
	"syscall"
	"time"

	client "report-stat/httpclient"

	"github.com/yin1999/healthreport/utils/email"
)

type timeSchedue struct {
	Hour     uint8 `json:"hour"`
	Minute   uint8 `json:"minute"`
	SendMail bool  `json:"sendMail"`
}

type timeArray []timeSchedue

var timeTable timeArray

var timeZone = time.FixedZone("CST", 8*3600)

var (
	logger   = log.Default()
	emailCfg *email.Config

	maxAttempts   uint
	accountPath   string
	emailCfgPath  string
	timeTablePath string
)

func main() {
	logger.Print("Starting app...\n")
	defer logger.Print("Exit.\n")
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	exit := false
	for !exit {
		ctx, cc := context.WithCancel(context.Background())
		go func(cc context.CancelFunc) {
			switch <-c {
			case syscall.SIGINT, syscall.SIGTERM:
				exit = true
				cc()
			case syscall.SIGHUP:
				logger.Print("Reloading app...\n")
				cc()
			}
		}(cc)
		err := app(ctx)
		cc()
		if err != nil && err != context.Canceled {
			log.Fatalln(err)
		}
	}
}

func init() {
	sort.Slice(timeTable, func(i, j int) bool {
		return less(timeTable[i], timeTable[j])
	})

	flagSet := flag.NewFlagSet("parser", flag.ExitOnError)
	flagSet.UintVar(&maxAttempts, "c", 4, "set max attepmts")
	flagSet.StringVar(&accountPath, "a", "config/account.json", "set account file path")
	flagSet.StringVar(&emailCfgPath, "e", "config/email.json", "set email file path")
	flagSet.StringVar(&timeTablePath, "t", "config/timeTable.json", "set time table file path")
	flagSet.Parse(os.Args[1:])
}

func app(ctx context.Context) error {
	var err error
	emailCfg, err = email.LoadConfig(emailCfgPath)
	if err != nil {
		logger.Printf("Warning: email is not enabled, err:%s\n", err.Error())
	}
	account := &client.Account{}
	err = loadJson(account, accountPath)
	if err != nil {
		logger.Fatalln(err)
	}
	sort.Strings(account.Class)
	err = loadJson(&timeTable, timeTablePath)
	if err != nil {
		logger.Fatalln(err)
	}
	sort.Sort(timeTable)

	duration, send := nextTime()
	timer := time.NewTimer(duration)
	for {
		select {
		case <-timer.C:
			if err = task(ctx, account, send); err != nil {
				return err
			}
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		}
		duration, send = nextTime()
		timer.Reset(duration)
	}
}

var ErrMaximumAttemptsExceeded = errors.New("serve: maximum attempts exceeded")

func task(ctx context.Context, account *client.Account, send bool) (err error) {
	logger.Print("Start get form routine\n")

	var timer *time.Timer
	for count := uint(1); true; count++ {
		logger.Print("Start getting form\n")
		var empty bool
		c, cc := context.WithTimeout(ctx, 50*time.Second)
		empty, err = client.GetFormData(c, account)
		cc()
		switch err {
		case nil:
			logger.Print("get form finished\n")
			if !empty && send && emailCfg != nil {
				err = emailCfg.Send("form bot", "未填报名单提醒", fmt.Sprintf("未填报名单查看: <a href=\"https://%s/report-stat/\">链接</a>", account.Domain))
				if err != nil {
					logger.Printf("send email err: %s\n", err.Error())
				} else {
					logger.Print("send email success\n")
				}
			}
			return
		case context.Canceled:
			return
		}
		if count >= maxAttempts {
			break
		}
		logger.Printf("Tried %d times. Retry after %v, err: %s\n", count, 5*time.Minute, err.Error())

		if timer == nil {
			timer = time.NewTimer(5 * time.Minute)
		} else {
			timer.Reset(5 * time.Minute)
		}
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		}
	}

	if err := emailCfg.Send("form bot", "获取表单失败提示", "获取表单失败 err: "+err.Error()); err != nil {
		logger.Printf("Send message failed, err: %s\n", err.Error())
	}
	return fmt.Errorf("maximum attempts: %d reached with error: %w", maxAttempts, err)
}

func loadJson(v interface{}, name string) error {
	val := reflect.ValueOf(v)
	if val.CanAddr() && !val.Elem().IsZero() {
		val.Elem().Set(reflect.New(val.Elem().Type()))
	}
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := json.NewDecoder(f)

	return dec.Decode(v)
}

func (arr timeArray) Len() int {
	return len(arr)
}

func (arr timeArray) Less(x, y int) bool {
	return less(arr[x], arr[y])
}

func (arr timeArray) Swap(x, y int) {
	arr[x], arr[y] = arr[y], arr[x]
}

func less(t1, t2 timeSchedue) bool {
	if t1.Hour == t2.Hour {
		return t1.Minute < t2.Minute
	}
	return t1.Hour < t2.Hour
}

func nextTime() (time.Duration, bool) {
	now := time.Now().In(timeZone)
	hour, minute, _ := now.Clock()
	year, month, day := now.Date()
	n := timeSchedue{
		Hour:   uint8(hour),
		Minute: uint8(minute),
	}
	index := sort.Search(len(timeTable), func(i int) bool {
		return less(n, timeTable[i])
	})
	if index < len(timeTable) {
		if n == timeTable[index] { // not reachable
			return 2 * time.Second, timeTable[index].SendMail // 2 second
		}
		nextTime := time.Date(year, month, day, int(timeTable[index].Hour), int(timeTable[index].Minute), 0, 0, timeZone)
		d := nextTime.Sub(now)
		if d < 2*time.Second {
			d = 2 * time.Second
		}
		return d, timeTable[index].SendMail
	}
	nextTime := time.Date(year, month, day+1, int(timeTable[0].Hour), int(timeTable[0].Minute), 0, 0, timeZone)
	d := nextTime.Sub(now)
	if d < 2*time.Second {
		d = 2 * time.Second
	}
	return d, timeTable[0].SendMail
}
