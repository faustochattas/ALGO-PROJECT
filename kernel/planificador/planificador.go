package planificador

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/sisoputnfrba/tp-golang/kernel/config"
	"github.com/sisoputnfrba/tp-golang/kernel/globales"
	"github.com/sisoputnfrba/tp-golang/utils/estructuras"
	"github.com/sisoputnfrba/tp-golang/utils/servidor"
)

var Contexto = make(chan *estructuras.Pcb_contexto, 1)

func IniciarPlanificadorLargoPlazo(chanMultiprogramacion chan estructuras.Pcb) {
	for {

		<-globales.HayPCBsEnNew

		globales.Semaforo_new.Wait()

		globales.MutexNew.Lock()
		pcb := globales.ListaNew[0]
		globales.ListaNew = globales.ListaNew[1:]
		globales.MutexNew.Unlock()

		chanMultiprogramacion <- *pcb

		globales.MutexReady.Lock()
		globales.ListaReady = append(globales.ListaReady, pcb)
		globales.MutexReady.Unlock()
		CambioEstadoProceso(pcb.Pid, "READY")
		globales.MutexReady.Lock()
		ListarReady()
		globales.MutexReady.Unlock()

		go Hay_procesos_ready()

	}

}

func Finalizador_de_procesos(chanMultiprogramacion chan estructuras.Pcb) {
	for {
		pcb := <-globales.Canal_Finalizar_Proceso

		if pcb.Estado != "EXECUTE" {
			log.Printf("Finaliza el proceso %d - Motivo: INTERRUPTED_BY_USER", pcb.Pid)
			Finalizar_proceso(pcb)
			<-chanMultiprogramacion
		} else {
			InterrumpirProceso("FINALIZAR_PROCESO")
		}

	}
}

