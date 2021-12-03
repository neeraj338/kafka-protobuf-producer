package config

import (
	"os"

	log "github.com/sirupsen/logrus"
)

type Configuration struct {
	KafkaConfig
	ProtoDescFilePath string
}
type KafkaConfig struct {
	BootstrapServer string
	ClientId        string
	Acks            string
}

func ReadConfiguration() (Configuration, error) {
	log.Printf("Initialize configs.")
	configuration := Configuration{}

	//kafka
	configuration.KafkaConfig.BootstrapServer = GetEnv("BOOTSTRAP_SERVER", "localhost:9092")
	configuration.KafkaConfig.ClientId = GetEnv("CLIENT_ID", "goloang-clientid")
	configuration.KafkaConfig.Acks = GetEnv("ACKS", "1")

	configuration.ProtoDescFilePath = GetEnv("PROTO_DESC_PATH", "./proto/descriptor.pb")

	return configuration, nil
}

func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	log.Printf("Could not load environment var %s, using default %s", key, fallback)
	return fallback
}
