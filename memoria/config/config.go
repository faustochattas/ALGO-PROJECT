package config

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/sisoputnfrba/tp-golang/utils/cliente"
)

type Config_memoria struct {
	Port             int    `json:"port"`
	Memory_size      int    `json:"memory_size"`
	Page_size        int    `json:"page_size"`
	Instruction_path string `json:"instructions_path"`
	Delay_response   int    `json:"delay_response"`
}

var Memoria *Config_memoria

func IniciarConfiguracion(filePath string) *Config_memoria {
	var config *Config_memoria
	configFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)

	return config
}

func Dar_Tamaño_pagina(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje *cliente.Mensaje
	err := decoder.Decode(&mensaje)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar pcb"))
		return
	}

	tamaño_pagina := Tamaño_pagina{
		Tamaño: Memoria.Page_size,
	}

	respuesta, err := json.Marshal(tamaño_pagina)

	if err != nil {
		log.Println("Error al codificar respuesta")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)

}

type Tamaño_pagina struct {
	Tamaño int `json:"tamaño"`
}
