package ft

import (
	"os"
	"strconv"
)

const (
	defaultHost              = "localhost"
	defaultKerberosPort      = 30000
	defaultMetricsPort       = 9464
	defaultJaegerReadAPIPort = 16685
)

func getPort() int {
	val, found := os.LookupEnv("KRB_FT_PORT")
	if !found {
		return defaultKerberosPort
	}

	decoded, err := strconv.Atoi(val)
	if err != nil {
		return defaultKerberosPort
	} else {
		return decoded
	}
}

func getHost() string {
	hostVal, found := os.LookupEnv("KRB_FT_HOST")
	if !found {
		return defaultHost
	} else {
		return hostVal
	}
}

func getMetricsPort() int {
	metricsPortVal, found := os.LookupEnv("KRB_FT_METRICS_PORT")
	if !found {
		return defaultMetricsPort
	}

	decodedMetricsPort, err := strconv.Atoi(metricsPortVal)
	if err != nil {
		return defaultMetricsPort
	} else {
		return decodedMetricsPort
	}
}

func getJaegerAPIPort() int {
	jaegerPortVal, found := os.LookupEnv("KRB_FT_JAEGER_PORT")
	if !found {
		return defaultJaegerReadAPIPort
	}

	decodedJaegerPort, err := strconv.Atoi(jaegerPortVal)
	if err != nil {
		return defaultJaegerReadAPIPort
	} else {
		return decodedJaegerPort
	}
}