func PlanificadorCortoPlazo(chanMultiprogramacion chan estructuras.Pcb) {

	for {
		globales.MutexReady.Lock()
		log.Printf("Cantidad de procesos en Ready: %d", len(globales.ListaReady))
		globales.MutexReady.Unlock()

		<-globales.CanalParaEjecutar

		var pcb *estructuras.Pcb

		globales.Semaforo_ready.Wait()

		//Se fija cual proceso mandar a execute
		if config.Kernel.Planning_algorithm == "VRR" {
			// Virtual Round Robin
			if len(globales.ListaReadyAux) != 0 { // revisa si la cola prioritaria tiene elementos, si es asi arranca por esta, sino la de ready normal
				globales.MutexReadyAux.Lock()
				pcb = globales.ListaReadyAux[0]
				globales.ListaReadyAux = globales.ListaReadyAux[1:]
				globales.MutexReadyAux.Unlock()

				log.Println("Proceso en ejecucion: PID ", pcb.Pid)

			} else {

				pcb = globales.ListaReady[0]
				globales.MutexReady.Lock()
				globales.ListaReady = globales.ListaReady[1:]
				globales.MutexReady.Unlock()

				log.Println("Proceso en ejecucion: PID ", pcb.Pid)
			}
			//logica de VRR
			//Hacer una lista de ready aux
		} else { //FIFO o RR
			pcb = globales.ListaReady[0]
			globales.MutexReady.Lock()
			globales.ListaReady = globales.ListaReady[1:]
			globales.MutexReady.Unlock()

			log.Println("Proceso en ejecucion: PID ", pcb.Pid)

		}

		CambioEstadoProceso(pcb.Pid, "EXECUTE")

		go enviar_pcb_a_cpu(pcb)

		var pcbContexto *estructuras.Pcb_contexto

		switch config.Kernel.Planning_algorithm {
		case "FIFO":
			// FIFO
			pcbContexto = <-Contexto
			Actualizar_pcb(pcb, pcbContexto)

		case "RR":
			quantum := config.Kernel.Quantum // Obtengo el quantum del archivo de configuración
			// Esperar hasta que el quantum termine
			time.Sleep(time.Duration(quantum) * time.Millisecond)
			InterrumpirProceso("QUANTUM")
			pcbContexto = <-Contexto
			Actualizar_pcb(pcb, pcbContexto)

		case "VRR":

			quantum := pcb.QuantumRestante
			timer := time.NewTimer(time.Duration(quantum) * time.Millisecond)
			star := time.Now()
			select {
			case <-timer.C:
				InterrumpirProceso("QUANTUM")
				pcbContexto = <-Contexto
				Actualizar_pcb(pcb, pcbContexto)
				pcb.QuantumRestante = config.Kernel.Quantum

			case pcbContexto = <-Contexto:
				elapsed := int(time.Since(star).Milliseconds())
				quantumRestante := quantum - elapsed
				if quantumRestante > 0 {
					pcb.QuantumRestante = quantumRestante
				}

				Actualizar_pcb(pcb, pcbContexto)

				log.Print("Quantum Restante: ", quantumRestante, "\n")
			}

		default:
		}

		globales.Semaforo_exec.Wait()

		if pcb.Estado == "EXIT" {
			pcbContexto.Bloqueo.Motivo = "CASO_EXCEPCIONAL"
		}

		switch pcbContexto.Bloqueo.Motivo {
		case "EXIT":

			log.Printf("Finaliza el proceso %d - Motivo: SUCCESS", pcb.Pid)
			go Finalizar_proceso(pcb)
			<-chanMultiprogramacion

		case "INTERRUPTED_BY_USER":

			log.Printf("Finaliza el proceso %d - Motivo: INTERRUPTED_BY_USER", pcb.Pid)
			go Finalizar_proceso(pcb)
			<-chanMultiprogramacion

		case "FIN_QUANTUM":
			log.Printf("PID: %d - Desalojado por fin de Quantum", pcb.Pid)

			Agregar_a_ready(pcb)
			CambioEstadoProceso(pcb.Pid, "READY")

			go Hay_procesos_ready()

		case "IO_GEN_SLEEP":

			go Bloqueo_IO_GENERICO(pcb, chanMultiprogramacion, pcbContexto.Bloqueo.Parametro1, pcbContexto.Bloqueo.Parametro2)

		case "IO_STDIN_READ":

			go Bloqueo_IO_STDIN_READ(pcb, chanMultiprogramacion, pcbContexto.Bloqueo.Parametro1, pcbContexto.Bloqueo.Direcciones_Fisicas)

		case "IO_STDOUT_WRITE":

			go Bloqueo_IO_STDOUT_WRITE(pcb, chanMultiprogramacion, pcbContexto.Bloqueo.Parametro1, pcbContexto.Bloqueo.Direcciones_Fisicas)

		case "IO_FS_CREATE":
			// CambiarEstadoPcb(pcb, "BLOCKED")
			go Bloqueo_IO_FS_CREATE_DELETE(pcb, chanMultiprogramacion, pcbContexto.Bloqueo.Parametro1, pcbContexto.Bloqueo.Parametro2, pcbContexto.Bloqueo.Motivo)

		case "IO_FS_DELETE":
			//	CambiarEstadoPcb(pcb, "BLOCKED")
			go Bloqueo_IO_FS_CREATE_DELETE(pcb, chanMultiprogramacion, pcbContexto.Bloqueo.Parametro1, pcbContexto.Bloqueo.Parametro2, pcbContexto.Bloqueo.Motivo)

		case "IO_FS_TRUNCATE":

			parametro3, _ := strconv.Atoi(pcbContexto.Bloqueo.Parametro3)

			go Bloqueo_IO_FS_TRUNCATE(pcb, chanMultiprogramacion, pcbContexto.Bloqueo.Parametro1, pcbContexto.Bloqueo.Parametro2, parametro3)

		case "IO_FS_WRITE":

			parametro3, _ := strconv.Atoi(pcbContexto.Bloqueo.Parametro3) //puntero archivo

			go Bloqueo_IO_FS_WRITE(pcb, chanMultiprogramacion, pcbContexto.Bloqueo.Parametro1, pcbContexto.Bloqueo.Parametro2, parametro3, pcbContexto.Bloqueo.Direcciones_Fisicas)

		case "IO_FS_READ":

			parametro3, _ := strconv.Atoi(pcbContexto.Bloqueo.Parametro3)

			go Bloqueo_IO_FS_READ(pcb, chanMultiprogramacion, pcbContexto.Bloqueo.Parametro1, pcbContexto.Bloqueo.Parametro2, parametro3, pcbContexto.Bloqueo.Direcciones_Fisicas)

		case "SIGNAL":
			recurso := pcbContexto.Bloqueo.Parametro1

			Signal(recurso)

			Analizar_quantum_y_agregar_ready(pcb)

			go Hay_procesos_ready()

		case "WAIT":
			recursoConsumir := pcbContexto.Bloqueo.Parametro1

			if !Wait(pcb, recursoConsumir) {
				log.Printf("Proceso %d bloqueado esperando %v\n", pcb.Pid, recursoConsumir)
			} else {
				log.Printf("Proceso %d consumió %v\n", pcb.Pid, recursoConsumir)

				go Hay_procesos_ready()
			}

		case "OUT_OF_MEMORY":
			log.Printf("Finaliza el proceso %d - Motivo: OUT_OF_MEMORY", pcb.Pid)
			Finalizar_proceso(pcb)

		case "CASO_EXCEPCIONAL":
			go Hay_procesos_ready()
			<-chanMultiprogramacion

		default:
			log.Println("Error en el motivo")
		}

	}
}

