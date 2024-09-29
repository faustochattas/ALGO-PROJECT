package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/sisoputnfrba/tp-golang/cpu/config"
	"github.com/sisoputnfrba/tp-golang/utils/estructuras"
	"github.com/sisoputnfrba/tp-golang/utils/servidor"
)

var TLB []*config.TLBEntry = make([]*config.TLBEntry, 0)

var interruptor_quantum = 0

var finalizar_proceso = 0

func Interrupir_Proceso(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var motivo servidor.Mensaje

	err := decoder.Decode(&motivo)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar motivo"))
		return
	}
	log.Println("INTERRUPCION DE PROCESO")
	log.Println("Motivo: ", motivo.Mensaje)

	switch motivo.Mensaje {
	case "QUANTUM":
		interruptor_quantum = 1
	case "FINALIZAR_PROCESO":
		finalizar_proceso = 1
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))

}

// Recibe el pcb del kernel
func Recibir_contexto(w http.ResponseWriter, r *http.Request) {
	interruptor_quantum = 0
	finalizar_proceso = 0
	decoder := json.NewDecoder(r.Body)
	var contexto *estructuras.Pcb_contexto
	err := decoder.Decode(&contexto)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar pcb"))
		return
	}
	//log.Println("LLEGO UN PROCESO DEL KERNEL")

	pcb, err := Ejecutar_proceso(contexto)
	if err != nil {
		log.Println("Error al ejecutar proceso")
		return
	}

	respuesta, err := json.Marshal(pcb)

	if err != nil {
		log.Println("Error al codificar respuesta")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)

}

func Pedir_instruccion_memoria(pid_proceso int, pc_proceso int) (string, error) {

	pedido := estructuras.Pedir_instruccion_memoria{
		Pid:             pid_proceso,
		Program_counter: pc_proceso,
	}

	body, err := json.Marshal(pedido)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/pedido_instruccion", config.Cpu.Ip_memory, config.Cpu.Port_memory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", config.Cpu.Ip_memory, config.Cpu.Port_memory)
	}

	var instruccionEjecutar estructuras.Instruccion_memoria

	json.NewDecoder(resp.Body).Decode(&instruccionEjecutar)

	//log.Println("Instruccion :", instruccionEjecutar.Instruccion)

	return instruccionEjecutar.Instruccion, nil

}

func Fetch(pcb *estructuras.Pcb_contexto) (string, error) {
	//log.Println("Fetch")

	pid_proceso := pcb.Pid
	pc_proceso := pcb.Program_counter

	fetch, err := Pedir_instruccion_memoria(pid_proceso, pc_proceso)
	if err != nil {
		log.Println("Error al pedir instruccion a memoria")
		return "", err
	}

	return fetch, nil

}

