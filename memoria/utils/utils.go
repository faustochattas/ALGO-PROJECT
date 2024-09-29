package utils

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/sisoputnfrba/tp-golang/memoria/config"
	"github.com/sisoputnfrba/tp-golang/utils/estructuras"
)

type Pagina struct {
	Numero int
	Marco  int
	// Otros campos según sea necesario
}

type Info struct {
	Pid            int
	NumeroPagina   int
	Marco          int
	Desplazamiento int
	Cantidad_bytes int
	Valor          []int
}

type Reserva struct {
	Pid              int
	Cantidad_paginas int
}

var TablaDePaginasPorProceso = make(map[int][]*Pagina)

var InstruccionesProceso = make(map[int]string)

//Canales

// Escritura en memoria
var Escribir_memoria = make(chan Info, 1)
var Respuesta_escritura_memoria = make(chan int, 1)

// Lectura en memoria
var Leer_Memoria = make(chan Info, 1)
var Respuesta_lectura_memoria = make(chan estructuras.Respuesta_lectura_memoria, 1)

// Borrar en memoria
var Borrar_memoria = make(chan int, 1)
var Respuesta_borrar_memoria = make(chan int, 1)

// Reservar paginas
var Reservar_paginas_canal = make(chan Reserva, 1)
var Respuesta_reservar_paginas = make(chan estructuras.Respuesta_reservar_paginas, 1)

//Marco para TLB

var Marco_tlb_canal = make(chan Info, 1)
var Respuesta_marco_tlb = make(chan int, 1)

////////////////////////////////////////////////////////////////////////////////////////////////////

func InicializarMemoria(memoria []byte, marcos_libres []int) {
	for {
		select {

		case escritura := <-Escribir_memoria:

			for i := 0; i < escritura.Cantidad_bytes; i++ {

				if escritura.Marco == -1 {

					proceso := TablaDePaginasPorProceso[escritura.Pid]

					for _, pagina := range proceso {
						if pagina.Numero == escritura.NumeroPagina {
							marco := pagina.Marco
							escritura.Marco = marco
							break
						}
					}

				}

				posicion := escritura.Marco*config.Memoria.Page_size + escritura.Desplazamiento + i

				valor_array := escritura.Valor[i]

				memoria[posicion] = byte(valor_array)

			}

			//fmt.Println(memoria)

			Respuesta_escritura_memoria <- 1

		case a_leer := <-Leer_Memoria:

			var array []int

			for i := 0; i < a_leer.Cantidad_bytes; i++ {

				if a_leer.Marco == -1 {

					proceso := TablaDePaginasPorProceso[a_leer.Pid]

					for _, pagina := range proceso {
						if pagina.Numero == a_leer.NumeroPagina {
							marco := pagina.Marco
							a_leer.Marco = marco
							break
						}
					}
				}

				posicion := a_leer.Marco*config.Memoria.Page_size + a_leer.Desplazamiento + i

				valor := int(memoria[posicion])

				array = append(array, valor)

			}
			resp := estructuras.Respuesta_lectura_memoria{
				Valor: array,
			}

			Respuesta_lectura_memoria <- resp

		case a_borrar := <-Borrar_memoria:

			Ajustar_proceso(a_borrar, 0, marcos_libres)

			delete(InstruccionesProceso, a_borrar)

			Respuesta_borrar_memoria <- 1

		case reservar_paginas := <-Reservar_paginas_canal:

			confirmacion := Ajustar_proceso(reservar_paginas.Pid, reservar_paginas.Cantidad_paginas, marcos_libres)

			if confirmacion == "OUT_OF_MEMORY" {

				resp := estructuras.Respuesta_reservar_paginas{
					Estado: "OUT_OF_MEMORY",
				}
				Respuesta_reservar_paginas <- resp

			} else {

				resp := estructuras.Respuesta_reservar_paginas{
					Estado: "OK",
				}
				Respuesta_reservar_paginas <- resp

			}

		case marco_tlb := <-Marco_tlb_canal:

			proceso := TablaDePaginasPorProceso[marco_tlb.Pid]

			var resp int

			for _, pagina := range proceso {
				if pagina.Numero == marco_tlb.NumeroPagina {
					marco := pagina.Marco
					resp = marco
					break
				}
			}

			Respuesta_marco_tlb <- resp

		}
	}

}

