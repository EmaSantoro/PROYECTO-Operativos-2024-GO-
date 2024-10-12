package utils

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/sisoputnfrba/tp-golang/memoria/globals"
)

/*-------------------- ESTRUCTURAS --------------------*/
type PCB struct { //NO ES LA MISMA PCB QUE TIENE KERNEL DIGAMOS ES UNA PROPIA DE MEMORIA
	Pid   int
	Base  uint32
	Limit uint32
}

type InstructionResponse struct {
	Instruction string `json:"instruction"`
}

type NewContext struct {
	PCB struct {
		Pid   int    `json:"pid"`
		Base  uint32 `json:"base"`
		Limit uint32 `json:"limit"`
	}
	estructuraHilo struct {
		Pid int    `json:"pid"`
		Tid int    `json:"tid"`
		AX  uint32 `json:"AX"`
		BX  uint32 `json:"BX"`
		CX  uint32 `json:"CX"`
		DX  uint32 `json:"DX"`
		EX  uint32 `json:"EX"`
		FX  uint32 `json:"FX"`
		GX  uint32 `json:"GX"`
		HX  uint32 `json:"HX"`
		PC  uint32 `json:"PC"`
	}
}

//estado de las particiones ocupada/libre
// var particiones = MemoriaConfig.Particiones //vector de particiones, aca tengo los tamaños en int

type Process struct {
	Size int `json:"size"`
	Pid  int `json:"pid"`
}

type Thread struct {
	Pid  int    `json:"pid"`
	Tid  int    `json:"tid"`
	Path string `json:"path"`
}

type estructuraHilo struct {
	Pid int
	Tid int
	AX  uint32
	BX  uint32
	CX  uint32
	DX  uint32
	EX  uint32
	FX  uint32
	GX  uint32
	HX  uint32
	PC  uint32
}

type KernelExeReq struct {
	Pid int `json:"pid"` // ver cuales son los keys usados en Kernel
	Tid int `json:"tid"`
}

type InstructionReq struct {
	Pid int `json:"pid"`
	Tid int `json:"tid"`
	Pc  int `json:"pc"`
}

/*-------------------- VAR GLOBALES --------------------*/
var esquemaMemoria string
var particiones []int
var algoritmoBusqueda string
var IpCpu string
var PuertoCpu int
var MemoriaConfig *globals.Config

// Mapa anidado que almacena los contextos de ejecución
var mapPCBPorTCB = make(map[PCB]map[estructuraHilo][]string) //ESTE ES EL PRINCIPAL DIGAMOS
var mapParticiones []bool

// // Mapa anidado que almacena los contextos de ejecución
// var mapPCBPorTCB = make(map[PCB]map[TCB][]string)

/*---------------------- FUNCIONES ----------------------*/
//	INICIAR CONFIGURACION Y LOGGERS

func IniciarConfiguracion(filePath string) *globals.Config {
	var config *globals.Config
	configFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)

	return config
}

func ConfigurarLogger() {
	logFile, err := os.OpenFile("tp.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
}

// INICIAR MODULO
func init() {

	MemoriaConfig = IniciarConfiguracion("memoria/configsMemoria/config.json")

	if MemoriaConfig != nil {
		particiones = MemoriaConfig.Particiones
		globals.MemoriaUsuario = make([]byte, MemoriaConfig.Tamanio_Memoria)
		esquemaMemoria = MemoriaConfig.EsquemaMemoria
		algoritmoBusqueda = MemoriaConfig.AlgoritmoBusqueda
		IpCpu = MemoriaConfig.IpCpu
		PuertoCpu = MemoriaConfig.PuertoCpu

	} else {
		log.Fatal("ClientConfig is not initialized")
	}
	log.Printf("%d", particiones)
}

///////////////////////////////////////////////////////////////////////////////

///--------------------------------------------GET INSTRUCTION---------------------------------------------

func GetInstruction(w http.ResponseWriter, r *http.Request) {

	var instructionReq InstructionReq
	log.Printf("Entre a get instruction")
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&instructionReq)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	time.Sleep(time.Duration(MemoriaConfig.Delay_Respuesta) * time.Millisecond)

	for pcb, tidMap := range mapPCBPorTCB {
		log.Printf("tengo guardado: %v", tidMap)
		log.Printf("IntructionReq.Pid %d", instructionReq.Pid)
		if pcb.Pid == instructionReq.Pid {
			log.Printf("me llego: %d", instructionReq.Tid)
			log.Printf("tengo guardado: %v", tidMap)
			for tcb, instrucciones := range tidMap {

				if tcb.Tid == instructionReq.Tid {
					if instructionReq.Pc >= 0 && instructionReq.Pc < len(instrucciones) {

						instruccion := instrucciones[instructionReq.Pc]

						instructionResponse := InstructionResponse{
							Instruction: instruccion,
						}

						// Log de obtencion de instruccion
						log.Printf("## Obtener instrucción - (PID:TID) - (%d:%d) - Instrucción: %s", instructionReq.Pid, instructionReq.Tid, instruccion)

						// Envio la respuesta en formato JSON
						json.NewEncoder(w).Encode(instructionResponse)
						w.Write([]byte(instruccion)) // Escribo la instrucción no se cual usar
						return
					} else {
						http.Error(w, "PC fuera del rango de instrucciones", http.StatusBadRequest)
						fmt.Println("PC fuera del rango de instrucciones")
						return
					}
				}
			}
			http.Error(w, "No se encontro el TID", http.StatusNotFound)
			fmt.Println("No se encontró el TID")
			return
		}
	}
	http.Error(w, "No se encontro el PID", http.StatusNotFound)
	fmt.Println("No se encontró el PID")
}

