package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/sisoputnfrba/tp-golang/memoria/config"
	"github.com/sisoputnfrba/tp-golang/memoria/utils"
	"github.com/sisoputnfrba/tp-golang/utils/cliente"
)

func main() {

	directorioActual, erro := os.Getwd()
	if erro != nil {
		fmt.Println("Error al obtener el directorio actual:", erro)
		return
	}

	cliente.ConfigurarLogger("memoria", directorioActual)

	path_json := directorioActual + "/config.json"

	config.Memoria = config.IniciarConfiguracion(path_json)

	log.Printf("Configuracion cargada: %+v\n", config.Memoria)
	// validar que la config este cargada correctamente

	if config.Memoria == nil {
		log.Println("Error al cargar la configuración")
		return
	}

	memoria := make([]byte, config.Memoria.Memory_size)

	marcos_libres := make([]int, config.Memoria.Memory_size/config.Memoria.Page_size)

	/*
		for i := 0; i < config.Memoria.Memory_size; i++ {
			memoria[i] = byte(i)
		}
	*/

	//fmt.Println(memoria)

	go utils.InicializarMemoria(memoria, marcos_libres)

	mux := http.NewServeMux()

	//Retornar el tamaño de pagina
	mux.HandleFunc("/tamaño_pagina", config.Dar_Tamaño_pagina)

	//Instrucciones
	mux.HandleFunc("/pedido_instruccion", utils.Buscar_instruccion)
	mux.HandleFunc("/crear_proceso", utils.Crear_proceso)

	//Pedido de la CPU
	mux.HandleFunc("/pedido_lectura", utils.Pedido_lectura)
	mux.HandleFunc("/pedido_escritura", utils.Pedido_escritura)
	mux.HandleFunc("/resize", utils.Reservar_paginas)
	mux.HandleFunc("/marco_tlb", utils.Marco_tlb)
	mux.HandleFunc("/borrar_proceso", utils.Borrar_proceso_memoria)

	puerto := strconv.Itoa(config.Memoria.Port)
	puerto = ":" + puerto

	log.Print("Escuchando en puerto: ", puerto)

	//panic("no implementado!")
	err := http.ListenAndServe(puerto, mux)
	if err != nil {
		panic(err)
	}

}
