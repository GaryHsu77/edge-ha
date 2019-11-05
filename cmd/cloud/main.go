package main

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/tidwall/gjson"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	appName     = "Mosquitto exporter"
	bindAddress = "0.0.0.0:2112"
	endpoint    = "tcp://127.0.0.1:1883"
)

var (
	counterKeyMetrics = map[string]string{
		"devs/+/status": "The total number of bytes received since the broker started.",
		"devs/+/tags/+": "The total number of bytes received since the broker started.",
	}
	counterMetrics = map[string]*MosquittoCounter{}
	jsonMetrics    = map[string]*prometheus.Desc{}
)

func main() {
	log.Printf("Starting mosquitto_broker")

	opts := mqtt.NewClientOptions()
	opts.SetCleanSession(true)
	opts.AddBroker(endpoint)

	opts.OnConnect = func(client mqtt.Client) {
		log.Printf("Connected to %s", endpoint)
		// subscribe on every (re)connect
		token := client.Subscribe("#", 0, func(_ mqtt.Client, msg mqtt.Message) {
			processUpdate(msg.Topic(), string(msg.Payload()))
		})
		if !token.WaitTimeout(10 * time.Second) {
			log.Println("Error: Timeout subscribing to topic $SYS/#")
		}
		if err := token.Error(); err != nil {
			log.Printf("Failed to subscribe to topic $SYS/#: %s", err)
		}
	}
	opts.OnConnectionLost = func(client mqtt.Client, err error) {
		log.Printf("Error: Connection to %s lost: %s", endpoint, err)
	}
	client := mqtt.NewClient(opts)

	// try to connect forever
	for {
		token := client.Connect()
		if token.WaitTimeout(5 * time.Second) {
			if token.Error() == nil {
				break
			}
			log.Printf("Error: Failed to connect to broker: %s", token.Error())
		} else {
			log.Printf("Timeout connecting to endpoint %s", endpoint)
		}
		time.Sleep(5 * time.Second)
	}

	// init the router and server
	http.Handle("/metrics", promhttp.Handler())
	log.Printf("Listening on %s...", bindAddress)
	err := http.ListenAndServe(bindAddress, nil)
	fatalfOnError(err, "Failed to bind on %s: ", bindAddress)
}

// $SYS/broker/bytes/received
func processUpdate(topic, payload string) {
	log.Printf("Got broker update with topic %s and data %s", topic, payload)
	arr := strings.Split(topic, "/")
	if len(arr) < 3 {
		log.Printf("invalid data %s:%s", topic, payload)
		return
	}
	switch arr[2] {
	case "status":
	case "tags":
	}
	processCounterMetric(topic, payload)
}

func processCounterMetric(topic, payload string) {
	if counterMetrics[topic] != nil {
		value := parseValue(payload)
		counterMetrics[topic].Set(value)
	} else {
		// create a mosquitto counter pointer
		mCounter := NewMosquittoCounter(prometheus.NewDesc(
			parseTopic(topic),
			topic,
			[]string{},
			prometheus.Labels{},
		))

		// save it
		counterMetrics[topic] = mCounter
		// register the metric
		prometheus.MustRegister(mCounter)
		// add the first value
		value := parseValue(payload)
		counterMetrics[topic].Set(value)
	}
}

func parseTopic(topic string) string {
	name := strings.Replace(topic, "/", "_", -1)
	name = strings.Replace(name, " ", "_", -1)
	name = strings.Replace(name, "-", "_", -1)
	name = strings.Replace(name, ".", "_", -1)
	return name
}

func parseValue(payload string) float64 {
	// fmt.Printf("Payload %s \n", payload)
	return float64(gjson.Get(payload, "value").Int())
}

func fatalfOnError(err error, msg string, args ...interface{}) {
	if err != nil {
		log.Fatalf(msg, args...)
	}
}
