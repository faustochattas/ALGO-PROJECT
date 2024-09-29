package config

import (
	"encoding/json"
	"log"
	"os"
)

type Config_io struct {
	Ip                   string `json:"ip"`
	Port                 int    `json:"port"`
	Type                 string `json:"type"`
	Unit_work_time       int    `json:"unit_work_time"`
	Ip_kernel            string `json:"ip_kernel"`
	Port_kernel          int    `json:"port_kernel"`
	Ip_memory            string `json:"ip_memory"`
	Port_memory          int    `json:"port_memory"`
	Dialfs_path          string `json:"dialfs_path"`
	Dialfs_block_size    int    `json:"dialfs_block_size"`
	Dialfs_block_count   int    `json:"dialfs_block_count"`
	Retraso_compactacion int    `json:"RETRASO_COMPACTACION"`
}

var Io *Config_io

func IniciarConfiguracion(filePath string) *Config_io {
	var config *Config_io
	configFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)

	return config
}
