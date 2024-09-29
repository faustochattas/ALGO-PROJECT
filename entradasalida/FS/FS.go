package FS

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sisoputnfrba/tp-golang/entradasalida/config"
	"github.com/sisoputnfrba/tp-golang/utils/estructuras"
)

func Crear_archivo_JSON(filePath string, metadata estructuras.Metadata) error {
	// Crear el archivo de datos
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Convertir la metadata a JSON
	metaDataBytes, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	// Escribir el JSON en el archivo
	_, err = file.Write(metaDataBytes)
	if err != nil {
		return err
	}

	return nil
}

func LeerArchivoJSON(filePath string) (*estructuras.Metadata, error) {
	// Leer el contenido del archivo
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Deserializar el contenido JSON en una estructura Metadata
	var metadata estructuras.Metadata
	err = json.Unmarshal(data, &metadata)
	if err != nil {
		return nil, err
	}

	return &metadata, nil
}

func Existe_archivo(filePath string) bool {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

func Crear_bitmap(filePath string, blockCount int) error {

	ruta := filepath.Join(filePath, "bitmap.dat")

	// Crear el directorio si no existe
	err := os.MkdirAll(filePath, os.ModePerm)
	if err != nil {
		fmt.Println("Error creating directory:", err)
		return err
	}

	if Existe_archivo(ruta) {
		print("El archivo ya existe")
		return nil
	}

	// Crear el archivo de bitmap
	file, err := os.Create(ruta)
	if err != nil {
		return err
	}
	defer file.Close()

	// Llenar el archivo con 0's según la cantidad de bloques especificada
	for i := 0; i < blockCount; i++ {
		_, err := file.Write([]byte{'0'})
		if err != nil {
			return err
		}
	}

	// Añadir un byte '\n' al final del archivo
	_, err = file.Write([]byte{'\n'})
	if err != nil {
		return err
	}

	return nil
}

func Crear_archivo_bloques(filePath string, blockSize int, blockCount int) error {

	ruta := filepath.Join(filePath, "bloques.dat")

	// Crear el directorio si no existe
	err := os.MkdirAll(filePath, os.ModePerm)
	if err != nil {
		fmt.Println("Error creating directory:", err)
		return err
	}

	if Existe_archivo(ruta) {
		print("El archivo ya existe")
		return nil
	}

	// Crear el archivo de bloques
	file, err := os.Create(ruta)
	if err != nil {
		return err
	}
	defer file.Close()

	// Crear un bloque con tamaño especificado terminado con '\n'
	block := strings.Repeat(" ", blockSize*blockCount)

	// Llenar el archivo con bloques
	_, err = file.WriteString(block)
	if err != nil {
		return err
	}

	return nil
}

func Leer_bitmap(filePath string) ([]byte, error) {
	// Abrir el archivo de bitmap
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var bitmapData []byte

	// Utilizar bufio.Reader para leer el archivo byte a byte
	reader := bufio.NewReader(file)
	for {
		byte, err := reader.ReadByte()
		if err != nil {
			return nil, err
		}
		if byte == '\n' {
			break
		}
		bitmapData = append(bitmapData, byte)
	}

	return bitmapData, nil
}

func Marcar_bit_libre_create(filePath string) (int, error) {

	ruta := filepath.Join(filePath, "bitmap.dat")

	// Leer el archivo de bitmap
	bitmapData, err := Leer_bitmap(ruta)
	if err != nil {
		return 0, err
	}

	posicion := 0

	// Encontrar el primer byte con valor 0 y cambiarlo a 1
	for i := 0; i < len(bitmapData); i++ {
		if bitmapData[i] == '0' {
			bitmapData[i] = '1'

			// Abrir el archivo de bitmap para escritura
			file, err := os.OpenFile(ruta, os.O_WRONLY, 0644)
			if err != nil {
				return -1, err
			}
			defer file.Close()

			// Escribir los cambios en el archivo
			_, err = file.WriteAt(bitmapData, 0)
			if err != nil {
				return -1, err
			}

			return posicion, nil
		}
		posicion++
	}

	return -1, fmt.Errorf("no free block found")
}

func Asignacion_mejorada(filePath string) (int, error) {
	ruta := filepath.Join(filePath, "bitmap.dat")

	// Leer el archivo de bitmap
	bitmapData, err := Leer_bitmap(ruta)
	if err != nil {
		return 0, err
	}

	var array_posiciones [][]int

	for i := 0; i < len(bitmapData); {
		var aux []int

		for {
			if i < len(bitmapData) && bitmapData[i] == '0' {
				aux = append(aux, i)
				i++
			} else {
				if len(aux) > 0 { // Añadir aux solo si contiene posiciones válidas
					array_posiciones = append(array_posiciones, aux)
				}
				if i < len(bitmapData) && bitmapData[i] != '0' {
					i++ // Avanzar i para evitar loop infinito
				}
				break
			}
		}
	}

	//Encontra el array de mayor longitud

	if len(array_posiciones) == 0 {

		log.Println("No hay bloques libres")
		return -1, nil
	}

	mayor_pos := 0

	mayor_long := 0

	for i := 0; i < len(array_posiciones); i++ {
		if len(array_posiciones[i]) >= mayor_long {
			mayor_long = len(array_posiciones[i])
			mayor_pos = i
		}
	}

	ubi := array_posiciones[mayor_pos]

	longitud := len(ubi)

	medio := (ubi[0] + ubi[longitud-1]) / 2

	bitmapData[medio] = '1'

	// Abrir el archivo de bitmap para escritura
	file, err := os.OpenFile(ruta, os.O_WRONLY, 0644)

	if err != nil {
		return -1, err
	}
	defer file.Close()

	// Escribir los cambios en el archivo

	_, err = file.WriteAt(bitmapData, 0)
	if err != nil {
		return -1, err
	}

	return medio, nil

}

func LeerBitmap(filename string, position int) (int, error) {
	// Leer el contenido del archivo
	content, err := os.ReadFile(filename)
	if err != nil {
		return 0, fmt.Errorf("error al leer el archivo: %v", err)
	}

	// Verificar que la posición sea válida
	if position < 0 || position >= len(content) {
		return 0, fmt.Errorf("posición inválida")
	}

	// Retornar el valor en la posición especificada
	return int(content[position]), nil
}

func EscribirBitmap(filename string, position int) error {
	// Leer el contenido del archivo
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("error al leer el archivo: %v", err)
	}

	// Verificar que la posición sea válida
	if position < 0 || position >= len(content) {
		return fmt.Errorf("posición inválida")
	}

	// Cambiar el bit en la posición especificada
	if content[position] == '1' {
		content[position] = '0'
	} else if content[position] == '0' {
		content[position] = '1'
	} else {
		return fmt.Errorf("el archivo contiene caracteres no válidos, solo se permiten '0' y '1'")
	}

	// Escribir el contenido modificado de nuevo en el archivo
	err = os.WriteFile(filename, content, 0644)
	if err != nil {
		return fmt.Errorf("error al escribir en el archivo: %v", err)
	}

	return nil
}

