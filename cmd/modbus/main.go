package main

import (
	"fmt"
	"log"
	"os"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/goburrow/modbus"
)

type tag struct {
	SrcName   string `json:"srcNmae"`
	TagName   string `json:"tagNmae"`
	ValueType string `json:"valueType"`
	Addr      int    `json:"addr"`
	Qty       int    `json:"qty"`
}
type modbusClient struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	DeviceID    int    `json:"deviceId"`
	IntervalSec int    `json:"intervalSec"`
}
type mqttClient struct {
	Addr         string `json:"addr"`
	Topic        string `json:"topic"`
	ClientID     string `json:"clientId"`
	CleanSession bool   `json:"cleanSession"`
	Qos          int    `json:"qos"`
}

// Conf slave configuration
type Conf struct {
	Tags   []tag        `json:"tags"`
	Modbus modbusClient `json:"modbus"`
	Mqtt   mqttClient   `json:"mqtt"`
}

type out struct {
	SrcName string `json:"srcNmae"`
	TagName string `json:"tagNmae"`
	Value   int    `json:"value"`
	Ts      string `json:"timestamp"`
}

var version = "0.0.5"

func main() {
	if len(os.Args) <= 1 {
		fmt.Println(version)
		return
	}

	modbusIP := os.Args[1]
	log.Println(modbusIP)

	go func() {
		for {
			// modbus >>>
			modbusClient, handler := modbusConnect(modbusIP)
			if modbusClient == nil || handler == nil {
				log.Println("[error] create modbus client failed")
				time.Sleep(1 * time.Second)
				continue
			}
			log.Println("[*] modbus connected")

			// run
			runAdd(modbusClient, handler, 0, 1, 255)

			// stop
			time.Sleep(1 * time.Second)
			handler.Close()
		}
	}()

	go func() {
		for {
			// modbus >>>
			modbusClient, handler := modbusConnect(modbusIP)
			if modbusClient == nil || handler == nil {
				log.Println("[error] create modbus client failed")
				time.Sleep(1 * time.Second)
				continue
			}
			log.Println("[*] modbus connected")

			// run
			runHeat(modbusClient, handler, 1, 1, 255)

			// stop
			time.Sleep(500 * time.Millisecond)
			handler.Close()
		}
	}()

	go func() {
		var mqttClient MQTT.Client
		choke := make(chan [2]string)
		for {
			if mqttClient != nil {
				log.Println("disconnect mqtt client")
				mqttClient.Disconnect(250)
			}
			opts := MQTT.NewClientOptions()
			opts.AddBroker("localhost:1883")
			opts.SetConnectTimeout(3 * time.Second)
			opts.SetCleanSession(true)
			opts.SetDefaultPublishHandler(func(client MQTT.Client, msg MQTT.Message) {
				choke <- [2]string{msg.Topic(), string(msg.Payload())}
			})
			mqttClient := MQTT.NewClient(opts)
			if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
				continue
			}

			if token := mqttClient.Subscribe("devs/mydev1/tags/+", byte(0), nil); token.Wait() && token.Error() != nil {
				fmt.Println(token.Error())
				os.Exit(1)
			}
			for {
				select {
				case <-choke:
					continue
				case <-time.After(2 * time.Second):
					token := mqttClient.Publish("devs/mydev1/status", byte(0), false, `{"value":0}`)
					token.Wait()
				}
			}
		}
	}()

	go func() {
		var mqttClient MQTT.Client
		choke := make(chan [2]string)
		for {
			if mqttClient != nil {
				log.Println("disconnect mqtt client")
				mqttClient.Disconnect(250)
			}
			opts := MQTT.NewClientOptions()
			opts.AddBroker("localhost:1883")
			opts.SetConnectTimeout(3 * time.Second)
			opts.SetCleanSession(true)
			opts.SetDefaultPublishHandler(func(client MQTT.Client, msg MQTT.Message) {
				choke <- [2]string{msg.Topic(), string(msg.Payload())}
			})
			mqttClient := MQTT.NewClient(opts)
			if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
				continue
			}

			if token := mqttClient.Subscribe("devs/mydev2/tags/+", byte(0), nil); token.Wait() && token.Error() != nil {
				fmt.Println(token.Error())
				os.Exit(1)
			}
			for {
				select {
				case <-choke:
					continue
				case <-time.After(3 * time.Second):
					token := mqttClient.Publish("devs/mydev2/status", byte(0), false, `{"value":0}`)
					token.Wait()
				}
			}
		}
	}()

	c := make(chan bool)
	<-c
}

func runHeat(modbusClient modbus.Client, handler *modbus.TCPClientHandler, addr, count, max int) {
	count = max
	for {
		_, err := modbusClient.WriteSingleRegister(uint16(addr), uint16(count))
		if err != nil {
			log.Printf("write failed, err:%s", err.Error())
			return
		}
		count--
		if count == 0 {
			count = max
		}
		time.Sleep(1 * time.Second)
	}
}

func runAdd(modbusClient modbus.Client, handler *modbus.TCPClientHandler, addr, count, max int) {
	count = 1
	for {
		_, err := modbusClient.WriteSingleRegister(uint16(addr), uint16(count))
		if err != nil {
			log.Printf("write failed, err:%s", err.Error())
			return
		}
		count++
		if count == max {
			count = 1
		}
		time.Sleep(1 * time.Second)
	}
}

func modbusConnect(ip string) (modbus.Client, *modbus.TCPClientHandler) {
	handler := modbus.NewTCPClientHandler(
		fmt.Sprintf("%s:502", ip))
	handler.Timeout = 1 * time.Second
	handler.SlaveId = byte(1)
	handler.Logger = log.New(os.Stdout, "modbus debug: ", log.LstdFlags)
	if err := handler.Connect(); err != nil {
		return nil, nil
	}
	defer handler.Close()
	modbusClient := modbus.NewClient(handler)
	if modbusClient == nil {
		return nil, nil
	}
	return modbusClient, handler
}