func Ejecutar_proceso(pcb *estructuras.Pcb_contexto) (*estructuras.Pcb_contexto, error) {
	for {

		if interruptor_quantum != 0 {
			pcb.Bloqueo.Motivo = "FIN_QUANTUM"
			return pcb, nil
		}

		if finalizar_proceso != 0 {
			pcb.Bloqueo.Motivo = "INTERRUPTED_BY_USER"
			return pcb, nil
		}

		instruccion, err := Fetch(pcb)

		log.Printf("PID: %d - FETCH - Program Counter : %d", pcb.Pid, pcb.Program_counter) //LOG OBLIGATORIO

		if err != nil {
			log.Println("Error al obtener instruccion")
			return nil, err
		}

		if instruccion == "EXIT" {
			break
		}

		palabras := strings.Split(instruccion, " ")

		switch palabras[0] {
		case "SET":
			pcb.Program_counter++

			registro := palabras[1]

			valor := strings.TrimRight(palabras[2], "\n")

			log.Printf("PID: %d - Ejecutando: SET - %s %s", pcb.Pid, registro, valor) //LOG OBLIGATORIO

			//pasar a entero valor
			valorInt, err := strconv.Atoi(valor)
			if err != nil {
				fmt.Println("Error al convertir el string a entero:", err)
				return nil, err
			}

			prefijo := registro[:1]

			if registro == "PC" {
				pcb.Program_counter = valorInt
				break
			}

			if prefijo == "E" || registro == "SI" || registro == "DI" {
				valor32 := uint32(valorInt)
				Obtener_setear_valor_registro32(pcb, registro, uint32(valor32))
				fmt.Print("Registro EAX: " + strconv.Itoa(int(pcb.EAX)))

			} else {
				valor8 := uint8(valorInt)
				Obtener_setear_valor_registro8(pcb, registro, valor8)
			}

		case "SUM":
			pcb.Program_counter++
			primerRegistro := palabras[1]
			segundoRegistro := strings.TrimRight(palabras[2], "\n")

			log.Printf("PID: %d - Ejecutando: SUM - %s %s", pcb.Pid, primerRegistro, segundoRegistro) //LOG OBLIGATORIO

			prefijo_primer := primerRegistro[:1]

			prefijo_segundo := segundoRegistro[:1]

			if prefijo_primer == "E" || segundoRegistro == "SI" || segundoRegistro == "DI" {

				if prefijo_segundo == "E" || segundoRegistro == "SI" || segundoRegistro == "DI" {

					valor1, err := Obtener_valor_registro32(pcb, segundoRegistro)
					if err != nil {
						log.Println("Error al obtener valor de registro")
						return nil, err
					}
					Sumar_valor_registro32(pcb, primerRegistro, valor1)
				} else {
					valor1, err := Obtener_valor_registro8(pcb, segundoRegistro)
					if err != nil {
						log.Println("Error al obtener valor de registro")
						return nil, err
					}
					Sumar_valor_registro32(pcb, primerRegistro, uint32(valor1))
				}

				//log.Print("Registro EAX: " + strconv.Itoa(int(pcb.EAX)))

			} else {

				if prefijo_segundo == "E" || segundoRegistro == "SI" || segundoRegistro == "DI" {

					valor1, err := Obtener_valor_registro32(pcb, segundoRegistro)
					if err != nil {
						log.Println("Error al obtener valor de registro")
						return nil, err
					}
					Sumar_valor_registro8(pcb, primerRegistro, uint8(valor1))

				} else {

					valor1, err := Obtener_valor_registro8(pcb, segundoRegistro)
					if err != nil {
						log.Println("Error al obtener valor de registro")
						return nil, err
					}
					Sumar_valor_registro8(pcb, primerRegistro, valor1)

				}
			}

		case "SUB":
			pcb.Program_counter++
			primerRegistro := palabras[1]
			segundoRegistro := strings.TrimRight(palabras[2], "\n")

			log.Printf("PID: %d - Ejecutando: SUB - %s %s", pcb.Pid, primerRegistro, segundoRegistro) //LOG OBLIGATORIO

			prefijo_primer := primerRegistro[:1]

			prefijo_segundo := segundoRegistro[:1]

			if prefijo_primer == "E" || segundoRegistro == "SI" || segundoRegistro == "DI" {

				if prefijo_segundo == "E" || segundoRegistro == "SI" || segundoRegistro == "DI" {

					valor1, err := Obtener_valor_registro32(pcb, segundoRegistro)
					if err != nil {
						log.Println("Error al obtener valor de registro")
						return nil, err
					}
					Restar_valor_registro32(pcb, primerRegistro, valor1)
				} else {
					valor1, err := Obtener_valor_registro8(pcb, segundoRegistro)
					if err != nil {
						log.Println("Error al obtener valor de registro")
						return nil, err
					}
					Restar_valor_registro32(pcb, primerRegistro, uint32(valor1))
				}

				//log.Print("Registro EAX: " + strconv.Itoa(int(pcb.EAX)))

			} else {

				if prefijo_segundo == "E" || segundoRegistro == "SI" || segundoRegistro == "DI" {

					valor1, err := Obtener_valor_registro32(pcb, segundoRegistro)
					if err != nil {
						log.Println("Error al obtener valor de registro")
						return nil, err
					}
					Restar_valor_registro8(pcb, primerRegistro, uint8(valor1))

				} else {

					valor1, err := Obtener_valor_registro8(pcb, segundoRegistro)
					if err != nil {
						log.Println("Error al obtener valor de registro")
						return nil, err
					}
					Restar_valor_registro8(pcb, primerRegistro, valor1)

				}
			}

		case "JNZ":

			pcb.Program_counter++
			registro := palabras[1]
			pcInstruccion := strings.TrimRight(palabras[2], "\n")

			log.Printf("PID: %d - Ejecutando: JNZ - %s %s", pcb.Pid, registro, pcInstruccion) //LOG OBLIGATORIO

			otroPC, err := strconv.Atoi(pcInstruccion)
			if err != nil {
				fmt.Println("Error al convertir el string a entero:", err)
				return nil, err
			}

			verificador, err := Verificar_registro_distinto_cero(pcb, registro)
			if err != nil {
				log.Println("Error al verificar registro")
				return nil, err
			}

			if verificador {
				pcb.Program_counter = otroPC
				log.Println("Instruccion a ejecutar: ", pcInstruccion)
			} else {
				log.Println("No se cumple la condicion")
			}

		case "IO_GEN_SLEEP":

			nombreIO := palabras[1]

			unidadesTrabajo := strings.TrimRight(palabras[2], "\n")

			log.Printf("PID: %d - Ejecutando: IO_GEN_SLEEP - %s %s", pcb.Pid, nombreIO, unidadesTrabajo) //LOG OBLIGATORIO

			pcb.Bloqueo.Motivo = "IO_GEN_SLEEP"
			pcb.Bloqueo.Parametro1 = nombreIO
			pcb.Bloqueo.Parametro2 = unidadesTrabajo

			pcb.Program_counter++

			return pcb, nil

		case "IO_STDIN_READ":

			// Esta instrucción solicita al Kernel que mediante la interfaz ingresada se lea desde el STDIN (Teclado) un valor cuyo
			//tamaño está delimitado por el valor del Registro Tamaño y el mismo se guarde a partir de la Dirección Lógica almacenada en el Registro Dirección.
			nombreIO := palabras[1]
			registro_direccion := palabras[2]
			registro_tamanio := strings.TrimRight(palabras[3], "\n")
			log.Printf("PID: %d - Ejecutando: IO_STDIN_READ - %s %s %s", pcb.Pid, nombreIO, registro_direccion, registro_tamanio)

			prefijo := registro_tamanio[:1]

			prefijo_direccion := registro_direccion[:1]

			var cantidad_bytes_leer int

			if prefijo == "E" || registro_tamanio == "SI" || registro_tamanio == "DI" {

				valor1, err := Obtener_valor_registro32(pcb, registro_tamanio)
				if err != nil {
					log.Println("Error al obtener valor de registro")
					return nil, err
				}
				cantidad_bytes_leer = int(valor1)

			} else {
				valor1, err := Obtener_valor_registro8(pcb, registro_tamanio)
				if err != nil {
					log.Println("Error al obtener valor de registro")
					return nil, err
				}
				cantidad_bytes_leer = int(valor1)
			}

			//log.Println("Cantidad de bytes a leer: ", cantidad_bytes_leer)

			var valorDireccionLogica int

			if prefijo_direccion == "E" || registro_direccion == "SI" || registro_direccion == "DI" {

				valor1, err := Obtener_valor_registro32(pcb, registro_direccion)
				if err != nil {
					log.Println("Error al obtener valor de registro")
					return nil, err
				}
				valorDireccionLogica = int(valor1)

			} else {
				valor1, err := Obtener_valor_registro8(pcb, registro_direccion)
				if err != nil {
					log.Println("Error al obtener valor de registro")
					return nil, err
				}
				valorDireccionLogica = int(valor1)
			}

			var direcciones_fisicas []*estructuras.Direccion_fisica

			for {

				if cantidad_bytes_leer == 0 {
					break
				}
				pagina, dezplazamiento, _ := Mmu(pcb, valorDireccionLogica)

				limite_bytes := config.Tamaño_pagina_memoria.Tamaño - dezplazamiento

				if limite_bytes > cantidad_bytes_leer {
					limite_bytes = cantidad_bytes_leer
				}

				direccion_fisica := &estructuras.Direccion_fisica{
					Pagina:         pagina,
					Desplazamiento: dezplazamiento,
					Cantidad_bytes: limite_bytes,
				}

				direcciones_fisicas = append(direcciones_fisicas, direccion_fisica)

				cantidad_bytes_leer -= limite_bytes

				valorDireccionLogica += limite_bytes

			}

			pcb.Bloqueo.Motivo = "IO_STDIN_READ"
			pcb.Bloqueo.Parametro1 = nombreIO

			//pcb.Bloqueo.Parametro2 = strconv.Itoa(cantidad_bytes_leer)
			pcb.Bloqueo.Direcciones_Fisicas = direcciones_fisicas

			pcb.Program_counter++

			return pcb, nil

		case "IO_STDOUT_WRITE":

			nombreIO := palabras[1]
			registro_direccion := palabras[2]
			registro_tamanio := strings.TrimRight(palabras[3], "\n")
			log.Printf("PID: %d - Ejecutando: IO_STDIN_WRITE- %s %s %s", pcb.Pid, nombreIO, registro_direccion, registro_tamanio)

			prefijo := registro_tamanio[:1]

			prefijo_direccion := registro_direccion[:1]

			var cantidad_bytes_escribir int

			if prefijo == "E" || registro_tamanio == "SI" || registro_tamanio == "DI" {

				valor1, err := Obtener_valor_registro32(pcb, registro_tamanio)
				if err != nil {
					log.Println("Error al obtener valor de registro")
					return nil, err
				}
				cantidad_bytes_escribir = int(valor1)

			} else {
				valor1, err := Obtener_valor_registro8(pcb, registro_tamanio)
				if err != nil {
					log.Println("Error al obtener valor de registro")
					return nil, err
				}
				cantidad_bytes_escribir = int(valor1)
			}

			var valorDireccionLogica int

			if prefijo_direccion == "E" || registro_direccion == "SI" || registro_direccion == "DI" {

				valor1, err := Obtener_valor_registro32(pcb, registro_direccion)
				if err != nil {
					log.Println("Error al obtener valor de registro")
					return nil, err
				}
				valorDireccionLogica = int(valor1)

			} else {
				valor1, err := Obtener_valor_registro8(pcb, registro_direccion)
				if err != nil {
					log.Println("Error al obtener valor de registro")
					return nil, err
				}
				valorDireccionLogica = int(valor1)
			}

			var direcciones_fisicas []*estructuras.Direccion_fisica

			for {

				if cantidad_bytes_escribir == 0 {
					break
				}
				pagina, dezplazamiento, _ := Mmu(pcb, valorDireccionLogica)

				limite_bytes := config.Tamaño_pagina_memoria.Tamaño - dezplazamiento

				if limite_bytes > cantidad_bytes_escribir {
					limite_bytes = cantidad_bytes_escribir
				}

				direccion_fisica := &estructuras.Direccion_fisica{
					Pagina:         pagina,
					Desplazamiento: dezplazamiento,
					Cantidad_bytes: limite_bytes,
				}

				direcciones_fisicas = append(direcciones_fisicas, direccion_fisica)

				cantidad_bytes_escribir -= limite_bytes

				valorDireccionLogica += limite_bytes

			}

			pcb.Bloqueo.Motivo = "IO_STDOUT_WRITE"
			pcb.Bloqueo.Parametro1 = nombreIO
			//pcb.Bloqueo.Parametro2 = strconv.Itoa(cantidad_bytes_escribir)
			pcb.Bloqueo.Direcciones_Fisicas = direcciones_fisicas

			pcb.Program_counter++

			return pcb, nil

		case "IO_FS_CREATE":

			nombreIO := palabras[1]
			nombreArchivo := strings.TrimRight(palabras[2], "\n")

			log.Printf("PID: %d - Ejecutando: IO_FS_CREATE - %s %s", pcb.Pid, nombreIO, nombreArchivo) //LOG OBLIGATORIO

			pcb.Bloqueo.Motivo = "IO_FS_CREATE"
			pcb.Bloqueo.Parametro1 = nombreIO
			pcb.Bloqueo.Parametro2 = nombreArchivo

			pcb.Program_counter++

			return pcb, nil

		case "IO_FS_TRUNCATE":
			nombreIO := palabras[1]
			nombreArchivo := palabras[2]
			registro_tamanio := strings.TrimRight(palabras[3], "\n")

			log.Printf("PID: %d - Ejecutando: IO_FS_TRUNCATE - %s %s", pcb.Pid, nombreIO, nombreArchivo) //LOG OBLIGATORIO

			prefijo := registro_tamanio[:1]

			var cantidad_bytes int

			if prefijo == "E" || registro_tamanio == "SI" || registro_tamanio == "DI" {

				valor1, err := Obtener_valor_registro32(pcb, registro_tamanio)
				if err != nil {
					log.Println("Error al obtener valor de registro")
					return nil, err
				}
				cantidad_bytes = int(valor1)

			} else {
				valor1, err := Obtener_valor_registro8(pcb, registro_tamanio)
				if err != nil {
					log.Println("Error al obtener valor de registro")
					return nil, err
				}
				cantidad_bytes = int(valor1)
			}

			bytes := strconv.Itoa(cantidad_bytes)

			pcb.Bloqueo.Motivo = "IO_FS_TRUNCATE"
			pcb.Bloqueo.Parametro1 = nombreIO
			pcb.Bloqueo.Parametro2 = nombreArchivo
			pcb.Bloqueo.Parametro3 = bytes

			pcb.Program_counter++

			return pcb, nil

		case "IO_FS_DELETE":

			nombreIO := palabras[1]
			nombreArchivo := strings.TrimRight(palabras[2], "\n")

			log.Printf("PID: %d - Ejecutando: IO_FS_DELETE - %s %s", pcb.Pid, nombreIO, nombreArchivo) //LOG OBLIGATORIO

			pcb.Bloqueo.Motivo = "IO_FS_DELETE"
			pcb.Bloqueo.Parametro1 = nombreIO
			pcb.Bloqueo.Parametro2 = nombreArchivo

			pcb.Program_counter++

			return pcb, nil

		case "IO_FS_WRITE":

			nombreIO := palabras[1]
			nombreArchivo := palabras[2]
			registro_direccion := palabras[3]
			registro_tamanio := palabras[4]
			punteroArchivo := strings.TrimRight(palabras[5], "\n")

			log.Printf("PID: %d - Ejecutando: IO_FS_WRITE - %s %s %s %s %s", pcb.Pid, nombreIO, nombreArchivo, registro_direccion, registro_tamanio, punteroArchivo)

			prefijo := registro_tamanio[:1]

			prefijo_direccion := registro_direccion[:1]

			prefijo_puntero := punteroArchivo[:1]

			var cantidad_bytes_leer int

			if prefijo == "E" || registro_tamanio == "SI" || registro_tamanio == "DI" {

				valor1, err := Obtener_valor_registro32(pcb, registro_tamanio)
				if err != nil {
					log.Println("Error al obtener valor de registro1")
					return nil, err
				}
				cantidad_bytes_leer = int(valor1)

			} else {
				valor1, err := Obtener_valor_registro8(pcb, registro_tamanio)
				if err != nil {
					log.Println("Error al obtener valor de registro2")
					return nil, err
				}
				cantidad_bytes_leer = int(valor1)
			}

			var valorDireccionLogica int

			if prefijo_direccion == "E" || registro_direccion == "SI" || registro_direccion == "DI" {

				valor1, err := Obtener_valor_registro32(pcb, registro_direccion)
				if err != nil {
					log.Println("Error al obtener valor de registro3")
					return nil, err
				}
				valorDireccionLogica = int(valor1)

			} else {
				valor1, err := Obtener_valor_registro8(pcb, registro_direccion)
				if err != nil {
					log.Println("Error al obtener valor de registro4")
					return nil, err
				}
				valorDireccionLogica = int(valor1)
			}

			var valor_puntero int

			if prefijo_puntero == "E" || punteroArchivo == "SI" || punteroArchivo == "DI" {

				valor1, err := Obtener_valor_registro32(pcb, punteroArchivo)
				if err != nil {
					log.Println("Error al obtener valor de registro5")
					return nil, err
				}
				valor_puntero = int(valor1)

			} else {
				valor1, err := Obtener_valor_registro8(pcb, punteroArchivo)
				if err != nil {
					log.Println("Error al obtener valor de registro6")
					return nil, err
				}
				valor_puntero = int(valor1)
			}

			var direcciones_fisicas []*estructuras.Direccion_fisica

			for {

				if cantidad_bytes_leer == 0 {
					break
				}
				pagina, dezplazamiento, _ := Mmu(pcb, valorDireccionLogica)

				limite_bytes := config.Tamaño_pagina_memoria.Tamaño - dezplazamiento

				if limite_bytes > cantidad_bytes_leer {
					limite_bytes = cantidad_bytes_leer
				}

				direccion_fisica := &estructuras.Direccion_fisica{
					Pagina:         pagina,
					Desplazamiento: dezplazamiento,
					Cantidad_bytes: limite_bytes,
				}

				direcciones_fisicas = append(direcciones_fisicas, direccion_fisica)

				cantidad_bytes_leer -= limite_bytes

				valorDireccionLogica += limite_bytes

			}

			string_puntero := strconv.Itoa(valor_puntero)

			pcb.Bloqueo.Motivo = "IO_FS_WRITE"
			pcb.Bloqueo.Parametro1 = nombreIO
			pcb.Bloqueo.Parametro2 = nombreArchivo
			pcb.Bloqueo.Parametro3 = string_puntero
			pcb.Bloqueo.Direcciones_Fisicas = direcciones_fisicas

			pcb.Program_counter++

			return pcb, nil

		case "IO_FS_READ":

			nombreIO := palabras[1]
			nombreArchivo := palabras[2]
			registro_direccion := palabras[3]
			registro_tamanio := palabras[4]
			punteroArchivo := strings.TrimRight(palabras[5], "\n")

			log.Printf("PID: %d - Ejecutando: IO_FS_WRITE - %s %s %s %s, %s", pcb.Pid, nombreIO, nombreArchivo, registro_direccion, registro_tamanio, punteroArchivo)

			prefijo := registro_tamanio[:1]

			prefijo_direccion := registro_direccion[:1]

			prefijo_puntero := punteroArchivo[:1]

			var cantidad_bytes_leer int

			if prefijo == "E" || registro_tamanio == "SI" || registro_tamanio == "DI" {

				valor1, err := Obtener_valor_registro32(pcb, registro_tamanio)
				if err != nil {
					log.Println("Error al obtener valor de registro1")
					return nil, err
				}
				cantidad_bytes_leer = int(valor1)

			} else {
				valor1, err := Obtener_valor_registro8(pcb, registro_tamanio)
				if err != nil {
					log.Println("Error al obtener valor de registro2")
					return nil, err
				}
				cantidad_bytes_leer = int(valor1)
			}

			var valorDireccionLogica int

			if prefijo_direccion == "E" || registro_direccion == "SI" || registro_direccion == "DI" {

				valor1, err := Obtener_valor_registro32(pcb, registro_direccion)
				if err != nil {
					log.Println("Error al obtener valor de registro3")
					return nil, err
				}
				valorDireccionLogica = int(valor1)

			} else {
				valor1, err := Obtener_valor_registro8(pcb, registro_direccion)
				if err != nil {
					log.Println("Error al obtener valor de registro4")
					return nil, err
				}
				valorDireccionLogica = int(valor1)
			}

			var valor_puntero int

			if prefijo_puntero == "E" || punteroArchivo == "SI" || punteroArchivo == "DI" {

				valor1, err := Obtener_valor_registro32(pcb, punteroArchivo)
				if err != nil {
					log.Println("Error al obtener valor de registro5")
					return nil, err
				}
				valor_puntero = int(valor1)

			} else {
				valor1, err := Obtener_valor_registro8(pcb, punteroArchivo)
				if err != nil {
					log.Println("Error al obtener valor de registro6")
					return nil, err
				}
				valor_puntero = int(valor1)
			}

			var direcciones_fisicas []*estructuras.Direccion_fisica

			for {

				if cantidad_bytes_leer == 0 {
					break
				}
				pagina, dezplazamiento, _ := Mmu(pcb, valorDireccionLogica)

				limite_bytes := config.Tamaño_pagina_memoria.Tamaño - dezplazamiento

				if limite_bytes > cantidad_bytes_leer {
					limite_bytes = cantidad_bytes_leer
				}

				direccion_fisica := &estructuras.Direccion_fisica{
					Pagina:         pagina,
					Desplazamiento: dezplazamiento,
					Cantidad_bytes: limite_bytes,
				}

				direcciones_fisicas = append(direcciones_fisicas, direccion_fisica)

				cantidad_bytes_leer -= limite_bytes

				valorDireccionLogica += limite_bytes

			}

			string_puntero := strconv.Itoa(valor_puntero)

			pcb.Bloqueo.Motivo = "IO_FS_READ"
			pcb.Bloqueo.Parametro1 = nombreIO
			pcb.Bloqueo.Parametro2 = nombreArchivo
			pcb.Bloqueo.Parametro3 = string_puntero
			pcb.Bloqueo.Direcciones_Fisicas = direcciones_fisicas

			pcb.Program_counter++

			return pcb, nil

		case "WAIT":

			recurso_wait := strings.TrimRight(palabras[1], "\n")

			log.Printf("PID: %d - Ejecutando: WAIT - %s", pcb.Pid, recurso_wait) //LOG OBLIGATORIO

			pcb.Bloqueo.Motivo = "WAIT"
			pcb.Bloqueo.Parametro1 = recurso_wait

			pcb.Program_counter++

			return pcb, nil

		case "SIGNAL":

			recurso := strings.TrimRight(palabras[1], "\n")

			log.Printf("PID: %d - Ejecutando: SIGNAL - %s", pcb.Pid, recurso) //LOG OBLIGATORIO

			pcb.Bloqueo.Motivo = "SIGNAL"
			pcb.Bloqueo.Parametro1 = recurso

			pcb.Program_counter++

			return pcb, nil

		case "RESIZE":

			pcb.Program_counter++

			aux := strings.TrimRight(palabras[1], "\n")

			tamanio, err := strconv.Atoi(aux)

			if err != nil {
				fmt.Println("Error al convertir el string a entero:", err)
				return nil, err
			}

			log.Printf("PID: %d - Ejecutando: RESIZE - %d", pcb.Pid, tamanio) //LOG OBLIGATORIO

			resto := tamanio % config.Tamaño_pagina_memoria.Tamaño

			cantidad_pagina_pedir := tamanio / config.Tamaño_pagina_memoria.Tamaño

			if resto != 0 {
				cantidad_pagina_pedir = (tamanio / config.Tamaño_pagina_memoria.Tamaño) + 1
			}

			if tamanio == 0 {
				Borrar_de_tlb(pcb.Pid)
			}

			confirmacion := Reservar_paginas(pcb.Pid, cantidad_pagina_pedir)

			if confirmacion == "OUT_OF_MEMORY" {
				log.Println("OUT OF MEMORY")
				pcb.Bloqueo.Motivo = "OUT_OF_MEMORY"
				return pcb, nil
			}

		case "MOV_IN":

			pcb.Program_counter++
			registro_datos := palabras[1]
			registro_direccion_logica := strings.TrimRight(palabras[2], "\n")

			log.Printf("PID: %d - Ejecutando: MOV_IN - %s %s", pcb.Pid, registro_datos, registro_direccion_logica) //LOG OBLIGATORIO

			prefijo := registro_direccion_logica[:1]

			prefijo_datos := registro_datos[:1]

			var valorDireccionLogica int

			if prefijo == "E" || registro_direccion_logica == "SI" || registro_direccion_logica == "DI" {

				valor1, err := Obtener_valor_registro32(pcb, registro_direccion_logica)
				if err != nil {
					log.Println("Error al obtener valor de registro")
					return nil, err
				}
				valorDireccionLogica = int(valor1)

			} else {
				valor1, err := Obtener_valor_registro8(pcb, registro_direccion_logica)
				if err != nil {
					log.Println("Error al obtener valor de registro")
					return nil, err
				}
				valorDireccionLogica = int(valor1)
			}

			var cantidad_bytes_leer int

			if prefijo_datos == "E" || registro_datos == "SI" || registro_datos == "DI" {
				cantidad_bytes_leer = 4

			} else {
				cantidad_bytes_leer = 1
			}

			var cadena string

			for {

				if cantidad_bytes_leer == 0 {
					break
				}
				pagina, dezplazamiento, marco := Mmu(pcb, valorDireccionLogica)

				limite_bytes := config.Tamaño_pagina_memoria.Tamaño - dezplazamiento

				if limite_bytes > cantidad_bytes_leer {
					limite_bytes = cantidad_bytes_leer
				}

				valor := Pedido_lectura(pcb.Pid, pagina, dezplazamiento, limite_bytes, marco)

				for _, otroValor := range valor {

					valor_string := strconv.Itoa(otroValor)

					cadena += valor_string

				}

				direccion_fisica := pagina*config.Tamaño_pagina_memoria.Tamaño + dezplazamiento

				log.Printf("PID: %d - Acción: LEER - Dirección Física: %d - Valor: %s ", pcb.Pid, direccion_fisica, cadena)

				cantidad_bytes_leer -= limite_bytes

				valorDireccionLogica += limite_bytes

			}

			cadena_int, err := strconv.Atoi(cadena)

			if err != nil {
				fmt.Println("Error al convertir el string a entero:", err)
				return nil, err
			}

			if prefijo == "E" || registro_datos == "SI" || registro_datos == "DI" {
				valor32 := uint32(cadena_int)
				Obtener_setear_valor_registro32(pcb, registro_datos, uint32(valor32))

			} else {
				valor8 := uint8(cadena_int)
				Obtener_setear_valor_registro8(pcb, registro_datos, valor8)
			}

		case "MOV_OUT":

			pcb.Program_counter++
			registro_direccion_logica := palabras[1]
			registro_datos := strings.TrimRight(palabras[2], "\n")

			log.Printf("PID: %d - Ejecutando: MOV_OUT - %s %s", pcb.Pid, registro_direccion_logica, registro_datos) //LOG OBLIGATORIO

			prefijo := registro_direccion_logica[:1]

			prefijo_datos := registro_datos[:1]

			var valorDireccionLogica int

			if prefijo == "E" || registro_direccion_logica == "SI" || registro_direccion_logica == "DI" {

				valor1, err := Obtener_valor_registro32(pcb, registro_direccion_logica)
				if err != nil {
					log.Println("Error al obtener valor de registro")
					return nil, err
				}
				valorDireccionLogica = int(valor1)

			} else {
				valor1, err := Obtener_valor_registro8(pcb, registro_direccion_logica)
				if err != nil {
					log.Println("Error al obtener valor de registro")
					return nil, err
				}
				valorDireccionLogica = int(valor1)
			}

			log.Printf("Valor direccion logica: %d", valorDireccionLogica)

			var valorRegistro int

			if prefijo_datos == "E" || registro_datos == "SI" || registro_datos == "DI" {

				valor1, err := Obtener_valor_registro32(pcb, registro_datos)
				if err != nil {
					log.Println("Error al obtener valor de registro")
					return nil, err
				}
				valorRegistro = int(valor1)

			} else {
				valor1, err := Obtener_valor_registro8(pcb, registro_datos)
				if err != nil {
					log.Println("Error al obtener valor de registro")
					return nil, err
				}
				valorRegistro = int(valor1)
			}

			var cantidad_bytes_escribir int

			if prefijo_datos == "E" || registro_datos == "SI" || registro_datos == "DI" {
				cantidad_bytes_escribir = (valorRegistro / 255) + 1

			} else {
				cantidad_bytes_escribir = 1
			}

			for {

				if cantidad_bytes_escribir == 0 {
					break
				}
				pagina, dezplazamiento, marco := Mmu(pcb, valorDireccionLogica)

				limite_bytes := config.Tamaño_pagina_memoria.Tamaño - dezplazamiento

				if limite_bytes > cantidad_bytes_escribir {
					limite_bytes = cantidad_bytes_escribir
				}

				direccion_fisica := pagina*config.Tamaño_pagina_memoria.Tamaño + dezplazamiento

				log.Printf("PID: %d - Acción: ESCRIBIR - Dirección Física: %d - Valor: %d ", pcb.Pid, direccion_fisica, valorRegistro)

				valor_array := []int{valorRegistro}

				Pedido_escritura(pcb.Pid, pagina, dezplazamiento, limite_bytes, marco, valor_array)

				cantidad_bytes_escribir -= limite_bytes

				valorDireccionLogica += limite_bytes

			}

		case "COPY_STRING":

			pcb.Program_counter++

			aux := strings.TrimRight(palabras[1], "\n")

			tamanio, err := strconv.Atoi(aux)

			if err != nil {
				fmt.Println("Error al convertir el string a entero:", err)
				return nil, err
			}

			valor1, err := Obtener_valor_registro32(pcb, "SI")
			if err != nil {
				log.Println("Error al obtener valor de registro")
				return nil, err
			}

			valorRegistroOrigen := int(valor1)

			cantidad_bytes_leer := tamanio

			var cadena []int

			for {

				if cantidad_bytes_leer == 0 {
					break
				}
				pagina, dezplazamiento, marco := Mmu(pcb, valorRegistroOrigen)

				limite_bytes := config.Tamaño_pagina_memoria.Tamaño - dezplazamiento

				if limite_bytes > cantidad_bytes_leer {
					limite_bytes = cantidad_bytes_leer
				}

				valor := Pedido_lectura(pcb.Pid, pagina, dezplazamiento, limite_bytes, marco)

				// para el LOG

				var valor_log string

				for _, otroValor := range valor {

					valor_string := strconv.Itoa(otroValor)

					valor_log += valor_string

				}

				direccion_fisica := pagina*config.Tamaño_pagina_memoria.Tamaño + dezplazamiento

				log.Printf("PID: %d - Acción: LEER - Dirección Física: %d - Valor: %s ", pcb.Pid, direccion_fisica, valor_log)

				cadena = append(cadena, valor...)

				cantidad_bytes_leer -= limite_bytes

				valorRegistroOrigen += limite_bytes

			}

			valor2, err := Obtener_valor_registro32(pcb, "DI")

			if err != nil {
				log.Println("Error al obtener valor de registro")
				return nil, err
			}

			valorRegistroDestino := int(valor2)

			cantidad_bytes_escribir := tamanio

			for {

				if cantidad_bytes_escribir == 0 {
					break
				}
				pagina, dezplazamiento, marco := Mmu(pcb, valorRegistroDestino)

				limite_bytes := config.Tamaño_pagina_memoria.Tamaño - dezplazamiento

				if limite_bytes > cantidad_bytes_escribir {
					limite_bytes = cantidad_bytes_escribir
				}

				if err != nil {
					fmt.Println("Error al convertir el string a entero:", err)
					return nil, err
				}

				cadena_escribir := cadena[:limite_bytes]

				//para el log

				direccion_fisica := pagina*config.Tamaño_pagina_memoria.Tamaño + dezplazamiento

				var valor_log string

				for _, otroValor := range cadena_escribir {

					valor_string := strconv.Itoa(otroValor)

					valor_log += valor_string

				}

				log.Printf("PID: %d - Acción: ESCRIBIR - Dirección Física: %d - Valor: %s ", pcb.Pid, direccion_fisica, valor_log)

				cadena = cadena[limite_bytes:]

				Pedido_escritura(pcb.Pid, pagina, dezplazamiento, limite_bytes, marco, cadena_escribir)

				cantidad_bytes_escribir -= limite_bytes

				valorRegistroDestino += limite_bytes

			}

		}

		//pcb.Program_counter++

	}

	pcb.Bloqueo.Motivo = "EXIT"
	return pcb, nil

}