func InterrumpirProceso(motivo string) {
	contexto := servidor.Mensaje{Mensaje: motivo}

	body, err := json.Marshal(contexto)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/interrumpir_proceso", config.Kernel.Ip_cpu, config.Kernel.Port_cpu)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", config.Kernel.Ip_cpu, config.Kernel.Port_cpu)
	}

	log.Printf("Respuesta del servidor: %s", resp.Status)

}

func Actualizar_pcb(pcb *estructuras.Pcb, pcbContexto *estructuras.Pcb_contexto) {
	pcb.AX = pcbContexto.AX
	pcb.BX = pcbContexto.BX
	pcb.CX = pcbContexto.CX
	pcb.DX = pcbContexto.DX
	pcb.EAX = pcbContexto.EAX
	pcb.EBX = pcbContexto.EBX
	pcb.ECX = pcbContexto.ECX
	pcb.EDX = pcbContexto.EDX
	pcb.SI = pcbContexto.SI
	pcb.DI = pcbContexto.DI
	pcb.Program_counter = pcbContexto.Program_counter
}

func enviar_pcb_a_cpu(pcb *estructuras.Pcb) {
	contexto := estructuras.Pcb_contexto{Pid: pcb.Pid, Program_counter: pcb.Program_counter,
		AX: pcb.AX, BX: pcb.BX, CX: pcb.CX, DX: pcb.DX,
		EAX: pcb.EAX, EBX: pcb.EBX, ECX: pcb.ECX, EDX: pcb.EDX, SI: pcb.SI, DI: pcb.DI,
		Bloqueo: &estructuras.Motivo{Motivo: "", Parametro1: "", Parametro2: "", Parametro3: "", Direcciones_Fisicas: nil}}

	body, err := json.Marshal(contexto)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/contexto", config.Kernel.Ip_cpu, config.Kernel.Port_cpu)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", config.Kernel.Ip_cpu, config.Kernel.Port_cpu)
	}

	var pcbNuevo *estructuras.Pcb_contexto

	json.NewDecoder(resp.Body).Decode(&pcbNuevo)

	log.Printf("Respuesta del servidor: %s", resp.Status)
	//log.Printf("respuesta del servidor MOtivo : %s", pcbNuevo.Bloqueo.Motivo)

	Contexto <- pcbNuevo

}

/*
func CambiarEstadoPcb(pcb *estructuras.Pcb, estado string) {
	log.Printf("PID: %d - Estado Anterior: %s - Estado Actual: %s", pcb.Pid, pcb.Estado, estado)
	pcb.Estado = estado
}
*/

func CrearProceso(pid int) {

	Pcb := new(estructuras.Pcb)

	Pcb.Pid = pid
	Pcb.Program_counter = 0
	Pcb.QuantumRestante = config.Kernel.Quantum
	Pcb.Estado = "-"

	globales.Pcb_por_pid[pid] = Pcb

	CambioEstadoProceso(pid, "NEW")

	// Agregar el proceso a la lista de new
	globales.MutexNew.Lock()
	globales.ListaNew = append(globales.ListaNew, Pcb)
	globales.MutexNew.Unlock()

	log.Printf("Se crea el Pid %d en NEW\n", pid)

	globales.HayPCBsEnNew <- 1

}

func CambioEstadoProceso(pid int, estado string) {
	globales.MutexProcesos.Lock()
	pcb := globales.Pcb_por_pid[pid]
	log.Printf("PID: %d - Estado Anterior: %s - Estado Actual: %s", pid, pcb.Estado, estado)
	pcb.Estado = estado
	globales.MutexProcesos.Unlock()
}

func ListarReady() {
	var ListaPids []int = make([]int, 0)
	for _, proceso := range globales.ListaReady {
		ListaPids = append(ListaPids, proceso.Pid)
	}
	log.Println("Cola Ready: ", ListaPids)
}

