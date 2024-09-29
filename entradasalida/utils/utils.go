package utils

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sisoputnfrba/tp-golang/entradasalida/FS"
	"github.com/sisoputnfrba/tp-golang/entradasalida/config"
	"github.com/sisoputnfrba/tp-golang/utils/estructuras"
)

var Stdin sync.Mutex
var Stdout sync.Mutex
var Generica sync.Mutex

func Ejecutar_interfaz(w http.ResponseWriter, r *http.Request) {

	switch config.Io.Type {
	case "GENERICA":

		Generica.Lock()

		var ejecutar estructuras.Ejecucion_interfaz_generica

		err := json.NewDecoder(r.Body).Decode(&ejecutar)
		if err != nil {
			log.Println("Error al decodificar la instruccion")
		}

		log.Printf("PID: %d - Operacion: INTERFAZ GENERICA", ejecutar.Pid)

		tiempo := ejecutar.Tiempo

		time.Sleep(time.Duration(tiempo) * time.Duration(config.Io.Unit_work_time) * time.Millisecond)

		Generica.Unlock()

		w.WriteHeader(http.StatusOK)

		w.Write([]byte("ok"))

	case "STDOUT":

		Stdout.Lock()

		var ejecutar estructuras.Ejecucion_interfaz_READ

		err := json.NewDecoder(r.Body).Decode(&ejecutar)
		if err != nil {
			log.Println("Error al decodificar la instruccion")
		}

		log.Printf("PID: %d - Operacion: STDOUT", ejecutar.Pid)

		pid := ejecutar.Pid

		direcciones_fisicas := ejecutar.Direcciones_fisicas

		var arrayBytes []byte

		for _, direccion_fisica := range direcciones_fisicas {

			valor := Lectura_memoria(pid, direccion_fisica.Pagina, direccion_fisica.Desplazamiento, direccion_fisica.Cantidad_bytes)

			for _, valor := range valor {

				arrayBytes = append(arrayBytes, byte(valor))
			}

		}

		str := string(arrayBytes)

		time.Sleep(time.Duration(config.Io.Unit_work_time) * time.Millisecond)

		log.Println("LECTURA :", str)

		Stdout.Unlock()

		w.WriteHeader(http.StatusOK)

		w.Write([]byte("ok"))

	case "STDIN":

		Stdin.Lock()

		log.Println("STDIN")

		var ejecutar estructuras.Ejecucion_interfaz_READ

		err := json.NewDecoder(r.Body).Decode(&ejecutar)
		if err != nil {
			log.Println("Error al decodificar la instruccion")
		}

		log.Printf("PID: %d - Operacion: STDIN", ejecutar.Pid)

		log.Println("Ingrese un valor:")

		reader := bufio.NewReader(os.Stdin)

		lectura, _ := reader.ReadString('\n')

		lectura = strings.TrimRight(lectura, "\n")

		direcciones_fisicas := ejecutar.Direcciones_fisicas

		//arrayBytes := []byte{10, 20, 30, 40, 50}

		lectura_bytes := []byte(lectura)

		var valores []int

		for i := 0; i < len(lectura_bytes); i++ {
			valores = append(valores, int(lectura_bytes[i]))
		}

		for _, direccion_fisica := range direcciones_fisicas {

			var valores_escritura []int

			for i := 0; i < direccion_fisica.Cantidad_bytes; i++ {
				valores_escritura = append(valores_escritura, valores[i])
			}

			Pedido_escritura(ejecutar.Pid, direccion_fisica.Pagina, direccion_fisica.Desplazamiento, direccion_fisica.Cantidad_bytes, valores_escritura)

			valores = valores[direccion_fisica.Cantidad_bytes:]

		}

		time.Sleep(time.Duration(config.Io.Unit_work_time) * time.Millisecond)

		Stdin.Unlock()

		w.WriteHeader(http.StatusOK)

		w.Write([]byte("ok"))

	case "DialFS":

		var ejecutar estructuras.Ejecucion_interfaz_FS

		err := json.NewDecoder(r.Body).Decode(&ejecutar)

		if err != nil {
			log.Println("Error al decodificar la instruccion")
		}

		log.Printf("PID: %d - Operacion: DialFS", ejecutar.Pid)

		switch ejecutar.Instruccion {

		case "IO_FS_CREATE":

			log.Printf("PID: %d - Crear Archivo: %s", ejecutar.Pid, ejecutar.Nombre_Archivo)

			posicion, err := FS.Asignacion_mejorada(config.Io.Dialfs_path)

			if err != nil {
				log.Println("Error al marcar el bit libre")
				break
			}

			var metadata estructuras.Metadata = estructuras.Metadata{
				InitialBlock: posicion,
				Size:         0,
			}

			ruta := filepath.Join(config.Io.Dialfs_path, ejecutar.Nombre_Archivo)

			FS.Crear_archivo_JSON(ruta, metadata)

			time.Sleep(time.Duration(config.Io.Unit_work_time) * time.Millisecond)

		case "DELETE":

			log.Printf("PID: %d - Eliminar Archivo: %s", ejecutar.Pid, ejecutar.Nombre_Archivo)

			ruta := filepath.Join(config.Io.Dialfs_path, ejecutar.Nombre_Archivo)

			metadata, err := FS.LeerArchivoJSON(ruta)
			if err != nil {
				fmt.Println("Error leyendo el archivo:", err)
				return
			}

			bloque_inicial := metadata.InitialBlock
			size := metadata.Size

			bloques_a_borrar := size / config.Io.Dialfs_block_size

			if size%config.Io.Dialfs_block_size != 0 {
				bloques_a_borrar++
			}

			for i := 0; i < bloques_a_borrar; i++ {

				FS.Marcar_posicion_en_cero(bloque_inicial + 1 + i) //Marcar apartir de su bloque inicial
			}

			FS.Borrar_archivo(ruta)

			time.Sleep(time.Duration(config.Io.Unit_work_time) * time.Millisecond)

		case "IO_FS_TRUNCATE":

			log.Printf("PID: %d - Truncar Archivo: %s", ejecutar.Pid, ejecutar.Nombre_Archivo)

			FS.Truncar_bloques(ejecutar.Nombre_Archivo, ejecutar.Tamanio_Trunc)

			time.Sleep(time.Duration(config.Io.Unit_work_time) * time.Millisecond)

		case "IO_FS_WRITE":

			pid := ejecutar.Pid

			direcciones_fisicas := ejecutar.Direcciones_fisicas

			//Obtener cuantos bytes tiene que leer

			bytes := 0

			for _, direccion_fisica := range direcciones_fisicas {

				bytes += direccion_fisica.Cantidad_bytes

			}

			log.Printf("PID: %d - Leer Archivo: %s - Tamaño a Escribir: %d - Puntero Archivo: %d", ejecutar.Pid, ejecutar.Nombre_Archivo, bytes, ejecutar.Puntero)

			var arrayBytes []byte

			for _, direccion_fisica := range direcciones_fisicas {

				valor := Lectura_memoria(pid, direccion_fisica.Pagina, direccion_fisica.Desplazamiento, direccion_fisica.Cantidad_bytes)

				for _, valor := range valor {

					arrayBytes = append(arrayBytes, byte(valor))
				}

			}

			ruta := filepath.Join(config.Io.Dialfs_path, ejecutar.Nombre_Archivo)

			ruta_bloques := filepath.Join(config.Io.Dialfs_path, "bloques.dat")

			metadata, err := FS.LeerArchivoJSON(ruta)
			if err != nil {
				fmt.Println("Error leyendo el archivo:", err)
				return
			}

			//str := fmt.Sprintf("%s", arrayBytes)

			//log.Println("Lectura Memoria:", str)

			bloque_inicial := metadata.InitialBlock

			FS.Escribir_bloques(ruta_bloques, bloque_inicial, ejecutar.Puntero, arrayBytes)

			time.Sleep(time.Duration(config.Io.Unit_work_time) * time.Millisecond)

		case "IO_FS_READ":

			pid := ejecutar.Pid

			direcciones_fisicas := ejecutar.Direcciones_fisicas

			//Obtener cuantos bytes tiene que leer

			bytes := 0

			for _, direccion_fisica := range direcciones_fisicas {

				bytes += direccion_fisica.Cantidad_bytes

			}

			log.Printf("PID: %d - Leer Archivo: %s - Tamaño a Leer: %d - Puntero Archivo: %d", ejecutar.Pid, ejecutar.Nombre_Archivo, bytes, ejecutar.Puntero)

			ruta := filepath.Join(config.Io.Dialfs_path, ejecutar.Nombre_Archivo)

			metadata, err := FS.LeerArchivoJSON(ruta)
			if err != nil {
				fmt.Println("Error leyendo el archivo:", err)
				return
			}

			ruta_bloques := filepath.Join(config.Io.Dialfs_path, "bloques.dat")

			bloque_inicial := metadata.InitialBlock

			valores, er := FS.Leer_bloques(ruta_bloques, bloque_inicial, ejecutar.Puntero, bytes)

			if er != nil {
				fmt.Println("Error leyendo los bloques:", er)
				return
			}

			var valores_int []int

			for i := 0; i < len(valores); i++ {
				valores_int = append(valores_int, int(valores[i]))
			}

			for _, direccion_fisica := range direcciones_fisicas {

				var valores_escritura []int

				for i := 0; i < direccion_fisica.Cantidad_bytes; i++ {
					valores_escritura = append(valores_escritura, valores_int[i])
				}

				Pedido_escritura(pid, direccion_fisica.Pagina, direccion_fisica.Desplazamiento, direccion_fisica.Cantidad_bytes, valores_escritura)

				valores_int = valores_int[direccion_fisica.Cantidad_bytes:]

			}

			time.Sleep(time.Duration(config.Io.Unit_work_time) * time.Millisecond)

		}
	}
}
func Lectura_memoria(pid int, pagina int, desplazamiento int, cantidad_bytes_leer int) []int {

	pedido := estructuras.Pedido_lectura_memoria{
		Pid:                 pid,
		Desplazamiento:      desplazamiento,
		Pagina:              pagina,
		Cantidad_bytes_leer: cantidad_bytes_leer,
		Marco:               -1,
	}

	body, err := json.Marshal(pedido)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/pedido_lectura", config.Io.Ip_memory, config.Io.Port_memory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", config.Io.Ip_memory, config.Io.Port_memory)
	}

	var respuesta estructuras.Respuesta_lectura_memoria

	json.NewDecoder(resp.Body).Decode(&respuesta)

	return respuesta.Valor

}

func Pedido_escritura(pid_proceso int, pagina int, desplazamiento int, cantidad_bytes_escribir int, valores []int) {

	pedido := estructuras.Pedido_escritura_memoria{
		Pid:                     pid_proceso,
		Desplazamiento:          desplazamiento,
		Pagina:                  pagina,
		Cantidad_bytes_escribir: cantidad_bytes_escribir,
		Valor:                   valores,
		Marco:                   -1,
	}

	body, err := json.Marshal(pedido)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/pedido_escritura", config.Io.Ip_memory, config.Io.Port_memory)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", config.Io.Ip_memory, config.Io.Port_memory)
	}

	log.Println("Escritura realizada:" + resp.Status)

}