// Dada una ruta , bloque inicial , puntero de archivo y los bloques a escribir , escribe los bloques en el archivo (FS_WRITE)
func Escribir_bloques(filename string, bloque_inicial int, punteroArchivo int, blocks []byte) error {
	// Abre el archivo en modo lectura/escritura
	file, err := os.OpenFile(filename, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error abriendo el archivo: %w", err)
	}
	defer file.Close()

	cantidad_bytes := len(blocks)

	limite := config.Io.Dialfs_block_size //block size

	for {

		posicion := bloque_inicial*(limite) + punteroArchivo

		if cantidad_bytes > (limite - punteroArchivo) {

			// Mueve el puntero del archivo a la posición inicial en bytes
			_, err = file.Seek(int64(posicion), 0)
			if err != nil {
				return fmt.Errorf("error moviéndose a la posición inicial: %w", err)
			}

			// Escribe los bloques en el archivo
			_, err = file.Write(blocks[:(limite - punteroArchivo)])
			if err != nil {
				return fmt.Errorf("error escribiendo los bloques: %w", err)
			}

			cantidad_bytes -= (limite - punteroArchivo)
			bloque_inicial++
			blocks = blocks[(limite - punteroArchivo):]

			punteroArchivo = 0

		} else {

			// Mueve el puntero del archivo a la posición inicial en bytes
			_, err = file.Seek(int64(posicion), 0)
			if err != nil {
				return fmt.Errorf("error moviéndose a la posición inicial: %w", err)
			}

			// Escribe los bloques en el archivo
			_, err = file.Write(blocks)
			if err != nil {
				return fmt.Errorf("error escribiendo los bloques: %w", err)
			}

			break

		}

	}

	return nil
}

