package services

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang/protobuf/jsonpb"
	"github.com/kafka-protobuf-producer/config"
	response "github.com/kafka-protobuf-producer/model"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

type ProtoService struct {
	Configuration config.Configuration
	ProtoTypeMap  map[string]descriptorpb.FileDescriptorProto
}

func NewProtoService(configuration config.Configuration) ProtoService {
	return ProtoService{Configuration: configuration, ProtoTypeMap: make(map[string]descriptorpb.FileDescriptorProto)}
}

func (p *ProtoService) DescribeProto(context *gin.Context) {
	protoName := context.Param("name")
	log.Printf("filter for name :: %v", protoName)
	fd, ok := p.ProtoTypeMap[protoName]
	if ok {
		code := http.StatusOK
		context.JSON(code, response.OfSuccess(fd.MessageType))
	} else {
		code := http.StatusNotFound
		context.JSON(code, response.OfFailure("NotFound"))
	}

}

func (p *ProtoService) GetAllProto(context *gin.Context) {
	names := make([]string, 0, len(p.ProtoTypeMap))
	for k := range p.ProtoTypeMap {
		names = append(names, k)
	}
	log.Printf("configured protos are ->  %v", names)
	code := http.StatusOK
	context.JSON(code, response.OfSuccess(names))
}

func (p *ProtoService) RegisterProtoFile() error {

	// Now load that temporary file as a file descriptor set protobuf.
	protoFile, err := ioutil.ReadFile(p.Configuration.ProtoDescFilePath)
	if err != nil {
		return err
	}

	pbSet := new(descriptorpb.FileDescriptorSet)
	if err := proto.Unmarshal(protoFile, pbSet); err != nil {
		return err
	}

	// We know protoc was invoked with a multiple .proto file.
	for _, pb := range pbSet.GetFile() {
		protoName := getName(pb.GetPackage(), pb.GetName())
		log.Printf("loading .. file descriptor --> %v", protoName)
		p.ProtoTypeMap[protoName] = *pb
		// Initialize the file descriptor object.
		fd, err := protodesc.NewFile(pb, protoregistry.GlobalFiles)
		if err != nil {
			return err
		}

		// and finally register it.
		err = protoregistry.GlobalFiles.RegisterFile(fd)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *ProtoService) ToProtoMessage(messageType string, messageBytes []byte) (*dynamicpb.Message, error) {
	marshaller := jsonpb.Unmarshaler{}

	d, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(messageType))

	if err != nil {
		return nil, err
	}
	md, _ := d.(protoreflect.MessageDescriptor)

	m := dynamicpb.NewMessage(md)
	marshaller.Unmarshal(strings.NewReader(string(messageBytes)), m)

	return m, nil
}

func (p *ProtoService) ToProtoMessageNOT_IN_USE(messageName string, messageBytes []byte) (protoreflect.ProtoMessage, error) {

	_, ok := p.ProtoTypeMap[messageName]

	if !ok {
		return nil, errors.New("proto not registered")
	}

	/*
		m := dynamicpb.NewMessage(d.ProtoReflect().Descriptor())
		marshaller := jsonpb.Unmarshaler{}
		marshaller.Unmarshal(strings.NewReader(string(messageBytes)), m)
		proto.Unmarshal(messageBytes, m)
		return m, nil
	*/

	var messageDescriptor protoreflect.MessageDescriptor = nil
	protoregistry.GlobalFiles.RangeFiles(func(descriptor protoreflect.FileDescriptor) bool {
		md := descriptor.Messages().ByName(protoreflect.Name("Whitelist"))
		if md != nil {
			messageDescriptor = md
		}
		return true
	})
	//md := fileDesc.Messages().Get(0) //(protoreflect.MessageDescriptor)
	if messageDescriptor == nil {
		return nil, errors.New("proto not registered")
	}
	mt := dynamicpb.NewMessageType(messageDescriptor)
	pm := mt.New().Interface()
	proto.Unmarshal(messageBytes, pm)

	return pm, nil

}

func getName(pkg string, name string) string {
	result := strings.Replace(name, pkg, "", 1)
	return strings.Replace(result, "/", "", 1)
}
