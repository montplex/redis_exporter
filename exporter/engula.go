package exporter

import (
	"github.com/gomodule/redigo/redis"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"strconv"
	"strings"
)

func (e *Exporter) extractEngulaMetrics(ch chan<- prometheus.Metric, c redis.Conn) {
	reply, err := redis.String(doRedisCmd(c, "ENGULA", "INFO"))
	if err != nil {
		log.Errorf("ENGULA INFO err: %s", err)
		return
	}

	for _, line := range strings.Split(reply, "\r\n") {
		if strings.Contains(line, "#") {
			continue
		}

		split := strings.Split(line, ":")
		if len(split) != 2 {
			continue
		}

		key := split[0]
		value, err := strconv.ParseFloat(split[1], 64)
		if err != nil {
			continue
		}

		e.registerConstMetricGauge(ch, key, value)
	}
}
