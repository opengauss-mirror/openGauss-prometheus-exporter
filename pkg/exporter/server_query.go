// Copyright © 2021 Bin Liu <bin.liu@enmotech.com>

package exporter

import (
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"strings"
	"sync"
	"time"
)

type metricError struct {
	lock   sync.Mutex
	Errors map[string]error
	Count  int64
}

func (e *metricError) addError(metricName string, err error) {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.Errors[metricName] = err
	e.Count++
}

// 查询监控指标. 先判断是否读取缓存. 禁用缓存或者缓存超时,则读取数据库
func (s *Server) queryMetrics(ch chan<- prometheus.Metric) map[string]error {
	metricErrors := &metricError{
		Errors: map[string]error{},
		Count:  0,
	}
	wg := sync.WaitGroup{}
	limit := newRateLimit(s.parallel)
	for _, queryInstance := range s.queryInstanceMap {
		metricName := queryInstance.Name
		wg.Add(1)
		limit.getToken()
		go func(queryInst *QueryInstance) {
			defer wg.Done()
			defer limit.putToken()
			err := s.queryMetric(ch, queryInst)
			if err != nil {
				// 存在并发写入问题. 改成结构体加锁
				metricErrors.addError(metricName, err)
			}
		}(queryInstance)

	}
	wg.Wait()

	s.ScrapeErrorCount = metricErrors.Count
	return metricErrors.Errors
}

func (s *Server) queryMetric(ch chan<- prometheus.Metric, queryInstance *QueryInstance) error {
	var (
		metricName     = queryInstance.Name
		scrapeMetric   = false // Whether to collect indicators from the database 是否从数据库里采集指标
		cachedMetric   = &cachedMetrics{}
		metrics        []prometheus.Metric
		nonFatalErrors []error
		err            error
	)

	querySQL := queryInstance.GetQuerySQL(s.lastMapVersion, s.primary)
	if querySQL == nil {
		log.Errorf("Collect Metric %s not define querySQL for version %s on %s database ", metricName, s.lastMapVersion.String(), s.DBRole())
		return nil
	}
	if strings.EqualFold(querySQL.Status, statusDisable) {
		log.Debugf("Collect Metric %s disable. skip", metricName)
		return nil
	}

	// 记录采集总个数
	s.ScrapeTotalCount++

	// Determine whether to enable caching and cache expiration 判断是否启用缓存和缓存过期
	if !s.disableCache {
		var found bool
		// Check if the metric is cached
		s.cacheMtx.Lock()
		cachedMetric, found = s.metricCache[metricName]
		s.cacheMtx.Unlock()
		// If found, check if needs refresh from cache
		if !found {
			scrapeMetric = true
		} else if !cachedMetric.IsValid(querySQL.TTL) {
			scrapeMetric = true
		}
		if cachedMetric != nil && (len(cachedMetric.nonFatalErrors) > 0 || len(cachedMetric.metrics) == 0) {
			scrapeMetric = true
		}
	} else {
		scrapeMetric = true
	}
	if scrapeMetric {
		metrics, nonFatalErrors, err = s.doCollectMetric(queryInstance)
	} else {
		log.Debugf("Collect Metric [%s] use cache", metricName)
		metrics, nonFatalErrors = cachedMetric.metrics, cachedMetric.nonFatalErrors
	}

	// Serious error - a namespace disappeared
	if err != nil {
		nonFatalErrors = append(nonFatalErrors, err)
		log.Errorf("Collect Metric [%s] err %s", metricName, err)
	}
	// Non-serious errors - likely version or parsing problems.
	if len(nonFatalErrors) > 0 {
		var errText string
		for _, err := range nonFatalErrors {
			log.Errorf("Collect Metric [%s] nonFatalErrors err %s", metricName, err)
			errText += err.Error()
		}
		err = errors.New(errText)
	}

	// Emit the metrics into the channel
	for _, m := range metrics {
		ch <- m
	}

	if scrapeMetric && queryInstance.TTL > 0 {
		// Only cache if metric is meaningfully cacheable
		s.cacheMtx.Lock()
		s.metricCache[metricName] = &cachedMetrics{
			metrics:        metrics,
			lastScrape:     time.Now(), // 改为查询完时间
			nonFatalErrors: nonFatalErrors,
		}
		s.cacheMtx.Unlock()
	}
	return err
}
