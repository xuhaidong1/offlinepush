package prometheus

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	defaultNameSpace = "xuhaidong"
	defaultSubsystem = "offlinepush"
)

var MessageGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: defaultNameSpace,
		Subsystem: defaultSubsystem,
		Name:      "MessageGauge",
	},
	[]string{"topic"},
)

func InitPrometheus() {
	prometheus.MustRegister(MessageGauge)
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(":8087", nil)
		if err != nil {
			panic(err)
		}
	}()
}
