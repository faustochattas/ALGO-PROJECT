package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/sisoputnfrba/tp-golang/cpu/config"
	"github.com/sisoputnfrba/tp-golang/cpu/utils"
	"github.com/sisoputnfrba/tp-golang/utils/cliente"
	"github.com/sisoputnfrba/tp-golang/utils/servidor"
)

func main() {

	directorioActual, erro := os.Getwd()
	if erro != nil {
		fmt.Println("Error al obtener el directorio actual:", erro)
		return
	}

	cliente.ConfigurarLogger("cpu", directorioActual)
	log.Println("Soy un log")

	path_json := directorioActual + "/config.json"

	config.Cpu = config.IniciarConfiguracion(path_json) //path hardcodeado
	// validar que la config este cargada correctamente

	config.Tamaño_pagina_memoria = config.Pedir_Tamaño_Pagina()

	if config.Cpu == nil {
		log.Println("Error al cargar la configuración")
		return
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/mensaje", servidor.RecibirMensaje)
	mux.HandleFunc("/interrumpir_proceso", utils.Interrupir_Proceso)
	mux.HandleFunc("/contexto", utils.Recibir_contexto)

	puerto := strconv.Itoa(config.Cpu.Port)
	puerto = ":" + puerto

	log.Print("Escuchando en puerto: ", puerto)

	//panic("no implementado!")
	err := http.ListenAndServe(puerto, mux)
	if err != nil {
		panic(err)
	}

}