// Función para obtener el valor de un registro en base a su nombre
func Obtener_setear_valor_registro8(pcb *estructuras.Pcb_contexto, registro string, valor uint8) {
	// Determinar qué registro se está solicitando y devolver su valor
	switch registro {
	case "AX":
		pcb.AX = valor
	case "BX":
		pcb.BX = valor
	case "CX":
		pcb.CX = valor
	case "DX":
		pcb.DX = valor
	default:
		fmt.Errorf("Registro no válido: %s", registro)
	}
}

func Obtener_setear_valor_registro32(pcb *estructuras.Pcb_contexto, registro string, valor uint32) {
	// Determinar qué registro se está solicitando y devolver su valor
	switch registro {
	case "EAX":
		pcb.EAX = valor
	case "EBX":
		pcb.EBX = valor
	case "ECX":
		pcb.ECX = valor
	case "EDX":
		pcb.EDX = valor
	case "SI":
		pcb.SI = valor
	case "DI":
		pcb.DI = valor
	default:
		fmt.Errorf("Registro no válido: %s", registro)
	}
}

func Sumar_valor_registro8(pcb *estructuras.Pcb_contexto, registro string, valor uint8) {
	// Determinar qué registro se está solicitando y devolver su valor
	switch registro {
	case "AX":
		pcb.AX += valor
	case "BX":
		pcb.BX += valor
	case "CX":
		pcb.CX += valor
	case "DX":
		pcb.DX += valor
	default:
		fmt.Errorf("Registro no válido: %s", registro)
	}
}

