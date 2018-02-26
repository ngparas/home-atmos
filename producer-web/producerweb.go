package main

import (
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

var (
	PORT              string = os.Getenv("PORT")
	BOOTSTRAP_SERVERS string = os.Getenv("BOOTSTRAP_SERVERS")
	TOPIC             string = os.Getenv("SENSOR_TOPIC")
)

func isJsonContent(h http.Header) bool {
	contentType, ok := h["Content-Type"]
	return ok && len(contentType) == 1 && contentType[0] == "application/json"
}

func makeProducerHandler(producerChannel chan<- *kafka.Message) func(http.ResponseWriter, *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			if isJsonContent(r.Header) {
				bodyBytes, err := ioutil.ReadAll(r.Body)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				fmt.Println(bodyBytes)

				producerChannel <- &kafka.Message{
					TopicPartition: kafka.TopicPartition{
						Topic:     &TOPIC,
						Partition: kafka.PartitionAny},
					Value: bodyBytes,
					Key:   []byte(r.RemoteAddr),
				}

				fmt.Fprintf(w, "OK")
			}
		} else {
			http.Error(w, "Only POST Requests allowed", http.StatusInternalServerError)
			return
		}
	}
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "OK")
}

func main() {

	time.Sleep(5 * time.Second) //sloppy but make sure kafka has time to start for now. KH would be so disappointed...

	p, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": BOOTSTRAP_SERVERS})

	if err != nil {
		fmt.Printf("Failed to create producer: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("Created Producer %v\n", p)

	doneChan := make(chan bool)

	go func() {
		defer close(doneChan)
		for e := range p.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				m := ev
				if m.TopicPartition.Error != nil {
					fmt.Printf("Delivery failed: %v\n", m.TopicPartition.Error)
				} else {
					fmt.Printf("Delivered message to topic %s [%d] at offset %v\n",
						*m.TopicPartition.Topic, m.TopicPartition.Partition, m.TopicPartition.Offset)
				}
				return

			default:
				fmt.Printf("Ignored event: %s\n", ev)
			}
		}
	}()

	// TODO - shut down properly, this isnt quite right
	defer func() {
		fmt.Println("Clearing events and Closing")
		// wait for delivery report goroutine to finish
		_ = <-doneChan
		p.Close()
	}()

	http.HandleFunc("/", makeProducerHandler(p.ProduceChannel()))
	http.HandleFunc("/health", healthCheck)

	http.ListenAndServe(":"+PORT, nil)
}
