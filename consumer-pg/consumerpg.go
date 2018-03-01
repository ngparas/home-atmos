package main

// Parts of this file adapted from Confluent-Go Examples
// https://github.com/confluentinc/confluent-kafka-go/blob/master/examples/consumer_channel_example/consumer_channel_example.go
// which uses the Apache 2.0 License - http://www.apache.org/licenses/LICENSE-2.0

import (
	"encoding/json"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/go-pg/pg"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var (
	BOOTSTRAP_SERVERS string = os.Getenv("BOOTSTRAP_SERVERS")
	TOPICS            string = os.Getenv("SENSOR_TOPICS")
	CONSUMER_GROUP_ID string = os.Getenv("CONSUMER_GROUP_ID")
	POSTGRES_DATABASE string = os.Getenv("POSTGRES_DATABASE")
	POSTGRES_USERNAME string = os.Getenv("POSTGRES_USERNAME")
	POSTGRES_PASSWORD string = os.Getenv("POSTGRES_PASSWORD")
	POSTGRES_ADDRESS  string = os.Getenv("POSTGRES_ADDRESS")
)

type rawReading struct {
	DeviceId    string  `json:"deviceId"`
	SignalName  string  `json:"signalName"`
	EpochTime   int64   `json:"signalTime"`
	SignalValue float64 `json:"signalValue"`
	SignalTime  time.Time
}

func (rr *rawReading) ConvertIntoReading() *reading {
	return &reading{
		DeviceId:    rr.DeviceId,
		SignalName:  rr.SignalName,
		SignalTime:  time.Unix(rr.EpochTime, 0),
		SignalValue: rr.SignalValue,
	}
}

type reading struct {
	DeviceId    string
	SignalName  string
	SignalValue float64
	SignalTime  time.Time
}

func upsertMessage(db *pg.DB, m *kafka.Message) {
	var msg rawReading
	json.Unmarshal(m.Value, &msg)
	fmt.Println(msg)
	_, err := db.Model(msg.ConvertIntoReading()).OnConflict("(device_id, signal_name, signal_time) DO UPDATE").Set("signal_value = EXCLUDED.signal_value").Insert()
	if err != nil {
		fmt.Println("Error Inserting to Postgres")
	}
}

func main() {
	time.Sleep(10 * time.Second)

	db := pg.Connect(&pg.Options{
		User:     POSTGRES_USERNAME,
		Password: POSTGRES_PASSWORD,
		Database: POSTGRES_DATABASE,
		Addr:     POSTGRES_ADDRESS,
	})

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":               BOOTSTRAP_SERVERS,
		"group.id":                        CONSUMER_GROUP_ID,
		"session.timeout.ms":              6000,
		"go.events.channel.enable":        true,
		"go.application.rebalance.enable": true,
		"default.topic.config":            kafka.ConfigMap{"auto.offset.reset": "earliest"}})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create consumer: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created Consumer %v\n", c)

	err = c.SubscribeTopics(strings.Split(TOPICS, ","), nil)

	run := true

	for run == true {
		select {
		case sig := <-sigchan:
			fmt.Printf("Caught signal %v: terminating\n", sig)
			run = false

		case ev := <-c.Events():
			switch e := ev.(type) {
			case kafka.AssignedPartitions:
				fmt.Fprintf(os.Stderr, "%% %v\n", e)
				c.Assign(e.Partitions)
			case kafka.RevokedPartitions:
				fmt.Fprintf(os.Stderr, "%% %v\n", e)
				c.Unassign()
			case *kafka.Message:
				// fmt.Printf("%% Message on %s:\n%s\n",
				//     e.TopicPartition, string(e.Value))
				upsertMessage(db, e)
			case kafka.PartitionEOF:
				fmt.Printf("%% Reached %v\n", e)
			case kafka.Error:
				fmt.Fprintf(os.Stderr, "%% Error: %v\n", e)
				run = false
			}
		}
	}

	fmt.Printf("Closing consumer\n")
	c.Close()
}