func Sumar_valor_registro32(pcb *estructuras.Pcb_contexto, registro string, valor uint32) {
	// Determinar qué registro se está solicitando y devolver su valor
	switch registro {
	case "EAX":
		pcb.EAX += valor
	case "EBX":
		pcb.EBX += valor
	case "ECX":
		pcb.ECX += valor
	case "EDX":
		pcb.EDX += valor
	case "SI":
		pcb.SI += valor
	case "DI":
		pcb.DI += valor
	default:
		fmt.Errorf("Registro no válido: %s", registro)
	}
}

func Restar_valor_registro8(pcb *estructuras.Pcb_contexto, registro string, valor uint8) {
	// Determinar qué registro se está solicitando y devolver su valor
	switch registro {
	case "AX":
		pcb.AX -= valor
	case "BX":
		pcb.BX -= valor
	case "CX":
		pcb.CX -= valor
	case "DX":
		pcb.DX -= valor
	default:
		fmt.Errorf("Registro no válido: %s", registro)
	}
}

func Restar_valor_registro32(pcb *estructuras.Pcb_contexto, registro string, valor uint32) {
	// Determinar qué registro se está solicitando y devolver su valor
	switch registro {
	case "EAX":
		pcb.EAX -= valor
	case "EBX":
		pcb.EBX -= valor
	case "ECX":
		pcb.ECX -= valor
	case "EDX":
		pcb.EDX -= valor
	case "SI":
		pcb.SI -= valor
	case "DI":
		pcb.DI -= valor
	default:
		fmt.Errorf("Registro no válido: %s", registro)
	}
}

