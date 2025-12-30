package ft

import (
	"net/http"
	"os"
	"strconv"
	"time"
)

var (
	port              = 0
	metricsPort       = 0
	jaegerReadAPIPort = 0
	host              = ""
	client            = &http.Client{}
)

const (
	defaultHost              = "localhost"
	defaultKerberosPort      = 30000
	defaultMetricsPort       = 9464
	defaultJaegerReadAPIPort = 16685
	defaultClientTimeout     = 4 * time.Second
)

func init() {
	client.Timeout = defaultClientTimeout

	val, found := os.LookupEnv("KRB_FT_PORT")
	if !found {
		port = defaultKerberosPort
	}

	decoded, err := strconv.Atoi(val)
	if err != nil {
		port = defaultKerberosPort
	} else {
		port = decoded
	}

	hostVal, found := os.LookupEnv("KRB_FT_HOST")
	if !found {
		host = defaultHost
	} else {
		host = hostVal
	}

	metricsPortVal, found := os.LookupEnv("KRB_FT_METRICS_PORT")
	if !found {
		metricsPort = defaultMetricsPort
	}

	decodedMetricsPort, err := strconv.Atoi(metricsPortVal)
	if err != nil {
		metricsPort = defaultMetricsPort
	} else {
		metricsPort = decodedMetricsPort
	}

	jaegerPortVal, found := os.LookupEnv("KRB_FT_JAEGER_PORT")
	if !found {
		jaegerReadAPIPort = defaultJaegerReadAPIPort
	}

	decodedJaegerPort, err := strconv.Atoi(jaegerPortVal)
	if err != nil {
		jaegerReadAPIPort = defaultJaegerReadAPIPort
	} else {
		jaegerReadAPIPort = decodedJaegerPort
	}
}
