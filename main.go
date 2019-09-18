package main

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/dimuls/analytics/file"
	"github.com/dimuls/analytics/web"
)

func main() {
	fs, err := file.NewStore("store.txt", 5*time.Second, 50000)
	if err != nil {
		logrus.WithError(err).Fatal("failed to create file store")
	}
	fs.Start()

	ws := web.NewServer(fs, ":8080")
	ws.Start()

	ss := make(chan os.Signal)
	signal.Notify(ss, os.Interrupt, syscall.SIGTERM)

	for s := range ss {
		logrus.Infof("captured %v signal, stopping", s)

		st := time.Now()

		wg := sync.WaitGroup{}

		wg.Add(1)
		go func() {
			defer wg.Done()
			fs.Stop()
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			ws.Stop()
		}()

		wg.Wait()

		et := time.Now()

		logrus.Infof("stopped in %g seconds, exiting",
			et.Sub(st).Seconds())
		break
	}
}
