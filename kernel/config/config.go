package config

import (
	"encoding/json"
	"log"
	"os"
)

type Config_kernel struct {
	Port                int      `json:"port"`
	Ip_memory           string   `json:"ip_memory"`
	Port_memory         int      `json:"port_memory"`
	Ip_cpu              string   `json:"ip_cpu"`
	Port_cpu            int      `json:"port_cpu"`
	Planning_algorithm  string   `json:"planning_algorithm"`
	Quantum             int      `json:"quantum"`
	Resources           []string `json:"resources"`
	Resources_instances []int    `json:"resource_instances"`
	Multiprogramming    int      `json:"multiprogramming"`
}

var Kernel *Config_kernel

func IniciarConfiguracion(filePath string) *Config_kernel {
	var config *Config_kernel
	configFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)

	return config
}
