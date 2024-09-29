package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/sisoputnfrba/tp-golang/entradasalida/FS"
	"github.com/sisoputnfrba/tp-golang/entradasalida/config"
	"github.com/sisoputnfrba/tp-golang/entradasalida/utils"

	"github.com/sisoputnfrba/tp-golang/utils/cliente"
	"github.com/sisoputnfrba/tp-golang/utils/estructuras"
	"github.com/sisoputnfrba/tp-golang/utils/servidor"
)

func main() {

	//fmt.Print(os.Args)

	directorioActual, erro := os.Getwd()
	if erro != nil {
		fmt.Println("Error al obtener el directorio actual:", erro)
		return
	}

	cliente.ConfigurarLogger("entradasalida", directorioActual)

	nombreIO := os.Args[1]
	pathIo := os.Args[2]

	//log.Printf("Path: %s\n ", pathIo)
	//log.Printf("Nombre IO: %s\n: ", nombreIO)

	config.Io = config.IniciarConfiguracion(pathIo) // "/Users/adriangil/tp-2024-1c-Los-Pre-Alfa/entradasalida/config.json"

	if config.Io.Type == "DialFS" {
		FS.Crear_bitmap(config.Io.Dialfs_path, config.Io.Dialfs_block_count)
		FS.Crear_archivo_bloques(config.Io.Dialfs_path, config.Io.Dialfs_block_size, config.Io.Dialfs_block_count)
	}

	ConectarseConKernel(nombreIO)

	mux := http.NewServeMux()
	mux.HandleFunc("/mensaje", servidor.RecibirMensaje)
	mux.HandleFunc("/kernel", PeticionKernel)

	mux.HandleFunc("/ejecutar_interfaz", utils.Ejecutar_interfaz)

	puerto := strconv.Itoa(config.Io.Port)
	puerto = ":" + puerto

	//panic("no implementado!")
	err := http.ListenAndServe(puerto, mux)
	if err != nil {
		panic(err)
	}

}

func ConectarseConKernel(nombreIO string) {

	interfaz := estructuras.Interfaz{
		Nombre: nombreIO, Tipo: config.Io.Type, UnidadesTrabajo: config.Io.Unit_work_time,
		Ip: config.Io.Ip, Port: config.Io.Port,
	}

	body, err := json.Marshal(interfaz)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/interfaz", config.Io.Ip_kernel, config.Io.Port_kernel)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", config.Io.Ip_kernel, config.Io.Port_kernel)
	}

	log.Printf("respuesta del servidor: %s", resp.Status)

}

func PeticionKernel(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	var mensaje cliente.Mensaje
	err := decoder.Decode(&mensaje)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