func Crear_proceso(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	var proceso estructuras.Path_proceso
	err := decoder.Decode(&proceso)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar pcb"))
		return
	}

	InstruccionesProceso[proceso.Pid] = proceso.Path_proceso

	TablaDePaginasPorProceso[proceso.Pid] = make([]*Pagina, 0) //nuevo probar despues

	time.Sleep(time.Duration(config.Memoria.Delay_response) * time.Millisecond)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))

}

func Buscar_instruccion(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var instruccion estructuras.Pedir_instruccion_memoria
	err := decoder.Decode(&instruccion)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar pcb"))
		return
	}

	instruccionPath, err := Buscar_en_path(instruccion)

	log.Printf("PID: %d - Accion: <LEER INSTRUCCION> - Instruccion: %s\n", instruccion.Pid, instruccionPath)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al buscar instruccion"))
		return
	}

	resp := estructuras.Instruccion_memoria{
		Instruccion: instruccionPath,
	}

	respuesta, err := json.Marshal(resp)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	time.Sleep(time.Duration(config.Memoria.Delay_response) * time.Millisecond)
	//time.Sleep(5 * time.Second)

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)

}

func Buscar_en_path(instruccion estructuras.Pedir_instruccion_memoria) (string, error) {

	algo := config.Memoria.Instruction_path

	instructions_path := algo + "/" + InstruccionesProceso[instruccion.Pid]

	objetivo, err := ReadLineFromFile(instructions_path, instruccion.Program_counter)

	if err != nil {
		log.Println("Error al obtener instruccion")
		return "", err
	}

	return objetivo, nil

	//retornar_instruccion(objetivo)

}

func ReadLineFromFile(filePath string, lineNumber int) (string, error) {
	file, err := os.Open(filePath)

	if err != nil {
		return "", err
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	lineCount := -1

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err.Error() == "EOF" {
				return line, nil
			} else {
				return "", err
			}
		}
		lineCount++
		if lineCount == lineNumber {
			return line, nil
		}
	}

	return "", fmt.Errorf("No se encontró la línea %d en el archivo", lineNumber)
}

func Ajustar_proceso(pid int, cantidad_paginas int, marcos_libres []int) string {

	//Hacer chequeo en la memoria si hay espacio para mas paginas

	paginasActuales := len(TablaDePaginasPorProceso[pid])

	diferencia := cantidad_paginas - paginasActuales

	var libres int

	for i := 0; i < len(marcos_libres); i++ {
		if marcos_libres[i] == 0 {
			libres++
		}
	}

	if libres < diferencia {
		log.Println("OUT_OF_MEMORY")
		return "OUT_OF_MEMORY"
	}

	if diferencia > 0 {

		log.Printf("PID: %d - Tamaño actual: %d - Tamaño a Ampliar: %d \n", pid, paginasActuales, cantidad_paginas)

		for i := 0; i < diferencia; i++ {

			indice := -1

			for j := 0; j < len(marcos_libres); j++ {

				if marcos_libres[j] == 0 {
					indice = j
					break
				}

			}

			pagina := Pagina{
				Numero: paginasActuales + i,
				Marco:  indice,
			}

			TablaDePaginasPorProceso[pid] = append(TablaDePaginasPorProceso[pid], &pagina)

			marcos_libres[indice] = 1

			log.Println("Pagina reservada: ", pagina.Numero, " Marco: ", pagina.Marco)

			log.Printf("PID: %d - Tamaño: %d\n", pid, cantidad_paginas)

		}

	} else {
		//Borrar paginas
		diferencia = diferencia * -1

		log.Printf("PID: %d - Tamaño actual: %d - Tamaño a Reducir: %d \n", pid, paginasActuales, cantidad_paginas)

		for i := 0; i < diferencia; i++ {

			paginas := TablaDePaginasPorProceso[pid]

			indice := paginas[len(paginas)-1].Marco

			marcos_libres[indice] = 0

			TablaDePaginasPorProceso[pid] = TablaDePaginasPorProceso[pid][:len(TablaDePaginasPorProceso[pid])-1]
		}

		log.Printf("PID: %d - Tamaño: %d\n", pid, cantidad_paginas)
	}

	return "OK"

}