// ------------------------------------ GET EXECUTION CONTEXT ---------------------------------------------
type GetExecutionContextResponse struct {
	Pcb PCB            `json:"pcb"`
	Tcb estructuraHilo `json:"tcb"`
}

func GetExecutionContext(w http.ResponseWriter, r *http.Request) {

	var solicitud KernelExeReq
	// queryParams := r.URL.Query()
	// pid, _ := strconv.Atoi(queryParams.Get("pid"))
	// tid, _ := strconv.Atoi(queryParams.Get("tid"))

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&solicitud)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("PCB : %d TID : %d - me llegaron estos valores", solicitud.Pid, solicitud.Tid)

	time.Sleep(time.Duration(MemoriaConfig.Delay_Respuesta) * time.Millisecond)
	var respuesta GetExecutionContextResponse
	for pcb, tidMap := range mapPCBPorTCB {
		if pcb.Pid == solicitud.Pid {
			for tcb := range tidMap {
				if tcb.Tid == solicitud.Tid {
					respuesta.Pcb = pcb
					respuesta.Tcb = tcb
					log.Printf("Pid %d y tid %d enontradas", pcb.Pid, tcb.Tid)
					respuestaJson, err := json.Marshal(respuesta)
					if err != nil {
						http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
					}
					log.Printf("Respuetsa %v", respuesta)
					w.WriteHeader(http.StatusOK)
					w.Write(respuestaJson)
					log.Printf("## Contexto <Solicitado> - (PID:TID) - (%d:%d)", solicitud.Pid, solicitud.Tid)
					/*executionContext := struct {
						PCB
						estructuraHilo
					}{
						PCB:            pcb,
						estructuraHilo: tcb,
					}
					log.Printf("Envio pid %d y tid %d", pcb.Pid, tcb.Tid)

					// Log de obtener contexto de ejecucion
					log.Printf("## Contexto <Solicitado> - (PID:TID) - (%d:%d)", solicitud.Pid, solicitud.Tid)

					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(executionContext)
					*/
					return
				}
			}
			http.Error(w, "No se encontro el TID", http.StatusNotFound)
			return
		}
	}
	http.Error(w, "No se encontro el PID", http.StatusNotFound)
}

//-------------------------------- UPDATE EXECUTION CONTEXT-----------------------------------------------

