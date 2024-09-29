package interfaces

import (
	"log"

	"github.com/sisoputnfrba/tp-golang/kernel/globales"
	"github.com/sisoputnfrba/tp-golang/utils/estructuras"
)

func SetInterfaz(interfaz *estructuras.Interfaz) (bool, error) {

	//revisar que el nombre no este en la lista general
	for _, io := range globales.ListaInterfacesGeneral {
		if io.Nombre == interfaz.Nombre {
			log.Println("Ya existe una interfaz con el nombre: ", interfaz.Nombre)
			return false, nil
		}
	}

	if interfaz.Tipo == "GENERICA" {
		globales.ListaInterfacesGenericas[interfaz.Nombre] = interfaz.UnidadesTrabajo
	}

	Io := new(globales.Io)
	Io.Nombre = interfaz.Nombre
	Io.Tipo = interfaz.Tipo
	Io.Ip = interfaz.Ip
	Io.Port = interfaz.Port

	globales.ListaInterfacesGeneral = append(globales.ListaInterfacesGeneral, Io)

	return true, nil

}
