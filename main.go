package main

import (
	"flag"
	"fmt"
	owm "github.com/briandowns/openweathermap"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
)

type mqttPayload struct {
	topic   string
	payload string
	retain  bool
}

func main() {
	var appid string
	var location string
	var broker string
	var port int
	var username string
	var password string
	var deviceName string

	// Command line arguments.
	flag.IntVar(&port, "port", 1883, "MQTT port")
	flag.StringVar(&appid, "apikey", "", "OpenWeathermap API key")
	flag.StringVar(&location, "location", "", "Location, for example Moscow,RU")
	flag.StringVar(&broker, "broker", "127.0.0.1", "MQTT broker address")
	flag.StringVar(&username, "username", "", "MQTT broker username")
	flag.StringVar(&password, "password", "", "MQTT broker password")
	flag.StringVar(&deviceName, "device", "", "device prefix for MQTT topic")
	flag.Parse()
	deviceLocation := "OpenWeatherMap " + location

	// Validate commandline parameters
	if appid == "" {
		log.Fatalln("OpenWeatherMap API key is empty. Please check for command line options")
	}

	if location == "" {
		log.Fatalln("Location is not specified. Please check for command line options")
	}

	if deviceName == "" {
		log.Fatalln("WirenBoard device not specified. Please check for command line options")
	}

	fmt.Printf("Broker: %s port %d\n", broker, port)
	fmt.Printf("OpenWeatherMap: %s Location: %s Device: %s\n", appid, location, deviceName)

	// OpenWeatherMap API
	w, err := owm.NewCurrent("C", "en", appid)
	if err != nil {
		log.Fatalln("OpenWeatherMap Error: ", err)
	}

	// Get OpenWeatherMap info
	err = w.CurrentByName(location)
	if err != nil {
		log.Fatalln("OpenWeatherMap Error: ", err)
	}

	// Connect to MQTT
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", broker, port))
	opts.SetClientID("go_mqtt_client")
	if username != "" {
		opts.SetUsername(username)
		opts.SetPassword(password)
	}

	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalln("MQTT Error: ", token.Error())
	}

	var topicPrefix string
	topicPrefix = "devices/" + deviceName

	//Topics for publish to Wirenboard
	topics := []mqttPayload{
		{topic: topicPrefix + "/meta/name", payload: deviceLocation, retain: true},
		{topic: topicPrefix + "/controls/temperature", payload: fmt.Sprintf("%f", w.Main.Temp), retain: true},
		{topic: topicPrefix + "/controls/temperature/meta/readonly", payload: "1", retain: true},
		{topic: topicPrefix + "/controls/temperature/meta/type", payload: "temperature", retain: true},
		{topic: topicPrefix + "/controls/humidity", payload: fmt.Sprintf("%d", w.Main.Humidity), retain: true},
		{topic: topicPrefix + "/controls/humidity/meta/readonly", payload: "1", retain: true},
		{topic: topicPrefix + "/controls/humidity/meta/type", payload: "rel_humidity", retain: true},
	}

	for _, topic := range topics {
		token := client.Publish(topic.topic, 0, topic.retain, topic.payload)
		topic := topic
		go func() {
			<-token.Done()
			if token.Error() != nil {
				log.Fatalln(token.Error())
			}
			log.Println("successfully published topic: " + topic.topic)
		}()
	}

	client.Disconnect(250)
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connect lost: %v\n", err)
}
