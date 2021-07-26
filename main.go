package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/avast/retry-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/bemasher/rtlamr/protocol"
	"github.com/bemasher/rtltcp"

	"github.com/bemasher/rtlamr/idm"
	"github.com/bemasher/rtlamr/netidm"
	"github.com/bemasher/rtlamr/r900"
	"github.com/bemasher/rtlamr/r900bcd"
	"github.com/bemasher/rtlamr/scm"
	"github.com/bemasher/rtlamr/scmplus"
)

var (
	consumption = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "rtlamr_consumption",
	}, []string{
		"id",
		"type",
	})
)

func main() {
	prometheus.MustRegister(consumption)
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":9090", nil)

	d := protocol.NewDecoder()
	rcvr := new(rtltcp.SDR)

	err := retry.Do(func() error {
		return rcvr.Connect(nil)
	})
	if err != nil {
		panic(err)
	}

	d.Allocate()

	go func() {
		for {
			block := make([]byte, d.Cfg.BlockSize2)
			_, err := io.ReadFull(rcvr, block)
			if err != nil {
				if err == io.EOF {
					panic(err)
				}
				log.Printf("error reading data block: %v", err)
				continue
			}

			for msg := range d.Decode(block) {
				val := float64(0)
				switch r := msg.(type) {
				case scm.SCM:
					val = float64(r.Consumption)
				case scmplus.SCM:
					val = float64(r.Consumption)
				case r900.R900:
					val = float64(r.Consumption)
				case r900bcd.R900BCD:
					val = float64(r.Consumption)
				case netidm.NetIDM:
					val = float64(r.LastConsumption)
				case idm.IDM:
					val = float64(r.LastConsumptionCount)
				default:
					log.Printf("Unknown message type: %#v", r)
					continue
				}

				consumption.WithLabelValues(
					strconv.Itoa(int(msg.MeterID())),
					strconv.Itoa(int(msg.MeterType())),
				).Set(val)
			}

		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	log.Println("Received Signal:", sig)
}