func Bloqueo_IO_GENERICO(pcb *estructuras.Pcb, canal chan estructuras.Pcb, nombre string, tiempo string) {

	//checkear que este exista en la lista de dispositivos
	log.Printf("PID: %d - Bloqueado por: INTERFAZ", pcb.Pid)

	Agregar_a_blocked(pcb)
	CambioEstadoProceso(pcb.Pid, "BLOCKED")

	for _, io := range globales.ListaInterfacesGeneral {
		if io.Nombre == nombre {
			log.Println("Existe este dispositivo: ", nombre)
			tiempo, err := strconv.Atoi(tiempo)
			if err != nil {
				log.Println("Error al convertir el tiempo")
			}

			confirmacion := Ejecutar_Io_Generica(pcb.Pid, io.Port, io.Ip, tiempo)

			if confirmacion {
				log.Println("Conexion exitosa")

				log.Println("PASO TIEMPO DE BLOQUEO: ", pcb.Pid)

				Sacar_de_blocked(pcb)

				Analizar_quantum_y_agregar_ready(pcb)

				Hay_procesos_ready()

				return

			} else {
				log.Printf("Finaliza el proceso %d - Motivo: INVALID_INTERFACE", pcb.Pid)
				Sacar_de_blocked(pcb)
				Finalizar_proceso(pcb)
				<-canal
				return
			}
		}
	}
	log.Printf("Finaliza el proceso %d - Motivo: INVALID_INTERFACE", pcb.Pid)
	Sacar_de_blocked(pcb)
	Finalizar_proceso(pcb)

	<-canal
}

func Bloqueo_IO_STDIN_READ(pcb *estructuras.Pcb, canal chan estructuras.Pcb, nombre string, direcciones_fisicas []*estructuras.Direccion_fisica) {
	//checkear que este exista en la lista de dispositivos
	log.Printf("PID: %d - Bloqueado por: INTERFAZ", pcb.Pid)

	Agregar_a_blocked(pcb)
	CambioEstadoProceso(pcb.Pid, "BLOCKED")

	for _, io := range globales.ListaInterfacesGeneral {
		if io.Nombre == nombre {
			log.Println("Existe este dispositivo: ", nombre)

			confirmacion := Ejecutar_IO_STDIN_READ(io.Port, io.Ip, direcciones_fisicas, pcb.Pid)

			if confirmacion {
				log.Println("Conexion exitosa")

				Sacar_de_blocked(pcb)

				Analizar_quantum_y_agregar_ready(pcb)

				Hay_procesos_ready()

				return

			} else {
				log.Printf("Finaliza el proceso %d - Motivo: INVALID_INTERFACE", pcb.Pid)

				Sacar_de_blocked(pcb)

				Finalizar_proceso(pcb)
				<-canal
				return
			}
		}
	}
	log.Printf("Finaliza el proceso %d - Motivo: INVALID_INTERFACE", pcb.Pid)
	Sacar_de_blocked(pcb)
	Finalizar_proceso(pcb)
	<-canal
}

func Bloqueo_IO_STDOUT_WRITE(pcb *estructuras.Pcb, canal chan estructuras.Pcb, nombre string, direcciones_fisicas []*estructuras.Direccion_fisica) {

	//checkear que este exista en la lista de dispositivos
	log.Printf("PID: %d - Bloqueado por: INTERFAZ", pcb.Pid)

	Agregar_a_blocked(pcb)
	CambioEstadoProceso(pcb.Pid, "BLOCKED")

	for _, io := range globales.ListaInterfacesGeneral {
		if io.Nombre == nombre {
			log.Println("Existe este dispositivo: ", nombre)

			confirmacion := Ejecutar_IO_STDIN_READ(io.Port, io.Ip, direcciones_fisicas, pcb.Pid)

			if confirmacion {
				log.Println("Conexion exitosa")

				Sacar_de_blocked(pcb)

				Analizar_quantum_y_agregar_ready(pcb)

				Hay_procesos_ready()

				return
			} else {
				log.Printf("Finaliza el proceso %d - Motivo: INVALID_INTERFACE", pcb.Pid)
				Sacar_de_blocked(pcb)
				Finalizar_proceso(pcb)
				<-canal
				return
			}
		}
	}
	log.Printf("Finaliza el proceso %d - Motivo: INVALID_INTERFACE", pcb.Pid)
	Sacar_de_blocked(pcb)
	Finalizar_proceso(pcb)
	//CambioEstadoProceso(pcb.Pid, "EXIT")
	<-canal
}

