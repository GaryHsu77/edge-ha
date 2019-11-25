package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
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
}

type mqttConf struct {
	Mqtt mqttClient `json:"mqtt"`
}

type out struct {
	SrcName string `json:"srcNmae"`
	TagName string `json:"tagNmae"`
	Value   int    `json:"value"`
	Ts      string `json:"timestamp"`
}

var version = "0.0.7"

func main() {
	if len(os.Args) <= 1 {
		fmt.Println(version)
		return
	}

	configPath := os.Args[1]
	log.Println(configPath)

	dir, _ := path.Split(configPath)

	modbusData, err := ioutil.ReadFile(filepath.Join(dir, "modbus.json"))
	if err != nil {
		panic(err)
	}
	mqttData, err := ioutil.ReadFile(filepath.Join(dir, "mqtt.json"))
	if err != nil {
		panic(err)
	}

	var modbusC Conf
	if err := json.Unmarshal(modbusData, &modbusC); err != nil {
		panic(err)
	}
	var mqttC mqttConf
	if err := json.Unmarshal(mqttData, &mqttC); err != nil {
		panic(err)
	}
	log.Println("[*] configuration load success")

	var mqttClient MQTT.Client
	for {
		if mqttClient != nil {
			log.Println("disconnect mqtt client")
			mqttClient.Disconnect(250)
		}

		// mqtt >>>
		mqttClient, err = mqttConnect(mqttC)
		if err != nil {
			log.Println("[error] create mqtt client failed")
			time.Sleep(1 * time.Second)
			continue
		}

		log.Println("[*] mqtt broker connected")

		// modbus >>>
		modbusClient, handler := modbusConnect(modbusC)
		if modbusClient == nil || handler == nil {
			log.Println("[error] create modbus client failed")
			time.Sleep(1 * time.Second)
			continue
		}
		log.Println("[*] modbus connected")

		// run
		run(mqttC, modbusC, modbusClient, handler, mqttClient)

		// stop
		time.Sleep(1 * time.Second)
		handler.Close()
	}
}

func run(mqttC mqttConf, modbusC Conf, modbusClient modbus.Client, handler *modbus.TCPClientHandler, mqttClient MQTT.Client) {
	for {
		for _, t := range modbusC.Tags {
			log.Printf("[*] tag[%s:%s] polling", t.SrcName, t.TagName)
			results, err := modbusClient.ReadHoldingRegisters(uint16(t.Addr), uint16(t.Qty))
			if err != nil {
				log.Printf("polling tag:%s failed, err:%s", t.TagName, err.Error())
				return
			}

			valueBytes := []byte{0, 0, 0, 0}
			for i := range results {
				valueBytes[i] = results[i]
			}
			i := out{
				t.SrcName,
				t.TagName,
				int(binary.BigEndian.Uint32(valueBytes)),
				time.Now().Format(time.RFC3339),
			}
			o, _ := json.Marshal(i)
			token := mqttClient.Publish(fmt.Sprintf("devs/%s/tags/%s", mqttC.Mqtt.ClientID, t.TagName), byte(mqttC.Mqtt.Qos), false, o)
			token.Wait()
			token = mqttClient.Publish(fmt.Sprintf("devs/%s/status", mqttC.Mqtt.ClientID), byte(mqttC.Mqtt.Qos), false, `{"value":1}`)
			token.Wait()
			time.Sleep(time.Duration(modbusC.Modbus.IntervalSec) * time.Second)
		}
	}
}

func modbusConnect(conf Conf) (modbus.Client, *modbus.TCPClientHandler) {
	handler := modbus.NewTCPClientHandler(
		fmt.Sprintf("%s:%d", conf.Modbus.Host, conf.Modbus.Port))
	handler.Timeout = 1 * time.Second
	handler.SlaveId = byte(conf.Modbus.DeviceID)
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

func mqttConnect(conf mqttConf) (MQTT.Client, error) {
	opts := MQTT.NewClientOptions()
	opts.AddBroker(conf.Mqtt.Addr)
	opts.SetKeepAlive(100 * time.Millisecond)
	opts.SetConnectTimeout(3 * time.Second)
	opts.SetCleanSession(conf.Mqtt.CleanSession)
	opts.SetCleanSession(true)
	opts.SetWill(fmt.Sprintf("devs/%s/status", conf.Mqtt.ClientID), `{"value":0}`, byte(conf.Mqtt.Qos), true)
	mqttClient := MQTT.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}
	return mqttClient, nil
}
