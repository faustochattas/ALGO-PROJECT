package apis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/sisoputnfrba/tp-golang/kernel/config"
	"github.com/sisoputnfrba/tp-golang/kernel/globales"
	"github.com/sisoputnfrba/tp-golang/kernel/interfaces"
	"github.com/sisoputnfrba/tp-golang/kernel/planificador"
	"github.com/sisoputnfrba/tp-golang/utils/estructuras"
)

type ProcessRequest struct {
	Pid  int    `json:"pid"`
	Path string `json:"path"`
}

type ProcessResponse struct {
	Pid int `json:"Pid"`
}

func HandleRequestIniciar_Listar(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodPut:
		HandleIniciarProceso(w, r)
	case http.MethodGet:
		HandleListarProcesos(w, r)

	default:
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}

}

// Implementacion de iniciar proceso
func HandleIniciarProceso(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req ProcessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Error al decodificar la solicitud", http.StatusBadRequest)
		return
	}

	//fmt.Print("Iniciando proceso con path: ", req.Path)

	enviarPidyPath(req.Path, req.Pid)

	go planificador.CrearProceso(req.Pid) //Sumar el envio del path de instrucciones

	// Devolver el contenido como respuesta
	resp := ProcessResponse{
		Pid: req.Pid, //Devolver el Pid del proceso iniciado
	}

	//globales.Pid++ // Sumar 1 al pid

	//globales.Semaforo.Done()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func enviarPidyPath(path string, pid int) {
	path_pid := estructuras.Path_proceso{Pid: pid, Path_proceso: path}

	body, err := json.Marshal(path_pid)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/crear_proceso", config.Kernel.Ip_memory, config.Kernel.Port_memory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", config.Kernel.Ip_cpu, config.Kernel.Port_cpu)
	}

	log.Printf("respuesta del servidor: %s", resp.Status)
}

type ProcessInfo struct {
	PID   int    `json:"pid"`
	State string `json:"state"`
}

func ListProcesses() ([]ProcessInfo, error) {
	//globales.MutexProcesos.Lock()

	listaStruct := []ProcessInfo{}

	//fmt.Println("Cantidad de procesos: ", cant)

	for _, pcb := range globales.Pcb_por_pid {
		//fmt.Println("Proceso: ", pcb.Pid)
		//fmt.Println("Estado: ", pcb.Estado)
		listaStruct = append(listaStruct, ProcessInfo{
			PID:   pcb.Pid,
			State: pcb.Estado,
		})
	}
	//globales.MutexProcesos.Unlock()

	return listaStruct, nil

}

func HandleListarProcesos(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	// Obtener la lista de procesos
	processList, err := ListProcesses()
	if err != nil {
		http.Error(w, "Error al obtener la lista de procesos", http.StatusInternalServerError)
		return
	}

	// Responder con la lista de procesos
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(processList)
}

func HandleRequestTerminar_VerEstado(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodDelete:
		HandleTerminarProceso(w, r)
	case http.MethodGet:
		HandleVerEstadoProceso(w, r)

	default:
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}

}

func HandleTerminarProceso(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	// Obtener el PID del path de la solicitud
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) != 3 || pathParts[1] != "process" {
		http.Error(w, "Ruta incorrecta", http.StatusBadRequest)
		return
	}

	pidStr := pathParts[2]
	pidTerminar, err := strconv.Atoi(pidStr)

	if err != nil {
		http.Error(w, "Error al convertir el PID a entero", http.StatusBadRequest)
		return
	}

	pcb := globales.Pcb_por_pid[pidTerminar]

	if pcb == nil {
		http.Error(w, "No se encontro el proceso", http.StatusBadRequest)
		return
	}

	fmt.Println("Terminando proceso con PID: ", pidTerminar)

	globales.Canal_Finalizar_Proceso <- pcb

	//go planificador.Finalizar_proceso(pcb)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Proceso terminado"))
}

type ProcessStateResponse struct {
	State string `json:"state"`
}

func GetProcessState(pid int) (string, error) {
	estado := globales.Pcb_por_pid[pid]

	return estado.Estado, nil
}

func HandleVerEstadoProceso(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	// Obtener el PID del path de la solicitud
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) != 3 || pathParts[1] != "process" {
		http.Error(w, "Ruta incorrecta", http.StatusBadRequest)
		return
	}

	pidStr := pathParts[2]

	pidEstado, err := strconv.Atoi(pidStr)

	if err != nil {
		http.Error(w, "Error al convertir el PID a entero", http.StatusBadRequest)
		return
	}

	// Obtener el estado del proceso
	state, err := GetProcessState(pidEstado)
	if err != nil {
		http.Error(w, "No se pudo obtener el estado del proceso", http.StatusInternalServerError)
		return
	}

	// Responder con el estado del proceso
	resp := ProcessStateResponse{
		State: state,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func HandleIniciar_Pausar_Planificacion(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodPut:
		HandleIniciarPlanificacion(w, r)
	case http.MethodDelete:
		HandlePausarPlanificacion(w, r)

	default:
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}

}

func HandleIniciarPlanificacion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	globales.Semaforo_new.Done()
	globales.Semaforo_ready.Done()
	globales.Semaforo_exec.Done()

	globales.MutexNew.Unlock()
	globales.MutexReady.Unlock()
	//globales.MutexProcesos.Unlock()
	globales.MutexReadyAux.Unlock()
	globales.MutexBlocked.Unlock()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Planificación iniciada"))
}

func HandlePausarPlanificacion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	globales.Semaforo_new.Add(1)
	globales.Semaforo_ready.Add(1)
	globales.Semaforo_exec.Add(1)

	globales.MutexNew.Lock()
	globales.MutexReady.Lock()
	//globales.MutexProcesos.Lock()
	globales.MutexReadyAux.Lock()
	globales.MutexBlocked.Lock()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Planificación pausada"))
}

func Interfaz_Conexion(w http.ResponseWriter, r *http.Request) {
	decode := json.NewDecoder(r.Body)
	var interfaz *estructuras.Interfaz
	err := decode.Decode(&interfaz)
	if err != nil {
		log.Printf("error decodificando mensaje: %s", err.Error())
	}

	log.Println("Interfaz: ", interfaz.Nombre)
	log.Println("Tipo: ", interfaz.Tipo)
	log.Println("Ip: ", interfaz.Ip)
	log.Println("Puerto: ", interfaz.Port)

	confirmacion, err := interfaces.SetInterfaz(interfaz)

	if err != nil {
		log.Printf("error seteando interfaz: %s", err.Error())
	}

	if confirmacion {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}

}