func Bloqueo_IO_FS_CREATE_DELETE(pcb *estructuras.Pcb, canal chan estructuras.Pcb, nombre_interfaz string, nombre_archivo string, instruccion string) {

	log.Printf("PID: %d - Bloqueado por: INTERFAZ", pcb.Pid)

	Agregar_a_blocked(pcb)
	CambioEstadoProceso(pcb.Pid, "BLOCKED")

	for _, io := range globales.ListaInterfacesGeneral {
		if io.Nombre == nombre_interfaz {
			log.Println("Existe este dispositivo: ", nombre_interfaz)

			confirmacion := Ejecutar_IO_FS_CREATE_DELETE(io.Port, io.Ip, nombre_archivo, pcb.Pid, instruccion)

			if confirmacion {
				log.Println("Conexion exitosa")

				Sacar_de_blocked(pcb)

				Analizar_quantum_y_agregar_ready(pcb)

				Hay_procesos_ready()

				return

			} else {
				log.Printf("Finaliza el proceso %d - Motivo: INVALID_INTERFACE", pcb.Pid)

				Sacar_de_blocked(pcb)

				Finalizar_proceso(pcb)
				<-canal
				return
			}
		}
	}

	log.Printf("Finaliza el proceso %d - Motivo: INVALID_INTERFACE", pcb.Pid)
	Sacar_de_blocked(pcb)
	Finalizar_proceso(pcb)
	//CambioEstadoProceso(pcb.Pid, "EXIT")
	<-canal

}

func Bloqueo_IO_FS_TRUNCATE(pcb *estructuras.Pcb, canal chan estructuras.Pcb, nombre_interfaz string, nombre_archivo string, tamanio int) {

	log.Printf("PID: %d - Bloqueado por: INTERFAZ", pcb.Pid)

	Agregar_a_blocked(pcb)
	CambioEstadoProceso(pcb.Pid, "BLOCKED")

	for _, io := range globales.ListaInterfacesGeneral {
		if io.Nombre == nombre_interfaz {
			log.Println("Existe este dispositivo: ", nombre_interfaz)

			confirmacion := Ejecutar_IO_FS_TRUNCATE(io.Port, io.Ip, nombre_archivo, pcb.Pid, tamanio)

			if confirmacion {
				log.Println("Conexion exitosa")

				Sacar_de_blocked(pcb)

				Analizar_quantum_y_agregar_ready(pcb)

				Hay_procesos_ready()

				return

			} else {
				log.Printf("Finaliza el proceso %d - Motivo: INVALID_INTERFACE", pcb.Pid)

				Sacar_de_blocked(pcb)

				Finalizar_proceso(pcb)
				<-canal
				return
			}
		}
	}

	log.Printf("Finaliza el proceso %d - Motivo: INVALID_INTERFACE", pcb.Pid)
	Sacar_de_blocked(pcb)
	Finalizar_proceso(pcb)
	//CambioEstadoProceso(pcb.Pid, "EXIT")
	<-canal

}

func Bloqueo_IO_FS_WRITE(pcb *estructuras.Pcb, canal chan estructuras.Pcb, nombre_interfaz string, nombre_archivo string, puntero_archivo int, direcciones_fisicas []*estructuras.Direccion_fisica) {

	Agregar_a_blocked(pcb)
	CambioEstadoProceso(pcb.Pid, "BLOCKED")

	for _, io := range globales.ListaInterfacesGeneral {
		if io.Nombre == nombre_interfaz {
			log.Println("Existe este dispositivo: ", nombre_interfaz)

			confirmacion := Ejecutar_IO_FS_WRITE(io.Port, io.Ip, nombre_archivo, pcb.Pid, puntero_archivo, direcciones_fisicas)

			if confirmacion {

				log.Println("Conexion exitosa")

				Sacar_de_blocked(pcb)

				Analizar_quantum_y_agregar_ready(pcb)

				Hay_procesos_ready()

				return

			} else {
				log.Printf("Finaliza el proceso %d - Motivo: INVALID_INTERFACE", pcb.Pid)

				Sacar_de_blocked(pcb)

				Finalizar_proceso(pcb)
				<-canal
				return
			}
		}
	}

	log.Printf("Finaliza el proceso %d - Motivo: INVALID_INTERFACE", pcb.Pid)
	Sacar_de_blocked(pcb)
	Finalizar_proceso(pcb)
	<-canal
}

