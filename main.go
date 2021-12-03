package main

import (
	"github.com/gin-gonic/gin"
	"github.com/kafka-protobuf-producer/config"
	service "github.com/kafka-protobuf-producer/services"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.SetFormatter(createJsonFormatter())

	configuration, _ := config.ReadConfiguration()

	protoService := service.NewProtoService(configuration)
	//register proto descriptor
	protoService.RegisterProtoFile()

	kafkaProducer := service.NewKafkaProducer(configuration, protoService)

	server := gin.Default()

	server.GET("/protos", protoService.GetAllProto)
	server.GET("/protos/:name/describe", protoService.DescribeProto)

	server.POST("/produces/:topic", kafkaProducer.ProduceRest)
	server.Run(":8080")
}

func createJsonFormatter() log.Formatter {
	return &log.JSONFormatter{
		FieldMap: log.FieldMap{
			log.FieldKeyMsg: "message",
		},
	}
}