func Pedido_lectura(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	var lectura estructuras.Pedido_lectura_memoria
	err := decoder.Decode(&lectura)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar pcb"))
		return
	}

	//log.Println("LLEGO UN PEDIDO DE LECTURA")

	//log.Print("Pid: ", lectura.Pid)

	info := Info{
		Pid:            lectura.Pid,
		Marco:          lectura.Marco,
		Desplazamiento: lectura.Desplazamiento,
		NumeroPagina:   lectura.Pagina,
		Cantidad_bytes: lectura.Cantidad_bytes_leer,
	}

	direccion_fisica := lectura.Pagina*config.Memoria.Page_size + lectura.Desplazamiento

	log.Printf("“PID: %d - Accion: <LEER> - Direccion fisica: %d ” - Tamaño %d\n", lectura.Pid, direccion_fisica, lectura.Cantidad_bytes_leer)

	Leer_Memoria <- info

	resp := <-Respuesta_lectura_memoria

	respuesta, err := json.Marshal(resp)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	time.Sleep(time.Duration(config.Memoria.Delay_response) * time.Millisecond)

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)

}

func Pedido_escritura(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	var escritura estructuras.Pedido_escritura_memoria
	err := decoder.Decode(&escritura)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar pcb"))
		return
	}

	//log.Println("LLEGO UN PEDIDO DE ESCRITURA")

	//log.Print("Pid: ", escritura.Pid)

	info := Info{
		Pid:            escritura.Pid,
		Marco:          escritura.Marco,
		Desplazamiento: escritura.Desplazamiento,
		NumeroPagina:   escritura.Pagina,
		Cantidad_bytes: escritura.Cantidad_bytes_escribir,
		Valor:          escritura.Valor,
	}

	direccion_fisica := escritura.Pagina*config.Memoria.Page_size + escritura.Desplazamiento

	log.Printf("“PID: %d - Accion: <ESCRIBIR> - Direccion fisica: %d ” - Tamaño %d\n", escritura.Pid, direccion_fisica, escritura.Cantidad_bytes_escribir)

	Escribir_memoria <- info

	<-Respuesta_escritura_memoria

	time.Sleep(time.Duration(config.Memoria.Delay_response) * time.Millisecond)

	w.WriteHeader(http.StatusOK)

}

func Reservar_paginas(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	var reserva estructuras.Resize
	err := decoder.Decode(&reserva)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar pcb"))
		return
	}

	//log.Println("LLEGO UN PEDIDO DE RESIZE")

	//log.Print("Pid: ", reserva.Pid)

	reserva_info := Reserva{
		Pid:              reserva.Pid,
		Cantidad_paginas: reserva.Ajuste,
	}

	Reservar_paginas_canal <- reserva_info

	resp := <-Respuesta_reservar_paginas

	respuesta, err := json.Marshal(resp)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	time.Sleep(time.Duration(config.Memoria.Delay_response) * time.Millisecond)

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)

}

func Marco_tlb(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var marco estructuras.TLB_miss
	err := decoder.Decode(&marco)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar pcb"))
		return
	}

	//log.Println("Buscar marco por TLB MISS")

	//log.Print("Pid: ", marco.Pid)

	info := Info{
		Pid:          marco.Pid,
		NumeroPagina: marco.Numero_pagina,
	}

	Marco_tlb_canal <- info

	resp := <-Respuesta_marco_tlb

	respuesta, err := json.Marshal(resp)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	time.Sleep(time.Duration(config.Memoria.Delay_response) * time.Millisecond)

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

func Borrar_proceso_memoria(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	var escritura estructuras.Pedido_borrar_memoria
	err := decoder.Decode(&escritura)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar pcb"))
		return
	}

	//log.Println("LLEGO UN PEDIDO DE BORRADO DE MEMORIA")

	//log.Print("Pid: ", escritura.Pid)

	Borrar_memoria <- escritura.Pid

	<-Respuesta_borrar_memoria

	time.Sleep(time.Duration(config.Memoria.Delay_response) * time.Millisecond)

	w.WriteHeader(http.StatusOK)

}