func Bloqueo_IO_FS_READ(pcb *estructuras.Pcb, canal chan estructuras.Pcb, nombre_interfaz string, nombre_archivo string, puntero_archivo int, direcciones_fisicas []*estructuras.Direccion_fisica) {
	CambioEstadoProceso(pcb.Pid, "BLOCKED")
	Agregar_a_blocked(pcb)

	for _, io := range globales.ListaInterfacesGeneral {
		if io.Nombre == nombre_interfaz {
			log.Println("Existe este dispositivo: ", nombre_interfaz)

			confirmacion := Ejecutar_IO_FS_READ(io.Port, io.Ip, nombre_archivo, pcb.Pid, puntero_archivo, direcciones_fisicas) //usa el mismo que IO_FS_WRITE pero abstraer

			if confirmacion {

				log.Println("Conexion exitosa")

				Sacar_de_blocked(pcb)

				Analizar_quantum_y_agregar_ready(pcb)

				Hay_procesos_ready()

				return

			} else {
				log.Printf("Finaliza el proceso %d - Motivo: INVALID_INTERFACE", pcb.Pid)

				Sacar_de_blocked(pcb)

				Finalizar_proceso(pcb)
				<-canal
				return
			}
		}
	}

	log.Printf("Finaliza el proceso %d - Motivo: INVALID_INTERFACE", pcb.Pid)
	Sacar_de_blocked(pcb)
	Finalizar_proceso(pcb)
	<-canal
}

func Ejecutar_Io_Generica(pid int, puerto int, ip string, tiempo int) bool {

	//log.Println("Ip: ", ip)
	//log.Println("Puerto: ", puerto)

	mensaje := estructuras.Ejecucion_interfaz_generica{Pid: pid, Tiempo: tiempo}

	body, err := json.Marshal(mensaje)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/ejecutar_interfaz", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Se desconecto la IO")
		log.Printf("error enviando mensaje a ip:%s puerto:%d", ip, puerto)
		return false
	}

	log.Printf("Respuesta del servidor: %s", resp.Status)
	return true

}

func Ejecutar_IO_STDIN_READ(puerto int, ip string, direcciones_fisicas []*estructuras.Direccion_fisica, pid int) bool {

	mensaje := estructuras.Ejecucion_interfaz_READ{
		Pid:                 pid,
		Direcciones_fisicas: direcciones_fisicas,
	}

	body, err := json.Marshal(mensaje)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/ejecutar_interfaz", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Se desconecto la IO")
		log.Printf("error enviando mensaje a ip:%s puerto:%d", ip, puerto)
		return false
	}

	log.Printf("Respuesta del servidor: %s", resp.Status)
	return true

}

func Ejecutar_IO_STDOUT_WRITE(puerto int, ip string, direcciones_fisicas []*estructuras.Direccion_fisica, pid int) bool {

	mensaje := estructuras.Ejecucion_interfaz_READ{
		Pid:                 pid,
		Direcciones_fisicas: direcciones_fisicas,
	}

	body, err := json.Marshal(mensaje)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/ejecutar_interfaz", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Se desconecto la IO")
		log.Printf("Error enviando mensaje a ip:%s puerto:%d", ip, puerto)
		return false
	}

	log.Printf("Respuesta del servidor: %s", resp.Status)
	return true

}

func Ejecutar_IO_FS_CREATE_DELETE(puerto int, ip string, nombre_archivo string, pid int, instruccion string) bool {

	mensaje := estructuras.Ejecucion_interfaz_FS{
		Pid:            pid,
		Nombre_Archivo: nombre_archivo,
		Instruccion:    instruccion,
	}

	body, err := json.Marshal(mensaje)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/ejecutar_interfaz", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Se desconecto la IO")
		log.Printf("Error enviando mensaje a ip:%s puerto:%d", ip, puerto)
		return false
	}

	log.Printf("Respuesta del servidor: %s", resp.Status)
	return true

}

func Ejecutar_IO_FS_TRUNCATE(puerto int, ip string, nombre_archivo string, pid int, tamanio int) bool {

	mensaje := estructuras.Ejecucion_interfaz_FS{
		Pid:            pid,
		Nombre_Archivo: nombre_archivo,
		Instruccion:    "IO_FS_TRUNCATE",
		Tamanio_Trunc:  tamanio,
	}

	body, err := json.Marshal(mensaje)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/ejecutar_interfaz", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Se desconecto la IO")
		log.Printf("error enviando mensaje a ip:%s puerto:%d", ip, puerto)
		return false
	}

	log.Printf("Respuesta del servidor: %s", resp.Status)
	return true

}