func Obtener_valor_registro8(pcb *estructuras.Pcb_contexto, registro string) (uint8, error) {
	// Determinar qué registro se está solicitando y devolver su valor
	switch registro {
	case "AX":
		return pcb.AX, nil
	case "BX":
		return pcb.BX, nil
	case "CX":
		return pcb.CX, nil
	case "DX":
		return pcb.DX, nil
	default:
		return 0, fmt.Errorf("Registro no válido: %s", registro)
	}
}

func Obtener_valor_registro32(pcb *estructuras.Pcb_contexto, registro string) (uint32, error) {
	// Determinar qué registro se está solicitando y devolver su valor
	switch registro {
	case "EAX":
		return pcb.EAX, nil
	case "EBX":
		return pcb.EBX, nil
	case "ECX":
		return pcb.ECX, nil
	case "EDX":
		return pcb.EDX, nil
	case "SI":
		return pcb.SI, nil
	case "DI":
		return pcb.DI, nil
	default:
		return 0, fmt.Errorf("Registro no válido: %s", registro)
	}
}

func Verificar_registro_distinto_cero(pcb *estructuras.Pcb_contexto, registro string) (bool, error) {

	switch registro {
	case "AX":
		return pcb.AX != 0, nil
	case "BX":
		return pcb.BX != 0, nil
	case "CX":
		return pcb.CX != 0, nil
	case "DX":
		return pcb.DX != 0, nil
	case "EAX":
		return pcb.EAX != 0, nil
	case "EBX":
		return pcb.EBX != 0, nil
	case "ECX":
		return pcb.ECX != 0, nil
	case "EDX":
		return pcb.EDX != 0, nil
	case "SI":
		return pcb.SI != 0, nil
	case "DI":
		return pcb.DI != 0, nil
	default:
		return false, fmt.Errorf("Registro no válido: %s", registro)
	}
}

