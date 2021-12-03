package services

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"encoding/base64"
	"encoding/json"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/gin-gonic/gin"

	"github.com/kafka-protobuf-producer/config"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

type RequestBody struct {
	Data string `json:"data"`
}
type KafkaProducer struct {
	Producer        *kafka.Producer
	ProtobufService ProtoService
}

func NewKafkaProducer(configuration config.Configuration, protoServic ProtoService) KafkaProducer {
	p, _ := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": configuration.BootstrapServer,
		"client.id":         configuration.ClientId,
		"acks":              configuration.Acks})

	return KafkaProducer{Producer: p, ProtobufService: protoServic}
}

func (kp *KafkaProducer) ProduceRest(context *gin.Context) {
	responseBodyBytes := []byte(`success`)
	code := http.StatusOK
	topic := context.Param("topic")

	decoder := json.NewDecoder(context.Request.Body)
	var requestJson RequestBody
	err := decoder.Decode(&requestJson)
	if err != nil {
		context.Data(http.StatusBadRequest, gin.MIMEHTML, []byte(`failure : bad request `))
		return
	}

	data, err := base64.StdEncoding.DecodeString(requestJson.Data)
	if err != nil {
		context.Data(http.StatusBadRequest, gin.MIMEHTML, []byte(err.Error()))
		return
	}
	log.Printf("Message  :: %v", string(data))
	kafkaMessageType := context.GetHeader("X-Proto-Message-Type")

	if len(strings.TrimSpace(kafkaMessageType)) == 0 {
		log.Warnf("Missing Request Header `X-Proto-Message-Type`, producing plaintext message to topic : %v", topic)
		kp.Produce(topic, data)
	} else {
		protoNameWithPackage, err := kp.formatProtoWithPackageName(strings.TrimSpace(kafkaMessageType))

		if err != nil {
			context.Data(http.StatusBadRequest, gin.MIMEHTML, []byte(err.Error()))
			return
		}
		//convert to protobuf
		pm, err := kp.ProtobufService.ToProtoMessage(protoNameWithPackage, data)
		if err != nil {
			context.Data(http.StatusBadRequest, gin.MIMEHTML, []byte(err.Error()))
			return
		}
		peBytes, _ := proto.Marshal(pm)
		kp.Produce(topic, peBytes)
	}

	context.Data(code, gin.MIMEHTML, responseBodyBytes)
}

func (kp *KafkaProducer) Produce(topic string, message []byte) {

	// Go-routine to handle message delivery reports and
	// possibly other event types (errors, stats, etc)
	go func() {
		for e := range kp.Producer.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					log.Printf("Failed to deliver message: %v\n", ev.TopicPartition)
				} else {
					log.Printf("Successfully produced record to topic %s partition [%d] @ offset %v\n",
						*ev.TopicPartition.Topic, ev.TopicPartition.Partition, ev.TopicPartition.Offset)
				}
			}
		}
	}()

	delivery_chan := make(chan kafka.Event, 10000)
	defer close(delivery_chan)
	kp.Producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          message},
		delivery_chan,
	)

	e := <-delivery_chan
	m := e.(*kafka.Message)

	if m.TopicPartition.Error != nil {
		log.Printf("Delivery failed: %v\n", m.TopicPartition.Error)
	} else {
		log.Printf("Delivered message to topic %s partition [%d] at offset %v\n",
			*m.TopicPartition.Topic, m.TopicPartition.Partition, m.TopicPartition.Offset)
	}
}

func (kp *KafkaProducer) formatProtoWithPackageName(protoName string) (string, error) {
	fileDescriptor, ok := kp.ProtobufService.ProtoTypeMap[protoName]
	if ok {
		pkg := fileDescriptor.GetPackage()
		var replace = strings.Replace(protoName, "proto.", "", 1)
		replace = strings.Replace(replace, ".proto", "", 1)
		return fmt.Sprintf("%s.%s", pkg, replace), nil
	}
	return "", errors.New("Proto Not Registered : " + protoName)
}