func Ejecutar_IO_FS_WRITE(puerto int, ip string, nombre_archivo string, pid int, puntero_archivo int, direcciones_fisicas []*estructuras.Direccion_fisica) bool {
	mensaje := estructuras.Ejecucion_interfaz_FS{
		Pid:                 pid,
		Nombre_Archivo:      nombre_archivo,
		Instruccion:         "IO_FS_WRITE",
		Puntero:             puntero_archivo,
		Direcciones_fisicas: direcciones_fisicas,
	}

	body, err := json.Marshal(mensaje)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/ejecutar_interfaz", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Se desconecto la IO")
		log.Printf("Error enviando mensaje a ip:%s puerto:%d", ip, puerto)
		return false
	}

	log.Printf("Respuesta del servidor: %s", resp.Status)
	return true
}

func Ejecutar_IO_FS_READ(puerto int, ip string, nombre_archivo string, pid int, puntero_archivo int, direcciones_fisicas []*estructuras.Direccion_fisica) bool {
	mensaje := estructuras.Ejecucion_interfaz_FS{
		Pid:                 pid,
		Nombre_Archivo:      nombre_archivo,
		Instruccion:         "IO_FS_READ",
		Puntero:             puntero_archivo,
		Direcciones_fisicas: direcciones_fisicas,
	}

	body, err := json.Marshal(mensaje)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/ejecutar_interfaz", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Se desconecto la IO")
		log.Printf("Error enviando mensaje a ip:%s puerto:%d", ip, puerto)
		return false
	}

	log.Printf("Respuesta del servidor: %s", resp.Status)
	return true
}

func Eliminar_proceso_de_lista(pid int) {

	pcb := globales.Pcb_por_pid[pid]

	//i := 0

	switch pcb.Estado {
	/*
		case "NEW":
			for _, proceso := range globales.ListaNew {
				if proceso.Pid == pid {
					//TODO
				}
			}
	*/
	case "READY":

		Sacar_de_ready(pcb)

		log.Printf("SE ELIMINO DE READY")

	case "BLOCKED":

		Sacar_de_blocked(pcb)

		//Sacar_de_blocked_Recursos(pcb)

		log.Printf("SE ELIMINO DE BLOCKED")

	}

}

func Borrar_proceso_memoria(pid int) {
	//log.Println("Borrando Proceso de Memoria")
	mensaje := estructuras.Pedido_borrar_memoria{Pid: pid}

	body, err := json.Marshal(mensaje)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/borrar_proceso", config.Kernel.Ip_memory, config.Kernel.Port_memory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Error enviando mensaje a ip:%s puerto:%d", config.Kernel.Ip_memory, config.Kernel.Port_memory)
	}

	log.Printf("Respuesta del servidor: %s", resp.Status)

}

func Finalizar_proceso(pcb *estructuras.Pcb) {

	Eliminar_proceso_de_lista(pcb.Pid)

	CambioEstadoProceso(pcb.Pid, "EXIT")

	Desbloquear_proceso_por_recurso(pcb.Pid)

	Borrar_proceso_memoria(pcb.Pid)

}

func Desbloquear_proceso_por_recurso(pid int) {

	// Libera los recursos asignados al proceso.
	if recursos, existe := globales.RecursosUsados[pid]; existe {
		for _, recurso := range recursos {
			Signal(recurso.Nombre)

		}
		delete(globales.RecursosUsados, pid)
	}
}

func Signal(rec string) {
	recurso := BuscarRecurso(rec)

	if recurso == nil {
		recurso := &estructuras.Recurso{Nombre: rec, Instancia: 1}
		procesos, existe := globales.ListaBlockedRecursos[recurso.Nombre]
		if existe && len(procesos) > 0 {

			liberarRecurso(recurso, procesos)

		} else {
			globales.Recursos_disponibles = append(globales.Recursos_disponibles, recurso)
			log.Print("Recurso añadido a los recursos disponibles: ", recurso.Nombre)
		}
	} else {
		log.Print("Sume una instancia a : ", recurso.Nombre)
		otroRec := BuscarRecurso(rec)
		otroRec.Instancia++
	}
}