func Mmu(pcb *estructuras.Pcb_contexto, valorDireccionLogica int) (int, int, int) {

	tamaño_pagina := config.Tamaño_pagina_memoria.Tamaño
	//fmt.Print(tamaño_pagina)

	numero_pagina := math.Floor(float64(valorDireccionLogica / tamaño_pagina))

	dezplazamiento := valorDireccionLogica - (int(numero_pagina) * tamaño_pagina)

	if config.Cpu.Number_felling_tlb > 0 {

		marco_tlb := Buscar_en_tlb(pcb.Pid, int(numero_pagina))

		if marco_tlb == -1 {

			log.Printf("PID: %d - TLB MISS - Pagina: %d", pcb.Pid, int(numero_pagina))

			marco_correspondiente := Pedir_marco_por_tlb_miss(pcb.Pid, int(numero_pagina))

			Agregar_marco_TLB(pcb.Pid, int(numero_pagina), marco_correspondiente)

			return int(numero_pagina), dezplazamiento, marco_correspondiente
		}

		return int(numero_pagina), dezplazamiento, marco_tlb

	}

	log.Printf("PID: %d - TLB MISS - Pagina: %d", pcb.Pid, int(numero_pagina)) //TLB deshabilitada

	return int(numero_pagina), dezplazamiento, -1 //TLB deshabilitada

}