func Truncar_bloques(filename string, tamanio_Trunc int) {

	ruta := filepath.Join(config.Io.Dialfs_path, filename)

	metadata, err := LeerArchivoJSON(ruta)
	if err != nil {
		fmt.Println("Error leyendo el archivo:", err)
		return
	}

	bloque_inicial := metadata.InitialBlock
	size := metadata.Size

	if tamanio_Trunc == size {
		return
	}

	var lectura_auxiliar []byte

	if size != 0 {
		lectura, err := Leer_bloques(config.Io.Dialfs_path+"bloques.dat", bloque_inicial, 0, size)
		if err != nil {
			fmt.Println("Error leyendo el archivo:", err)
			return
		}

		lectura_auxiliar = append(lectura_auxiliar, lectura...)

	}

	if tamanio_Trunc > size {

		//Analizar contiguedad de los bloques
		var distancia_bloque int

		if size == 0 { //bloques config.Io.Dialfs_block_sizebytes -> size 7 bytes ;
			distancia_bloque = config.Io.Dialfs_block_size //tamaño de bloque
		} else {
			distancia_bloque = size % config.Io.Dialfs_block_size // 1
		}

		diferencia := tamanio_Trunc - size // 4 restar distancia bloque

		if diferencia <= distancia_bloque {
			//No se necesita agregar bloques
			//cambiar su metadata
			log.Println("Asigne sin agregar bloques")
			return
		}

		//Se necesita agregar bloques

		//Calcular la cantidad de bloques a agregar

		//Si llego hasta aca, por lo menos necesita un bloque

		count := diferencia - distancia_bloque

		var cantidad_bloques int
		if size != 0 {
			if count%config.Io.Dialfs_block_size == 0 {
				cantidad_bloques = (count / config.Io.Dialfs_block_size)

			} else {
				cantidad_bloques = (count / config.Io.Dialfs_block_size) + 1 // 1

			}
		} else {

			if count%config.Io.Dialfs_block_size == 0 {

				cantidad_bloques = (count / config.Io.Dialfs_block_size)

			} else {
				cantidad_bloques = (count / config.Io.Dialfs_block_size) + 1
			}
		}

		//bloques actuales

		var bloques_actuales int

		if size == 0 {
			bloques_actuales = 1

		} else {
			if size%config.Io.Dialfs_block_size == 0 {
				bloques_actuales = (size / config.Io.Dialfs_block_size) - 1

			} else {
				bloques_actuales = (size / config.Io.Dialfs_block_size) + 1
			}
		}

		//Leer el bitmap

		bitmap, err := Leer_bitmap(config.Io.Dialfs_path + "bitmap.dat")
		if err != nil {
			fmt.Println("Error leyendo el archivo:", err)
			return
		}

		//Buscar bloques libres

		posicion_bitmap := bloque_inicial + bloques_actuales

		/*
			bloques_libres := 0
			fmt.Println("Posicion bitmap: ", posicion_bitmap)

			for i := posicion_bitmap + 1; i < len(bitmap); i++ {
				if bitmap[i] == '0' {
					bloques_libres++
				}
			}
		*/
		bloques_libres_contiguos := 0

		for i := posicion_bitmap; i < len(bitmap); i++ {
			if bitmap[i] == '0' {
				bloques_libres_contiguos++
			} else {
				break
			}
		}

		//fmt.Println("Bloques libres totales: ", bloques_libres)

		if bloques_libres_contiguos < cantidad_bloques {
			log.Println("No hay bloques suficientes contiguos")
			log.Println("COMPACTAR")

			Marcar_posicion_en_cero(posicion_bitmap) //hacer un for si borro mas de uno

			Compactar(filename)

			time.Sleep(time.Duration(config.Io.Retraso_compactacion) * time.Millisecond) // Delay

			var array []int

			for i := 0; i < cantidad_bloques+bloques_actuales; i++ { // 2

				pos, _ := Marcar_bit_libre_create(config.Io.Dialfs_path)

				array = append(array, pos)
			}

			if size != 0 {
				Escribir_bloques(config.Io.Dialfs_path+"bloques.dat", array[0], 0, lectura_auxiliar)
			}

			Borrar_archivo(filepath.Join(config.Io.Dialfs_path, filename))

			Crear_archivo_JSON(filepath.Join(config.Io.Dialfs_path, filename), estructuras.Metadata{InitialBlock: array[0], Size: tamanio_Trunc})

			//asignas bloques contiguos

			return //Retorna Falso y compactar o compactar en esta misma funcion
		}

		for i := 0; i < cantidad_bloques; i++ {

			Marcar_posicion_en_uno(bloque_inicial + 1 + i) //Marcar apartir de su bloque inicial
		}

		Borrar_archivo(filepath.Join(config.Io.Dialfs_path, filename))

		Crear_archivo_JSON(filepath.Join(config.Io.Dialfs_path, filename), estructuras.Metadata{InitialBlock: bloque_inicial, Size: tamanio_Trunc})

	} else {

		if size == 0 {
			fmt.Println("Error NO se puede disminuir el tamaño del archivo")
		} else {
			var ultimoBloque int
			if size%config.Io.Dialfs_block_size == 0 {
				ultimoBloque = bloque_inicial + (size / config.Io.Dialfs_block_size) - 1
			} else {
				ultimoBloque = bloque_inicial + (size / config.Io.Dialfs_block_size)
			}

			diferencia := size - tamanio_Trunc

			if size <= 10 { //tamaño de bloque

				Borrar_archivo(filepath.Join(config.Io.Dialfs_path, filename))

				Crear_archivo_JSON(filepath.Join(config.Io.Dialfs_path, filename), estructuras.Metadata{InitialBlock: bloque_inicial, Size: tamanio_Trunc})
			} else {
				cantidadBloquesBorrables := diferencia / config.Io.Dialfs_block_size
				if diferencia%config.Io.Dialfs_block_size == 0 && cantidadBloquesBorrables == (size/config.Io.Dialfs_block_size) {
					cantidadBloquesBorrables = (diferencia / config.Io.Dialfs_block_size) - 1
				}

				for i := ultimoBloque; i > ultimoBloque-cantidadBloquesBorrables; i-- {

					Marcar_posicion_en_cero(i)
				}

				Borrar_archivo(filepath.Join(config.Io.Dialfs_path, filename))

				Crear_archivo_JSON(filepath.Join(config.Io.Dialfs_path, filename), estructuras.Metadata{InitialBlock: bloque_inicial, Size: tamanio_Trunc})

			}

		}
		//Checkear si hay que borrar bloques

		return
	}

}

