package estructuras

type Pcb struct {
	Pid                int
	Program_counter    int
	QuantumRestante    int
	AX                 uint8
	BX                 uint8
	CX                 uint8
	DX                 uint8
	EAX                uint32
	EBX                uint32
	ECX                uint32
	EDX                uint32
	SI                 uint32
	DI                 uint32
	Estado             string
	Recursos_asignados []*Recurso
}

type Pcb_contexto struct {
	Pid             int     `json:"Pid"`
	Program_counter int     `json:"Program_counter"`
	AX              uint8   `json:"AX"`
	BX              uint8   `json:"BX"`
	CX              uint8   `json:"CX"`
	DX              uint8   `json:"DX"`
	EAX             uint32  `json:"EAX"`
	EBX             uint32  `json:"EBX"`
	ECX             uint32  `json:"ECX"`
	EDX             uint32  `json:"EDX"`
	SI              uint32  `json:"SI"`
	DI              uint32  `json:"DI"`
	Bloqueo         *Motivo `json:"Motivo"`
}

type Motivo struct {
	Motivo              string              `json:"Motivo"`
	Parametro1          string              `json:"Parametro1"`
	Parametro2          string              `json:"Parametro2"`
	Parametro3          string              `json:"Parametro3"`
	Direcciones_Fisicas []*Direccion_fisica `json:"Direcciones_Fisicas"`
}

type Direccion_fisica struct {
	Pagina         int `json:"Pagina"`
	Desplazamiento int `json:"Desplazamiento"`
	Cantidad_bytes int `json:"Cantidad_bytes"`
}

type Pedir_instruccion_memoria struct {
	Pid             int `json:"Pid"`
	Program_counter int `json:"Program_counter"`
}

type Path_proceso struct {
	Pid          int    `json:"Pid"`
	Path_proceso string `json:"Path_proceso"`
}

type Instruccion_memoria struct {
	Instruccion string `json:"Instruccion"`
}

type Registros struct {
	Program_counter int
	AX              uint8
	BX              uint8
	CX              uint8
	DX              uint8
	EAX             uint32
	EBX             uint32
	ECX             uint32
	EDX             uint32
	SI              uint32
	DI              uint32
}

type Interfaz struct {
	Nombre          string
	Tipo            string `json:"Type"`
	UnidadesTrabajo int    `json:"Unit_work_time"`
	Ip              string `json:"Ip"`
	Port            int    `json:"Port"`
}

type Resize struct {
	Pid    int `json:"Pid"`
	Ajuste int `json:"Ajuste"`
}

type Pedido_lectura_memoria struct {
	Pid                 int `json:"Pid"`
	Desplazamiento      int `json:"Desplazamiento"`
	Marco               int `json:"Marco"`
	Pagina              int `json:"Pagina"`
	Cantidad_bytes_leer int `json:"Cantidad_bytes_leer"`
}

type Pedido_escritura_memoria struct {
	Pid                     int   `json:"Pid"`
	Desplazamiento          int   `json:"Desplazamiento"`
	Marco                   int   `json:"Marco"`
	Pagina                  int   `json:"Pagina"`
	Cantidad_bytes_escribir int   `json:"Cantidad_bytes_escribir"`
	Valor                   []int `json:"Valor"`
}

type Pedido_borrar_memoria struct {
	Pid int `json:"Pid"`
}

type Respuesta_lectura_memoria struct {
	Valor []int `json:"Valor"`
}

type Respuesta_reservar_paginas struct {
	Estado string `json:"Estado"`
}

type TLB_miss struct {
	Pid           int `json:"Pid"`
	Numero_pagina int `json:"Numero_pagina"`
	Marco         int `json:"Marco"`
}

type Ejecucion_interfaz_generica struct {
	Pid    int `json:"Pid"`
	Tiempo int `json:"Tiempo"`
}

type Ejecucion_interfaz_READ struct {
	Pid                 int                 `json:"Pid"`
	Direcciones_fisicas []*Direccion_fisica `json:"Direccion_fisica"`
}

type Ejecucion_interfaz_FS struct {
	Pid                 int                 `json:"Pid"`
	Nombre_Archivo      string              `json:"Nombre_Archivo"`
	Instruccion         string              `json:"Instruccion"`
	Tamanio_Trunc       int                 `json:"Tamanio_Trunc"`
	Tamanio             int                 `json:"Tamanio"`
	Puntero             int                 `json:"puntero"`
	Direcciones_fisicas []*Direccion_fisica `json:"Direccion_fisica"`
}

type Recurso struct {
	Nombre    string
	Instancia int
}

type Metadata struct {
	InitialBlock int `json:"initial_block"`
	Size         int `json:"size"`
}