func Pedido_lectura(pid_proceso int, pagina int, desplazamiento int, cantidad_bytes_leer int, marco int) []int {

	pedido := estructuras.Pedido_lectura_memoria{
		Pid:                 pid_proceso,
		Desplazamiento:      desplazamiento,
		Marco:               marco,
		Pagina:              pagina,
		Cantidad_bytes_leer: cantidad_bytes_leer,
	}

	body, err := json.Marshal(pedido)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/pedido_lectura", config.Cpu.Ip_memory, config.Cpu.Port_memory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", config.Cpu.Ip_memory, config.Cpu.Port_memory)
	}

	var respuesta estructuras.Respuesta_lectura_memoria

	json.NewDecoder(resp.Body).Decode(&respuesta)

	log.Println("Valor leido:", respuesta.Valor)

	return respuesta.Valor

}

func Pedido_escritura(pid_proceso int, pagina int, desplazamiento int, cantidad_bytes_escribir int, marco int, valor []int) {

	pedido := estructuras.Pedido_escritura_memoria{
		Pid:                     pid_proceso,
		Desplazamiento:          desplazamiento,
		Marco:                   marco,
		Pagina:                  pagina,
		Cantidad_bytes_escribir: cantidad_bytes_escribir,
		Valor:                   valor,
	}

	body, err := json.Marshal(pedido)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/pedido_escritura", config.Cpu.Ip_memory, config.Cpu.Port_memory)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", config.Cpu.Ip_memory, config.Cpu.Port_memory)
	}

	log.Println("Escritura realizada:" + resp.Status)

}

