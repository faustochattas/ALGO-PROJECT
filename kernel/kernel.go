package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/sisoputnfrba/tp-golang/kernel/apis"
	"github.com/sisoputnfrba/tp-golang/kernel/config"
	"github.com/sisoputnfrba/tp-golang/kernel/globales"
	"github.com/sisoputnfrba/tp-golang/kernel/planificador"
	"github.com/sisoputnfrba/tp-golang/utils/cliente"
	"github.com/sisoputnfrba/tp-golang/utils/estructuras"
	"github.com/sisoputnfrba/tp-golang/utils/servidor"
)

func main() {

	directorioActual, erro := os.Getwd()
	if erro != nil {
		fmt.Println("Error al obtener el directorio actual:", erro)
		return
	}

	cliente.ConfigurarLogger("kernel", directorioActual)
	log.Println("Soy un log")

	// Usar la ruta del archivo de configuración
	fmt.Println("Ruta del archivo de configuración:", directorioActual)

	path_json := directorioActual + "/config.json"

	config.Kernel = config.IniciarConfiguracion(path_json) //path hardcodeado
	// validar que la config este cargada correctamente

	if config.Kernel == nil {
		log.Println("Error al cargar la configuración")
		return
	}

	cargarRecursosDisponibles()

	sem_multiprogramacion := make(chan estructuras.Pcb, config.Kernel.Multiprogramming)

	go planificador.IniciarPlanificadorLargoPlazo(sem_multiprogramacion)

	go planificador.PlanificadorCortoPlazo(sem_multiprogramacion)

	go planificador.Finalizador_de_procesos(sem_multiprogramacion)

	mux := http.NewServeMux()

	mux.HandleFunc("/mensaje", servidor.RecibirMensaje)

	//EndPoints Para el Checkpoint 1

	mux.HandleFunc("/process", apis.HandleRequestIniciar_Listar) //Implementacion de iniciar proceso Probado o listar procesos Probado

	mux.HandleFunc("/process/", apis.HandleRequestTerminar_VerEstado) //Implementacion de terminar proceso Probado o ver estado de proceso Probado

	mux.HandleFunc("/plani", apis.HandleIniciar_Pausar_Planificacion) //Implementacion de reanudar planificacion Probado o pausar planificacion Probado

	mux.HandleFunc("/interfaz", apis.Interfaz_Conexion) //Interfaz IO

	puerto := strconv.Itoa(config.Kernel.Port)
	puerto = ":" + puerto

	log.Print("Escuchando en puerto: ", puerto)

	//panic("no implementado!")
	err := http.ListenAndServe(puerto, mux)
	if err != nil {
		panic(err)
	}

}

func cargarRecursosDisponibles() {

	i := 0
	for _, recurso := range config.Kernel.Resources {

		instancia := config.Kernel.Resources_instances[i]

		globales.Recursos_disponibles = append(globales.Recursos_disponibles, &estructuras.Recurso{Nombre: recurso, Instancia: instancia})

		i++

	}
}
