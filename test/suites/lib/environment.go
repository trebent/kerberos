package lib

import (
	"os"
	"strconv"
)

const (
	defaultHost                 = "localhost"
	defaultKerberosPort         = 30000
	defaultAdminPort            = 30001
	defaultMetricsPort          = 9464
	defaultJaegerReadAPIPort    = 16685
	defaultConnectorPort        = 30100
	defaultConnectorMetricsPort = 9462
)

func GetHost() string {
	hostVal, found := os.LookupEnv("KRB_FT_HOST")
	if !found {
		return defaultHost
	}

	return hostVal
}

func GetPort() int {
	val, found := os.LookupEnv("KRB_FT_PORT")
	if !found {
		return defaultKerberosPort
	}

	decoded, err := strconv.Atoi(val)
	if err != nil {
		return defaultKerberosPort
	}

	return decoded
}

func GetAdminPort() int {
	val, found := os.LookupEnv("KRB_FT_ADMIN_PORT")
	if !found {
		return defaultAdminPort
	}

	decoded, err := strconv.Atoi(val)
	if err != nil {
		return defaultAdminPort
	}

	return decoded
}

func GetMetricsPort() int {
	metricsPortVal, found := os.LookupEnv("KRB_FT_METRICS_PORT")
	if !found {
		return defaultMetricsPort
	}

	decodedMetricsPort, err := strconv.Atoi(metricsPortVal)
	if err != nil {
		return defaultMetricsPort
	}

	return decodedMetricsPort
}

func GetJaegerAPIPort() int {
	jaegerPortVal, found := os.LookupEnv("KRB_FT_JAEGER_PORT")
	if !found {
		return defaultJaegerReadAPIPort
	}

	decodedJaegerPort, err := strconv.Atoi(jaegerPortVal)
	if err != nil {
		return defaultJaegerReadAPIPort
	}

	return decodedJaegerPort
}

func GetConnectorPort() int {
	val, found := os.LookupEnv("KRB_FT_CONNECTOR_PORT")
	if !found {
		return defaultConnectorPort
	}

	decoded, err := strconv.Atoi(val)
	if err != nil {
		return defaultConnectorPort
	}

	return decoded
}

func GetConnectorMetricsPort() int {
	val, found := os.LookupEnv("KRB_FT_CONNECTOR_METRICS_PORT")
	if !found {
		return defaultConnectorMetricsPort
	}

	decoded, err := strconv.Atoi(val)
	if err != nil {
		return defaultConnectorMetricsPort
	}

	return decoded
}
