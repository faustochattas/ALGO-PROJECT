package globales

import (
	"sync"

	"github.com/sisoputnfrba/tp-golang/utils/estructuras"
)

//var Pid int = 0 //Process ID

var ListaNew []*estructuras.Pcb = make([]*estructuras.Pcb, 0)
var ListaReady []*estructuras.Pcb = make([]*estructuras.Pcb, 0)
var ListaReadyAux []*estructuras.Pcb = make([]*estructuras.Pcb, 0)

var ListaExecute []*estructuras.Pcb = make([]*estructuras.Pcb, 0)
var ListaBlocked []*estructuras.Pcb = make([]*estructuras.Pcb, 0)
var ListaExit []*estructuras.Pcb = make([]*estructuras.Pcb, 0)

///////SEMAFOROS

var Semaforo_new sync.WaitGroup
var Semaforo_ready sync.WaitGroup
var Semaforo_exec sync.WaitGroup

//////////////////////////////////

var HayPCBsEnNew = make(chan int)

var Pcb_por_pid = make(map[int]*estructuras.Pcb)

var CanalParaEjecutar = make(chan int, 1)

var Canal_Finalizar_Proceso = make(chan *estructuras.Pcb, 1)

var ListaInterfacesGenericas = make(map[string]int)
var ListaInterfacesGeneral []*Io = make([]*Io, 0)

var MutexNew sync.Mutex
var MutexReady sync.Mutex
var MutexProcesos sync.Mutex
var MutexReadyAux sync.Mutex

var MutexBlocked sync.Mutex

//Recursos

var ListaBlockedRecursos = make(map[string][]*estructuras.Pcb) // [recurso]pcb
var Recursos_disponibles []*estructuras.Recurso = make([]*estructuras.Recurso, 0)
var RecursosUsados = make(map[int][]*estructuras.Recurso) // [pid]recurso

//var MutexBlocked sync.Mutex

type Io struct {
	Nombre string
	Tipo   string
	Ip     string
	Port   int
}
