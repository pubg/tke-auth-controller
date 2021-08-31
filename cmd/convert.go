package main

import (
	"example.com/tke-auth-controller/internal"
	"flag"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"log"
	"os"
)

var file = ""

func init() {
	flag.StringVar(&file, "file", "", "file contains valid configMap")
	flag.Parse()

	if file == "" {
		flag.PrintDefaults()
		os.Exit(0)
	}
}

func main() {
	// load text from given path
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatalln(err)
	}

	// https://stackoverflow.com/questions/47116811/client-go-parse-kubernetes-json-files-to-k8s-structures
	decoder := serializer.NewCodecFactory(scheme.Scheme).UniversalDecoder()
	object := &v1.ConfigMap{}
	err = runtime.DecodeInto(decoder, bytes, object)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("current configMap")
	marshalAndPrint(object)

	bindings, ok := object.Data["bindings"]
	if !ok {
		log.Fatalf("no bindings found in ConfigMap.\n")
	}

	tkeAuthCfg, err := internal.UnMarshalYAMLToTKEAuthConfig(bindings)

	log.Println("parsed tkeConfigMap")
	marshalAndPrint(tkeAuthCfg.GetBindings())

	return
}

func marshalAndPrint(data interface{}) {
	buf, err := yaml.Marshal(data)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("\n%s\n", string(buf))
}