func Marcar_posicion_en_cero(posicion int) {

	dir := config.Io.Dialfs_path //config.path

	ruta := filepath.Join(dir, "bitmap.dat")

	// Leer el archivo de bitmap
	bitmap, err := Leer_bitmap(ruta)
	if err != nil {
		return
	}

	bitmap[posicion] = '0'

	// Abrir el archivo de bitmap para escritura
	file, err := os.OpenFile(config.Io.Dialfs_path+"bitmap.dat", os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer file.Close()

	// Escribir los cambios en el archivo
	_, err = file.WriteAt(bitmap, 0)
	if err != nil {
		return
	}

}

func Marcar_posicion_en_uno(posicion int) {

	dir := config.Io.Dialfs_path //config.path

	ruta := filepath.Join(dir, "bitmap.dat")

	// Leer el archivo de bitmap
	bitmap, err := Leer_bitmap(ruta)
	if err != nil {
		return
	}

	bitmap[posicion] = '1'

	// Abrir el archivo de bitmap para escritura
	file, err := os.OpenFile(config.Io.Dialfs_path+"bitmap.dat", os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer file.Close()

	// Escribir los cambios en el archivo
	_, err = file.WriteAt(bitmap, 0)
	if err != nil {
		return
	}

}

func Compactar(filename string) {

	// Archivos a mover

	dir := config.Io.Dialfs_path //config.path

	files, err := LeerArchivos(dir, filename)
	if err != nil {
		log.Fatal(err)
	}

	ruta := filepath.Join(dir, "bitmap.dat")

	// Leer el archivo de bitmap
	bitmapData, err := Leer_bitmap(ruta)
	if err != nil {
		return
	}

	posicion := 0

	var bitmap_nuevo []byte

	var lista_bytes = make(map[string][]byte)

	// Leer los contenidos - 10000000000
	for _, file := range files {

		ruta := filepath.Join(config.Io.Dialfs_path, file)

		metadata, err := LeerArchivoJSON(ruta)
		if err != nil {
			fmt.Println("Error leyendo el archivo:", err)
			return
		}

		ruta_bloques := filepath.Join(dir, "bloques.dat")

		bloque_inicial := metadata.InitialBlock
		size := metadata.Size

		lectura, err := Leer_bloques(ruta_bloques, bloque_inicial, 0, size)

		if err != nil {
			fmt.Println("Error leyendo el archivo:", err)
			return
		}

		lista_bytes[file] = lectura
	}

	for _, file := range files {

		a_escribir := lista_bytes[file]

		longitud := len(a_escribir)

		cantidad_bloques := longitud / config.Io.Dialfs_block_size // config.Io.Dialfs_block_sizees igual a block size

		if longitud%config.Io.Dialfs_block_size != 0 {
			cantidad_bloques++
		}

		ruta_bloques := filepath.Join(dir, "bloques.dat")

		Escribir_bloques(ruta_bloques, posicion, 0, a_escribir)

		Borrar_archivo(filepath.Join(dir, file))

		Crear_archivo_JSON(filepath.Join(dir, file), estructuras.Metadata{InitialBlock: posicion, Size: longitud})

		posicion += cantidad_bloques //posicion sig bloque

		//cambiar metadata

		for i := 0; i < cantidad_bloques; i++ {
			bitmap_nuevo = append(bitmap_nuevo, '1')
		}

	}

	cantidad := len(bitmapData) - len(bitmap_nuevo)

	for i := 0; i < cantidad; i++ {
		bitmap_nuevo = append(bitmap_nuevo, '0')
	}

	rutax := filepath.Join(dir, "bitmap.dat")

	// Abrir el archivo de bitmap para escritura
	file, err := os.OpenFile(rutax, os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer file.Close()

	// Escribir los cambios en el archivo
	_, err = file.WriteAt(bitmap_nuevo, 0)
	if err != nil {
		return
	}

}

func LeerArchivos(dir string, otro string) ([]string, error) {
	var files []string

	// Abre el directorio
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Lee los nombres de los archivos en el directorio
	fileInfo, err := f.Readdir(-1)
	if err != nil {
		return nil, err
	}

	// Agrega los nombres de los archivos a la lista
	for _, file := range fileInfo {
		if !file.IsDir() {
			if file.Name() != "bitmap.dat" && file.Name() != "bloques.dat" && file.Name() != otro {
				files = append(files, file.Name())
			}
		}
	}

	return files, nil
}

func Leer_bloques(filename string, bloque_inicial int, punteroArchivo int, cantidad_bytes_a_leer int) ([]byte, error) {
	// Abre el archivo en modo lectura/escritura
	file, err := os.OpenFile(filename, os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("error abriendo el archivo: %w", err)
	}
	defer file.Close()

	limite := config.Io.Dialfs_block_size //block size

	var lectura []byte

	for {

		posicion := bloque_inicial*(limite) + punteroArchivo

		diferencia := limite - punteroArchivo

		if cantidad_bytes_a_leer < diferencia {
			diferencia = cantidad_bytes_a_leer
		}

		if cantidad_bytes_a_leer > (limite - punteroArchivo) {

			// Mueve el puntero del archivo a la posición inicial en bytes
			_, err = file.Seek(int64(posicion), 0)
			if err != nil {
				return nil, fmt.Errorf("error moviéndose a la posición inicial: %w", err)
			}

			buffer := make([]byte, diferencia)

			// Escribe los bloques en el archivo
			_, err = file.Read(buffer)
			if err != nil {
				return nil, fmt.Errorf("error escribiendo los bloques: %w", err)
			}

			cantidad_bytes_a_leer -= (limite - punteroArchivo)
			bloque_inicial++
			punteroArchivo = 0

			lectura = append(lectura, buffer...)

		} else {

			// Mueve el puntero del archivo a la posición inicial en bytes
			_, err = file.Seek(int64(posicion), 0)
			if err != nil {
				return nil, fmt.Errorf("error moviéndose a la posición inicial: %w", err)
			}

			buffer := make([]byte, diferencia)

			// Escribe los bloques en el archivo
			_, err = file.Read(buffer)
			if err != nil {
				return nil, fmt.Errorf("error escribiendo los bloques: %w", err)
			}

			lectura = append(lectura, buffer...)

			break

		}

	}

	return lectura, nil
}

func Borrar_archivo(filePath string) error {
	// Crear el archivo de datos
	err := os.Remove(filePath)
	if err != nil {
		return err
	}

	return nil
}