func UpdateExecutionContext(w http.ResponseWriter, r *http.Request) {
	// queryParams := r.URL.Query()
	// pid, _ := strconv.Atoi(queryParams.Get("pid")) //esto me parece que no va
	// tid, _ := strconv.Atoi(queryParams.Get("tid")) //esto tampoco
	log.Printf("Entra a actualziar contexto")
	var actualizadoContexto NewContext

	time.Sleep(time.Duration(MemoriaConfig.Delay_Respuesta) * time.Millisecond)

	if err := json.NewDecoder(r.Body).Decode(&actualizadoContexto); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("Respuesta codificara PID = %d , TID = %d", actualizadoContexto.PCB.Pid, actualizadoContexto.estructuraHilo.Tid)
	log.Printf("MAP PCB x TCB = %v", mapPCBPorTCB)
	for pcb, tidMap := range mapPCBPorTCB {
		if pcb.Pid == actualizadoContexto.PCB.Pid {
			log.Printf("PID actualizar : %d", pcb.Pid)
			for tcb := range tidMap {
				log.Printf("TID actualizar : %d", tcb.Tid)
				if tcb.Tid == actualizadoContexto.estructuraHilo.Tid {

					pcb.Base = actualizadoContexto.PCB.Base
					pcb.Limit = actualizadoContexto.PCB.Limit
					tcb.AX = uint32(actualizadoContexto.estructuraHilo.AX)
					tcb.BX = uint32(actualizadoContexto.estructuraHilo.BX)
					tcb.CX = uint32(actualizadoContexto.estructuraHilo.CX)
					tcb.DX = uint32(actualizadoContexto.estructuraHilo.DX)
					tcb.EX = uint32(actualizadoContexto.estructuraHilo.EX)
					tcb.FX = uint32(actualizadoContexto.estructuraHilo.FX)
					tcb.GX = uint32(actualizadoContexto.estructuraHilo.GX)
					tcb.HX = uint32(actualizadoContexto.estructuraHilo.HX)
					tcb.PC = uint32(actualizadoContexto.estructuraHilo.PC)

					//CREO QUE NO SE ESTA ACTUALIZANDO EL MAPA ANIDADO
					tidMap[tcb] = tidMap[tcb]
					mapPCBPorTCB[pcb] = tidMap

					// Log de obtener contexto de ejecucion
					log.Printf("## Contexto <Solicitado> - (PID:TID) - (%d:%d)", actualizadoContexto.PCB.Pid, actualizadoContexto.estructuraHilo.Tid)

					w.WriteHeader(http.StatusOK)
					w.Write([]byte("contexto de ejecucion ha sido actualizado"))
					return
				}
			}
			http.Error(w, "TID no ha sido encontrado", http.StatusNotFound)
			return
		}
	}
	http.Error(w, "PID no ha sido encontrado", http.StatusNotFound)
}

//-----------------------------------------CREATE PROCESS-------------------------------------------

