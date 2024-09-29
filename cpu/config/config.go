package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/sisoputnfrba/tp-golang/utils/cliente"
)

type Config_cpu struct {
	Port               int    `json:"port"`
	Ip_memory          string `json:"ip_memory"`
	Port_memory        int    `json:"port_memory"`
	Number_felling_tlb int    `json:"number_felling_tlb"`
	Algorithm_tlb      string `json:"algorithm_tlb"`
}

var Cpu *Config_cpu

// Estructura para representar una entrada en la TLB
type TLBEntry struct {
	PID    int
	Pagina int
	Marco  int
}

func IniciarConfiguracion(filePath string) *Config_cpu {
	var config *Config_cpu
	configFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)

	return config
}

func Pedir_Tamaño_Pagina() *Tamaño_pagina {
	pedido := cliente.Mensaje{
		Mensaje: "Quiero el tamaño de pagina",
	}

	body, err := json.Marshal(pedido)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/tamaño_pagina", Cpu.Ip_memory, Cpu.Port_memory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", Cpu.Ip_memory, Cpu.Port_memory)
	}

	defer resp.Body.Close()

	var tamaño_pagina *Tamaño_pagina

	json.NewDecoder(resp.Body).Decode(&tamaño_pagina)

	log.Println("Tamaño de paginas:", tamaño_pagina.Tamaño)

	return tamaño_pagina
}

type Tamaño_pagina struct {
	Tamaño int
}

var Tamaño_pagina_memoria *Tamaño_pagina
