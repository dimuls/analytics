package web

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type Store interface {
	AddMetric(metric string)
}

type Server struct {
	httpServer *http.Server
	store      Store
	bindAddr   string

	wg  sync.WaitGroup
	log *logrus.Entry
}

func NewServer(s Store, bindAddr string) *Server {
	return &Server{
		store:    s,
		bindAddr: bindAddr,
		log:      logrus.WithField("subsystem", "web_server"),
	}
}

func (s *Server) Start() {
	s.httpServer = &http.Server{
		Addr:    s.bindAddr,
		Handler: &handler{store: s.store},
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		err := s.httpServer.ListenAndServe()
		if err != nil {
			if err == http.ErrServerClosed {
				return
			}
			s.log.WithError(err).Error("failed to start http server")
		}
	}()
}

func (s *Server) Stop() {
	ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancel()

	err := s.httpServer.Shutdown(ctx)
	if err != nil {
		s.log.WithError(err).Error("failed to stop http server")
		return
	}

	s.wg.Wait()
	s.log.Info("stopped")
}