func CreateProcess(w http.ResponseWriter, r *http.Request) { //recibe la pid y el size
	var process Process
	var limitEnInt int
	time.Sleep(time.Duration(MemoriaConfig.Delay_Respuesta) * time.Millisecond)

	if err := json.NewDecoder(r.Body).Decode(&process); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("Pid: %d", process.Pid)
	pcb := PCB{ //creo la estructura necesaria
		Pid:   process.Pid,
		Base:  0,
		Limit: 0,
	}

	if esquemaMemoria == "FIJAS" {
		mapParticiones = make([]bool, len(particiones))
		numeroDeParticion := asignarPorAlgoritmo(algoritmoBusqueda, process.Size) //asigno por algoritmo
		if numeroDeParticion == -1 {
			http.Error(w, "No hay espacio en la memoria", http.StatusConflict)
			return
		}

		//BASE
		var baseEnInt int
		pcb.Base = 0
		for i := 0; i < numeroDeParticion; i++ {
			baseEnInt += particiones[i] //tengo que ver tema int y uint32
		}
		pcb.Base = uint32(baseEnInt)
		//LIMIT
		limitEnInt = baseEnInt + particiones[numeroDeParticion] - 1
		pcb.Limit = uint32(limitEnInt)
		mapParticiones[numeroDeParticion] = true                                              //marcar particion como ocupada
		if err := guardarPCBenMapConRespectivaParticion(pcb, numeroDeParticion); err != nil { //GUARDO EN EL MAP pcb, y el numero de particion
			http.Error(w, err.Error(), http.StatusInternalServerError) //MII MAP DE PCB X NMRO DE PARTICION
			return
		}

		if err := guardarPCBEnElMap(pcb); err != nil { //ACA ESTOY GUARDANDO LA PCB EN MI MAP PRINCIPAL EL MAS IMPORTANTE DE TODOS
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Log de creación de proceso
		log.Printf("## Proceso Creado - PID: %d - Tamaño: %d", process.Pid, process.Size)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Ok"))
		return
	} else if esquemaMemoria == "DINAMICAS" {

		numeroDeParticion := asignarPorAlgoritmo(algoritmoBusqueda, process.Size)

		//SI NO HAY PARTICION DISPONIBLE
		if numeroDeParticion == -1 {
			if espacioLibreSuficiente(process.Size) { //funcion que me devuelve true o false si hay espacio suficiente sumando todas las particiones libres
				compactarLasParticiones() //compacto las particiones libres
				numeroDeParticion = asignarPorAlgoritmo(algoritmoBusqueda, process.Size)
			} else {
				http.Error(w, "No hay espacio en la memoria", http.StatusConflict)
				return
			}
		}

		//SI HAY PARTICION DISPONIBLE PARA EL TAMAÑO DEL PROCESO
		if particiones[numeroDeParticion] > process.Size {
			subdividirParticion(numeroDeParticion, process.Size) //subdivir la particion en dos (una ocupada y otra libre)
		}

		//BASE
		var baseEnInt int
		pcb.Base = 0
		for i := 0; i < numeroDeParticion; i++ {
			baseEnInt += particiones[i] //tengo que ver tema int y uint32
		}
		pcb.Base = uint32(baseEnInt)

		//LIMIT
		limitEnInt = baseEnInt + particiones[numeroDeParticion] - 1
		pcb.Limit = uint32(limitEnInt)

		// mapParticiones[numeroDeParticion] = true //marcar particion como ocupada

		if err := guardarPCBenMapConRespectivaParticion(pcb, numeroDeParticion); err != nil { //GUARDO EN EL MAP pcb, y el numero de particion
			http.Error(w, err.Error(), http.StatusInternalServerError) //MII MAP DE PCB X NMRO DE PARTICION
			return
		}

		if err := guardarPCBEnElMap(pcb); err != nil { //ACA ESTOY GUARDANDO LA PCB EN MI MAP PRINCIPAL EL MAS IMPORTANTE DE TODOS
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Log de creación de proceso
		log.Printf("## Proceso Creado - PID: %d - Tamaño: %d", process.Pid, process.Size)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Ok"))
		return
	}
}

//------------------------------FUNCIONES PARA MEMORIA DINAMICA---------------------------------------------------

// SUMATORIA DE PARTICIONES LIBRES
func espacioLibreSuficiente(Size int) bool {
	espacioLibre := 0
	for i, ocupada := range mapParticiones { //recorreria mi map de particiones de booleanos, entonces agarro libres las que tienen 0
		if !ocupada { //si la particion esta en 0
			espacioLibre += particiones[i] //la voy a sumar
		}
	}
	return espacioLibre >= Size
}

// COMPACTAR LAS PARTICIONES QUE ESTAN LIBRES
func compactarLasParticiones() {
	nuevaParticion := 0
	i := 0
	mapeoOriginalANuevo := make(map[int]int)

	for i < len(particiones) { //la idea es recorrer todas las particiones
		if !mapParticiones[i] { // Si la partición está libre, la sumamos al total y la eliminamos
			nuevaParticion += particiones[i]                                     //aca guardo el tamaño para mi nueva particion que va a ser la compacta
			particiones = append(particiones[:i], particiones[i+1:]...)          // se saca la partición
			mapParticiones = append(mapParticiones[:i], mapParticiones[i+1:]...) // actualizar el map de estados
		} else {
			mapeoOriginalANuevo[i] = len(particiones)
			i++ // si llega a estar ocupada la particion, paso a la siguiente
		}
	}

	particiones = append(particiones, nuevaParticion)
	mapParticiones = append(mapParticiones, false) // La nueva partición estará libre

	actualizarPCBxParticionNueva(mapeoOriginalANuevo) //actualizo el mapa de pcb por particion
}

func actualizarPCBxParticionNueva(mapeoOriginalANuevo map[int]int) {

	nuevoMapPCBPorParticion := make(map[PCB]int)

	for pcb, particionOriginal := range mapPCBPorParticion {
		if nuevaParticion, ok := mapeoOriginalANuevo[particionOriginal]; ok {
			nuevoMapPCBPorParticion[pcb] = nuevaParticion
		} else {
			nuevoMapPCBPorParticion[pcb] = particionOriginal
		}
	}

	mapPCBPorParticion = nuevoMapPCBPorParticion
}

func subdividirParticion(numeroDeParticion, processSize int) {

	originalTam := particiones[numeroDeParticion] //ej: 500 y mi proceso es 100, enntonces en originalTam sera 500

	particiones[numeroDeParticion] = processSize // cambio el tamaño de esa particion que antes era de 500 por ahora 100
	mapParticiones[numeroDeParticion] = true     // la marco como una particion ocupada

	espacioRestante := originalTam - processSize //me sobraron 400 de espacio que no se uso, entonces creo una nueva particion que esta va a estar libre
	if espacioRestante > 0 {
		particiones = append(particiones, espacioRestante) //agrego la nueva particion al vector de particiones
		mapParticiones = append(mapParticiones, false)     // y esta nueva particion va a estar libre para ser la proxima a usar
	}
}

//--------------------------------------------------------------------

var mapPCBPorParticion = make(map[PCB]int)

func guardarPCBenMapConRespectivaParticion(pcb PCB, numeroDeParticion int) error {
	mapPCBPorParticion[pcb] = numeroDeParticion
	return nil
}

func guardarPCBEnElMap(pcb PCB) error {
	if _, found := mapPCBPorTCB[pcb]; !found {
		mapPCBPorTCB[pcb] = make(map[estructuraHilo][]string)
	}
	return nil
}

func asignarPorAlgoritmo(tipoDeAlgoritmo string, size int) int {
	switch tipoDeAlgoritmo {
	case "FIRST":
		return firstFit(size)
	case "BEST":
		return bestFit(size)
	case "WORST":
		return worstFit(size)
	default:
		fmt.Println("Tipo de algoritmo no reconocido")
		return -1
	}
}

// var mapParticiones[]bool //estado de las particiones ocupada/libre
// var particiones = MemoriaConfig.Particiones //vector de particiones, aca tengo los tamaños en int

func firstFit(processSize int) int {
	for i, size := range particiones {
		if !mapParticiones[i] && size >= processSize {
			return i
		}
	}
	return -1
}

func bestFit(processSize int) int {
	bestIndex := -1
	minDifference := math.MaxInt32

	for i, size := range particiones {
		if !mapParticiones[i] && size >= processSize {
			difference := size - processSize
			if difference < minDifference {
				minDifference = difference
				bestIndex = i
			}
		}
	}
	return bestIndex
}

func worstFit(processSize int) int {
	worstIndex := -1
	maxDifference := -1

	for i, size := range particiones {
		if !mapParticiones[i] && size >= processSize {
			difference := size - processSize
			if difference > maxDifference {
				maxDifference = difference
				worstIndex = i
			}
		}
	}
	return worstIndex
}

// func guardarTodoEnElMap(pcb PCB, tcb TCB, path string) error{

// 	// Abro el archivo de pseudocódigo
// 	readFile, err := os.Open(path)
// 	if err != nil {
// 		log.Printf("Error: PATH %s opening file", path)
// 		return err
// 	}
// 	defer readFile.Close()

// 	time.Sleep(time.Duration(MemoriaConfig.Delay_Respuesta) * time.Millisecond)

// 	fileScanner := bufio.NewScanner(readFile)
// 	fileScanner.Split(bufio.ScanLines)

// 	var instrucciones []string // Almaceno cada instrucción en un slice de strings
// 	for fileScanner.Scan() {
// 		instrucciones = append(instrucciones, fileScanner.Text()) //esta linea lee los codigos
// 	}

// 	// Verifico si el PCb ya existe en el mapa
// 	if _, found := mapPCBPorTCB[pcb]; !found {
// 		mapPCBPorTCB[pcb] = make(map[TCB][]string)
// 	}

// 	// Guardo las instrucciones en el mapa correspondiente al PID y TID
// 	mapPCBPorTCB[pcb][tcb] = instrucciones

// 	return nil

// }

//--------------------------------TERMINATE PROCESS---------------------------------------------

func TerminateProcess(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	pid, _ := strconv.Atoi(queryParams.Get("pid"))

	if esquemaMemoria == "FIJAS" { //PARA FIJAS
		var numeroDeParticion int
		encontrado := false
		for pcb, particion := range mapPCBPorParticion {
			if pcb.Pid == pid {
				numeroDeParticion = particion
				encontrado = true
				break
			}
		}

		if !encontrado {
			http.Error(w, "PID no encontrado", http.StatusNotFound)
			return
		}

		mapParticiones[numeroDeParticion] = false // libero el map booleano que indicaba si la particion esta libre o no

		delete(mapPCBPorParticion, PCB{Pid: pid}) // elimino la estructura del pcb en el map de particiones
		delete(mapPCBPorTCB, PCB{Pid: pid})       // elimino el pcb del map anidado

		// Log de destrucción de proceso
		log.Printf("## Proceso Destruido - PID: %d - Tamaño: %d", pid, particiones[numeroDeParticion])

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Proceso finalizado exitosamente"))
	} else if esquemaMemoria == "DINAMICAS" {

		var numeroDeParticion int
		encontrado := false
		for pcb, particion := range mapPCBPorParticion {
			if pcb.Pid == pid {
				numeroDeParticion = particion
				encontrado = true
				break
			}
		}
		if !encontrado {
			http.Error(w, "PID no encontrado", http.StatusNotFound)
			return
		}
		mapParticiones[numeroDeParticion] = false

		consolidarParticiones(numeroDeParticion) //consolido las particiones libres

		delete(mapPCBPorParticion, PCB{Pid: pid})
		delete(mapPCBPorTCB, PCB{Pid: pid})

		// Log de destrucción de proceso
		log.Printf("## Proceso Destruido - PID: %d - Tamaño: %d", pid, particiones[numeroDeParticion])

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Proceso finalizado exitosamente"))
	}
}

func consolidarParticiones(numeroDeParticion int) {
	mapeoOriginalANuevo := make(map[int]int)

	//IZQUIERDA
	for numeroDeParticion > 0 && !mapParticiones[numeroDeParticion-1] {
		particiones[numeroDeParticion-1] += particiones[numeroDeParticion]                          // Sumar tamaño de la partición actual a la anterior
		particiones = append(particiones[:numeroDeParticion], particiones[numeroDeParticion+1:]...) // Eliminar partición actual
		mapParticiones = append(mapParticiones[:numeroDeParticion], mapParticiones[numeroDeParticion+1:]...)

		for pcb, particion := range mapPCBPorParticion {
			if particion == numeroDeParticion {
				mapPCBPorParticion[pcb] = numeroDeParticion - 1
			} else if particion > numeroDeParticion {
				mapeoOriginalANuevo[particion] = particion - 1
			}
		}
		numeroDeParticion--
	}

	//DERECHA
	for numeroDeParticion < len(particiones)-1 && !mapParticiones[numeroDeParticion+1] {
		particiones[numeroDeParticion] += particiones[numeroDeParticion+1]
		particiones = append(particiones[:numeroDeParticion+1], particiones[numeroDeParticion+2:]...)
		mapParticiones = append(mapParticiones[:numeroDeParticion+1], mapParticiones[numeroDeParticion+2:]...)

		for pcb, particion := range mapPCBPorParticion {
			if particion == numeroDeParticion+1 {
				mapPCBPorParticion[pcb] = numeroDeParticion
			} else if particion > numeroDeParticion+1 {
				mapeoOriginalANuevo[particion] = particion - 1
			}
		}
	}

	actualizarPCBxParticionNueva(mapeoOriginalANuevo)
}

//-----------------------------------------CREATE THREAD--------------------------------------------

func CreateThread(w http.ResponseWriter, r *http.Request) {
	log.Printf("entre a crear thread")
	var thread Thread
	time.Sleep(time.Duration(MemoriaConfig.Delay_Respuesta) * time.Millisecond)

	// if err := json.NewDecoder(r.Body).Decode(&thread); err != nil {
	// 	http.Error(w, err.Error(), http.StatusBadRequest)
	// 	return
	// }

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&thread)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	TCB := estructuraHilo{ //creo la estructura necesaria
		Pid: thread.Pid,
		Tid: thread.Tid,
		AX:  0,
		BX:  0,
		CX:  0,
		DX:  0,
		EX:  0,
		FX:  0,
		GX:  0,
		HX:  0,
		PC:  0,
	}
	log.Printf("cree el hilo PID:%d TID:%d", thread.Pid, thread.Tid)
	if err := guardarTodoEnElMap(thread.Pid, TCB, thread.Path); err != nil { //GUARDO EN EL MAP
		log.Printf("ERROR AL GUARDAR")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Lo que se guardo en map: %v", mapPCBPorTCB)
	// Log de creación de hilo
	log.Printf("## Hilo Creado - (PID:TID) - (%d:%d)", thread.Pid, thread.Tid)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ok"))
	return
}

func guardarTodoEnElMap(pid int, TCB estructuraHilo, path string) error {
	log.Printf("%s", path)
	// Abro el archivo de pseudocódigo
	readFile, err := os.Open(path)
	if err != nil {
		log.Printf("Error: PATH %s opening file", path)
		return err
	}
	defer readFile.Close()
	log.Printf("Abri el archivo")
	time.Sleep(time.Duration(MemoriaConfig.Delay_Respuesta) * time.Millisecond)

	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)

	var instrucciones []string // Almaceno cada instrucción en un slice de strings
	for fileScanner.Scan() {
		instrucciones = append(instrucciones, fileScanner.Text()) //esta linea lee los codigos
	}
	log.Printf("escaneo")
	var pcbEncontrado PCB //para encontrar el pcb y poder entrar al mapa anidado
	for pcb := range mapPCBPorTCB {
		if pcb.Pid == pid {
			pcbEncontrado = pcb
			break
		}
	}
	log.Printf("busco pcb")
	if _, found := mapPCBPorTCB[pcbEncontrado]; !found {
		mapPCBPorTCB[pcbEncontrado] = make(map[estructuraHilo][]string)
	}
	log.Printf("hago el map")
	mapPCBPorTCB[pcbEncontrado][TCB] = instrucciones
	log.Printf("fin")
	return nil

}

//---------------------------------------TERMINATE THREAD--------------------------------------------

type Req struct {
	Pid int `json:"pid"`
	Tid int `json:"tid"`
}

func TerminateThread(w http.ResponseWriter, r *http.Request) {

	var req Req
	time.Sleep(time.Duration(MemoriaConfig.Delay_Respuesta) * time.Millisecond)

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if _, exists := mapPCBPorTCB[PCB{Pid: req.Pid}]; !exists {
		http.Error(w, "No se pudo encontrar el proceso", http.StatusNotFound)
		return
	}

	if tcbMap, found := mapPCBPorTCB[PCB{Pid: req.Pid}]; found {
		delete(tcbMap, estructuraHilo{Pid: req.Pid, Tid: req.Tid})
		if len(tcbMap) == 0 {
			delete(mapPCBPorTCB, PCB{Pid: req.Pid}) //por si llega a quedar vacio
		}
	}

	// Log de destrucción de hilo
	log.Printf("## Hilo Destruido - (PID:TID) - (%d:%d)", req.Pid, req.Tid)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

//-----------------------------------------READ MEMORY-------------------------------------------

type MemoryRequest struct {
	PID     int    `json:"pid"`
	TID     int    `json:"tid,omitempty"`
	Address uint32 `json:"address"`        //direccion de memoria a leer
	Size    int    `json:"size,omitempty"` //tamaño de la memoria a leer
	Data    []byte `json:"data,omitempty"` //datos a escribir o leer y los devuelvo
	Port    int    `json:"port,omitempty"` //puerto
}

func ReadMemoryHandler(w http.ResponseWriter, r *http.Request) {
	var memReq MemoryRequest
	time.Sleep(time.Duration(MemoriaConfig.Delay_Respuesta) * time.Millisecond)

	if err := json.NewDecoder(r.Body).Decode(&memReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	data, err := ReadMemory(memReq.PID, memReq.TID, memReq.Address) //, memReq.Size )
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := sendDataToCPU(data); err != nil {
		http.Error(w, "Error al enviar los datos a la CPU", http.StatusInternalServerError)
		return
	}
}

var mu sync.Mutex

// tener en cuenta lo de que si me dan para leer y en vez de leer 4 voy a llegar a leer 2 porque se me termino la particion
// tener en cuenta lo de que si me dan para leer desde 4 y leo hasta el 8 pero mi particion termina en 12.
// tengo que leer y escribir pero en mi slice de memoria, las particiones corte el limite y base lo voy a utilizar para calcular todo lo anterior

func ReadMemory(PID int, TID int, address uint32) ([]byte, error) { //size capaz sacarlo y poner directamente 4
	mu.Lock()
	defer mu.Unlock()

	if _, exists := mapPCBPorTCB[PCB{Pid: PID}]; !exists {
		return nil, fmt.Errorf("no se encontró el PID")
	}

	var pcbEncontrado PCB //LO HAGO PARA PODER ENTRAR AL MAPA ANIDADO Y AGARRAR LA PCB DE ESE PID
	encontrado := false

	for pcb := range mapPCBPorTCB {
		if pcb.Pid == PID {
			pcbEncontrado = pcb
			encontrado = true
			break
		}
	}

	tcbMap, found := mapPCBPorTCB[pcbEncontrado]
	if !found {
		return nil, fmt.Errorf("no se encontró el TID para el PID: %d", PID)
	}

	encontrado = false
	for tcb := range tcbMap {
		if tcb.Tid == TID {
			encontrado = true
			break
		}
	}

	if !encontrado {
		return nil, fmt.Errorf("no se encontró el TID: %d para el PID: %d", TID, PID)
	}

	//primero tengo que ver si la direccion que me dieron esta dentro del rango de la particion del pid
	if address < pcbEncontrado.Base || address > pcbEncontrado.Limit {
		return nil, fmt.Errorf("Direccion fuera de rango")
	}

	solocuatro := uint32(4)

	//si se me esta por terminar la particion y no llegue a cuatro
	if address+solocuatro > pcbEncontrado.Limit {
		solocuatro = pcbEncontrado.Limit - address
	}

	// Leer los bytes en la memoria
	data := make([]byte, solocuatro)
	copy(data, globals.MemoriaUsuario[address:address+solocuatro])

	//completo con ceros si no se llego a leer 4 bytes
	if len(data) < 4 {
		padding := make([]byte, 4-len(data)) //creo un slice con los numeros que va a tener la cantidad de bytes que me faltan para llegar a 4
		data = append(data, padding...)      //asi cada data tiene 4 bytes
	}

	return data, nil
}

func sendDataToCPU(content []byte) error {

	CPUurl := fmt.Sprintf("http://%s:%d/receiveDataFromMemory", IpCpu, PuertoCpu)
	ContentResponseTest, err := json.Marshal(content)
	if err != nil {
		log.Fatalf("Error al serializar el Input: %v", err)
	}

	resp, err := http.Post(CPUurl, "application/json", bytes.NewBuffer(ContentResponseTest))
	if err != nil {
		log.Fatalf("Error al enviar la solicitud al módulo de memoria: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Error en la respuesta del módulo de memoria: %v", resp.StatusCode)
	}

	return nil
}

//----------------------------------------------WRITE MEMORY-------------------------------------------------

// primero me tiene que llegar el pid y el tid, la direccion de memoria y los datos a escribir
// el pid el tid la direccion de memoria en la cual voy a comenzar a escribir los datos que me llegan,
// Y los datos me llegan en un string, y eso lo voy a convertir a un slice de bytes y lo voy a escribir en la memoria
// otra vez fundamentalmente escribir sobre la memoria "grande"

func WriteMemoryHandler(w http.ResponseWriter, r *http.Request) {
	var memReq MemoryRequest
	time.Sleep(time.Duration(MemoriaConfig.Delay_Respuesta) * time.Millisecond)

	if err := json.NewDecoder(r.Body).Decode(&memReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := WriteMemory(memReq.PID, memReq.TID, memReq.Address, memReq.Data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func WriteMemory(PID int, TID int, address uint32, data []byte) error {

	mu.Lock()
	defer mu.Unlock()

	if _, exists := mapPCBPorTCB[PCB{Pid: PID}]; !exists {
		return fmt.Errorf("no se encontró el PID")
	}

	var pcbEncontrado PCB //LO HAGO PARA PODER ENTRAR AL MAPA ANIDADO Y AGARRAR LA PCB DE ESE PID
	encontrado := false

	for pcb := range mapPCBPorTCB {
		if pcb.Pid == PID {
			pcbEncontrado = pcb
			encontrado = true
			break
		}
	}

	tcbMap, found := mapPCBPorTCB[pcbEncontrado]
	if !found {
		return fmt.Errorf("no se encontró el TID para el PID: %d", PID)
	}

	encontrado = false
	for tcb := range tcbMap {
		if tcb.Tid == TID {
			encontrado = true
			break
		}
	}

	if !encontrado {
		return fmt.Errorf("no se encontró el TID: %d para el PID: %d", TID, PID)
	}

	//primero tengo que ver si la direccion que me dieron esta dentro del rango de la particion del pid
	if address < pcbEncontrado.Base || address > pcbEncontrado.Limit {
		return fmt.Errorf("dirección fuera de rango para el PID: %d", PID)
	}

	espaciodisponible := pcbEncontrado.Limit - address // Espacio disponible desde la dirección hasta el límite

	if espaciodisponible >= 4 {
		copy(globals.MemoriaUsuario[address:address+4], data[:4])
	} else {
		copy(globals.MemoriaUsuario[address:address+espaciodisponible], data[:espaciodisponible])
	}

	return nil
}

//-----------------------------------------------------------------------------------------------------

// func EnviarAModulo(ipModulo string, puertoModulo int, body io.Reader, endPoint string) error {

// 	url := fmt.Sprintf("http://%s:%d/%s", ipModulo, puertoModulo, endPoint)
// 	resp, err := http.Post(url, "application/json", body)
// 	if err != nil {
// 		log.Printf("error enviando mensaje al End point %s - IP:%s - Puerto:%d", endPoint, ipModulo, puertoModulo)
// 		return err
// 	}
// 	if resp.StatusCode != http.StatusOK {
// 		log.Printf("Error al recibir la respuesta del End point %s - IP:%s - Puerto:%d", endPoint, ipModulo, puertoModulo)
// 		err := fmt.Errorf("%s", resp.Status)
// 		return err
// 	}
// 	return nil
// }

//-------------------------------DUMP MEMORY------------------------------------------------

// func DumpMemory(w http.ResponseWriter, r *http.Request) {
// 	queryParams := r.URL.Query()
// 	pid, _ := strconv.Atoi(queryParams.Get("pid"))
// 	tid, _ := strconv.Atoi(queryParams.Get("tid"))

// 	time.Sleep(time.Duration(MemoriaConfig.Delay_Respuesta) * time.Millisecond)

// 	for pcb, tidMap := range mapPCBPorTCB {

// }
