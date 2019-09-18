package file

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type Store struct {
	filePath   string
	syncPeriod time.Duration

	metrics   []string
	metricsMx sync.Mutex

	stop chan struct{}
	wg   sync.WaitGroup

	log *logrus.Entry
}

func NewStore(filePath string, syncPeriod time.Duration, targetRPS int) (
	*Store, error) {
	syncPeriodSecs := syncPeriod.Seconds()

	targetMetricsCap := 1.2 * syncPeriodSecs * float64(targetRPS)

	if targetMetricsCap < 1 {
		return nil, errors.New(
			"computed target metrics cap is less than one")
	}

	return &Store{
		filePath:   filePath,
		syncPeriod: syncPeriod,
		metrics:    make([]string, 0, int(targetMetricsCap)),
		log:        logrus.WithField("subsystem", "file_store"),
	}, nil
}

func (s *Store) Start() {
	s.stop = make(chan struct{})
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		t := time.NewTicker(s.syncPeriod)

		for {
			select {
			case <-t.C:
				s.sync()
			case <-s.stop:
				t.Stop()
				return
			}
		}
	}()
}

func (s *Store) Stop() {
	close(s.stop)
	s.wg.Wait()
	s.log.Info("stopped")
}

func (s *Store) AddMetric(metric string) {
	s.metricsMx.Lock()
	defer s.metricsMx.Unlock()
	s.metrics = append(s.metrics, metric)
}

func (s *Store) sync() {
	defer func() {
		if r := recover(); r != nil {
			s.log.WithFields(logrus.Fields{
				"recover": r,
				"stack":   string(debug.Stack()),
			}).Error("panic occurred during sync")
		}
	}()

	s.metricsMx.Lock()

	if len(s.metrics) == 0 {
		s.metricsMx.Unlock()
		return
	}

	metricsSnapshot := s.metrics

	s.metrics = make([]string, 0, cap(s.metrics))

	s.metricsMx.Unlock()

	f, err := os.OpenFile(s.filePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE,
		0666)
	if err != nil {
		s.log.WithError(err).Error("failed to open file")
	}

	for _, m := range metricsSnapshot {
		fmt.Fprintln(f, m)
	}

	err = f.Sync()
	if err != nil {
		s.log.WithError(err).Error("failed to sync file")
	}

	err = f.Close()
	if err != nil {
		s.log.WithError(err).Error("failed to close file")
	}

}
