package main

import (
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	log "github.com/sirupsen/logrus"
)

func slowProducer(wg *sync.WaitGroup) {
	defer wg.Done()
	producer, err := sarama.NewAsyncProducer([]string{"localhost:9092"}, nil)
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := producer.Close(); err != nil {
			log.Fatalf("Error closing producer: %s", err.Error())
		}
	}()

	// Trap SIGINT to trigger a shutdown.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	counter := 0
	var enqueued, producerErrors int
ProducerLoop:
	for {
		select {
		case producer.Input() <- &sarama.ProducerMessage{Topic: "foo", Key: nil, Value: sarama.StringEncoder("testing 123")}:
			enqueued++
			counter++
			if counter >= 50 {
				counter = 0
				time.Sleep(1 * time.Second)
				log.Debug("Pausing producer for one second to throttle message production")
			}
		case err := <-producer.Errors():
			log.Infof("Error attempting to produce message: %s", err)
			producerErrors++
		case <-signals:
			break ProducerLoop
		}
	}

	log.Infof("Enqueued: %d; errors: %d\n", enqueued, producerErrors)
}