func liberarRecurso(recurso *estructuras.Recurso, procesos []*estructuras.Pcb) {

	for i, proceso := range procesos {

		if proceso.Estado == "BLOCKED" {

			Sacar_de_blocked(proceso)

			Analizar_quantum_y_agregar_ready(proceso)

			//log.Print("EL  ", i, " ESTABA BLOQUEADO ")

			log.Print("Libere un recurso bloqueado  ", recurso.Nombre, "  para el proceso  ", proceso.Pid)

			go Hay_procesos_ready()

			globales.RecursosUsados[proceso.Pid] = append(globales.RecursosUsados[proceso.Pid], recurso)
			proceso.Recursos_asignados = append(proceso.Recursos_asignados, recurso)

			globales.ListaBlockedRecursos[recurso.Nombre] = globales.ListaBlockedRecursos[recurso.Nombre][i:]

			return
		}
	}

}

func Wait(proceso *estructuras.Pcb, nombreRecurso string) bool {
	recurso := BuscarRecurso(nombreRecurso)
	if recurso == nil {
		// El recurso no está disponible, bloquea el proceso.
		claveRecurso := nombreRecurso

		Agregar_a_blocked(proceso)
		CambioEstadoProceso(proceso.Pid, "BLOCKED")
		globales.ListaBlockedRecursos[claveRecurso] = append(globales.ListaBlockedRecursos[claveRecurso], proceso)
		return false
	}

	// El recurso está disponible, lo asignamos al proceso.
	for i, r := range globales.Recursos_disponibles {
		if r == recurso {
			globales.Recursos_disponibles = append(globales.Recursos_disponibles[:i], globales.Recursos_disponibles[i+1:]...)
			break
		}
	}

	Sacar_de_blocked(proceso)

	Analizar_quantum_y_agregar_ready(proceso)

	globales.RecursosUsados[proceso.Pid] = append(globales.RecursosUsados[proceso.Pid], recurso)
	proceso.Recursos_asignados = append(proceso.Recursos_asignados, recurso)
	return true
}

func BuscarRecurso(nombre string) *estructuras.Recurso {
	for _, recurso := range globales.Recursos_disponibles {
		if recurso.Nombre == nombre {
			return recurso
		}
	}
	return nil
}

func Agregar_a_ready(pcb *estructuras.Pcb) {
	globales.MutexReady.Lock()
	globales.ListaReady = append(globales.ListaReady, pcb)
	globales.MutexReady.Unlock()
}

func Agregar_a_ready_aux(pcb *estructuras.Pcb) {
	globales.MutexReadyAux.Lock()
	globales.ListaReadyAux = append(globales.ListaReadyAux, pcb)
	globales.MutexReadyAux.Unlock()
}

func Agregar_a_blocked(pcb *estructuras.Pcb) {
	globales.MutexBlocked.Lock()
	globales.ListaBlocked = append(globales.ListaBlocked, pcb)
	globales.MutexBlocked.Unlock()
}

func Sacar_de_blocked(pcb *estructuras.Pcb) {
	globales.MutexBlocked.Lock()
	for i, proceso := range globales.ListaBlocked {
		if proceso.Pid == pcb.Pid {
			globales.ListaBlocked = append(globales.ListaBlocked[:i], globales.ListaBlocked[i+1:]...)
			break
		}
	}
	globales.MutexBlocked.Unlock()
}

func Sacar_de_ready(pcb *estructuras.Pcb) {
	globales.MutexReady.Lock()
	for i, proceso := range globales.ListaReady {
		if proceso.Pid == pcb.Pid {
			globales.ListaReady = append(globales.ListaReady[:i], globales.ListaReady[i+1:]...)
			break
		}
	}
	globales.MutexReady.Unlock()
}

func Hay_procesos_ready() {
	globales.CanalParaEjecutar <- 1
}

func Analizar_quantum_y_agregar_ready(pcb *estructuras.Pcb) {
	if pcb.QuantumRestante < config.Kernel.Quantum {

		Agregar_a_ready_aux(pcb)
		CambioEstadoProceso(pcb.Pid, "READY")

	} else {

		Agregar_a_ready(pcb)
		CambioEstadoProceso(pcb.Pid, "READY")

	}
}

func Sacar_de_blocked_Recursos(pcb *estructuras.Pcb) {
	for _, recurso := range pcb.Recursos_asignados {
		for i, proceso := range globales.ListaBlockedRecursos[recurso.Nombre] {
			if proceso.Pid == pcb.Pid {
				globales.ListaBlockedRecursos[recurso.Nombre] = append(globales.ListaBlockedRecursos[recurso.Nombre][:i], globales.ListaBlockedRecursos[recurso.Nombre][i+1:]...)
				break
			}
		}
	}
}