func Buscar_en_tlb(pid int, pagina int) int {

	for idx, entrada := range TLB {
		if entrada.PID == pid && entrada.Pagina == pagina {
			log.Printf("PID: %d - TLB HIT - Pagina: %d", pid, entrada.Pagina)
			if config.Cpu.Algorithm_tlb == "LRU" {

				TLB = append(TLB[:idx], TLB[idx+1:]...)

				TLB = append([]*config.TLBEntry{entrada}, TLB...)
			}
			return entrada.Marco
		}
	}

	return -1
}

func Pedir_marco_por_tlb_miss(pid int, pagina int) int {

	pedido := estructuras.TLB_miss{
		Pid:           pid,
		Numero_pagina: pagina,
	}

	body, err := json.Marshal(pedido)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/marco_tlb", config.Cpu.Ip_memory, config.Cpu.Port_memory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", config.Cpu.Ip_memory, config.Cpu.Port_memory)
	}

	var respuesta int

	json.NewDecoder(resp.Body).Decode(&respuesta)

	log.Println("Marco:", respuesta)

	return respuesta

}

func Agregar_marco_TLB(pid int, pagina int, marco int) {
	nuevo_marco := &config.TLBEntry{
		PID:    pid,
		Pagina: pagina,
		Marco:  marco,
	}

	switch config.Cpu.Algorithm_tlb {
	case "FIFO":
		if len(TLB) == config.Cpu.Number_felling_tlb {

			TLB = TLB[1:]
		}
		TLB = append(TLB, nuevo_marco)

	case "LRU":
		if len(TLB) == config.Cpu.Number_felling_tlb {
			TLB = TLB[:len(TLB)-1]
		}
		TLB = append([]*config.TLBEntry{nuevo_marco}, TLB...)
	}

}

func Reservar_paginas(pid int, cantidad_pagina_pedir int) string {

	pedido := estructuras.Resize{
		Pid:    pid,
		Ajuste: cantidad_pagina_pedir,
	}

	//fmt.Println("PID: ", pid)

	body, err := json.Marshal(pedido)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/resize", config.Cpu.Ip_memory, config.Cpu.Port_memory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", config.Cpu.Ip_memory, config.Cpu.Port_memory)
	}

	//Hacer la respuesta Correspondiente
	var respuesta estructuras.Respuesta_reservar_paginas
	json.NewDecoder(resp.Body).Decode(&respuesta)

	return respuesta.Estado
}

func Borrar_de_tlb(pid int) {

	for idx, entrada := range TLB {
		if entrada.PID == pid {
			TLB = append(TLB[:idx], TLB[idx+1:]...)
		}
	}

}
