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
	Base  uint32 //no las usaria
	Limit uint32 //no las usaria
}

type Valor struct {
	Base  uint32
	Limit uint32
}

type InstructionResponse struct {
	Instruction string `json:"instruction"`
}
type DataRead struct {
	Data []byte `json:"data"`
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
type TCB struct {
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

type estadoMemoria struct {
	Estado int `json:"estado"`
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

var mapPIDxBaseLimit = make(map[int]Valor) //map de pid por base y limit

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
	// Si el config no tiene nada termina
	if MemoriaConfig == nil {
		log.Fatal("ClientConfig is not initialized")
		panic("ClientConfig is not initialized")
	}
	// Modifica las variables globales
	particiones = MemoriaConfig.Particiones
	mapParticiones = make([]bool, len(particiones))
	globals.MemoriaUsuario = make([]byte, MemoriaConfig.Tamanio_Memoria)
	esquemaMemoria = MemoriaConfig.EsquemaMemoria
	algoritmoBusqueda = MemoriaConfig.AlgoritmoBusqueda
	IpCpu = MemoriaConfig.IpCpu
	PuertoCpu = MemoriaConfig.PuertoCpu
	//MemoriaTamanio = MemoriaConfig.Tamanio_Memoria

	log.Printf("%d", particiones)
}

///////////////////////////////////////////////////////////////////////////////

// Función para buscar la estructura Valor dado un pid
func BuscarBaseLimitPorPID(pid int) (Valor, error) {

	if valor, existe := mapPIDxBaseLimit[pid]; existe {
		return valor, nil
	}
	return Valor{}, fmt.Errorf("PID %d no encontrado en el mapa", pid)
}

// /--------------------------------------------GET INSTRUCTION---------------------------------------------
func GetInstruction(w http.ResponseWriter, r *http.Request) {
	var instructionReq InstructionReq
	log.Printf("Entrando a GetInstruction")

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&instructionReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	time.Sleep(time.Duration(MemoriaConfig.Delay_Respuesta) * time.Millisecond)
	// Buscar el PCB que tenga el Pid solicitado y nos da las estructuras de los hilos asociado
	tidMap := buscarTCBPorPid(instructionReq.Pid)
	if tidMap == nil {
		http.Error(w, fmt.Sprintf("No se encontró el PID %d", instructionReq.Pid), http.StatusNotFound)
		log.Printf("error: no se encontró el PID %d", instructionReq.Pid)
		return
	}
	// Buscar el TCB por el Tid
	var instrucciones []string
	for tcb, inst := range tidMap {
		if tcb.Tid == instructionReq.Tid {
			instrucciones = inst
			break
		}
	}
	if instrucciones == nil {
		http.Error(w, fmt.Sprintf("No se encontró el TID %d para el PID %d", instructionReq.Tid, instructionReq.Pid), http.StatusNotFound)
		log.Printf("error: no se encontró el TID %d para el PID %d", instructionReq.Tid, instructionReq.Pid)
		return
	}

	// Verificar si el PC está dentro del rango de instrucciones
	if instructionReq.Pc < 0 || instructionReq.Pc >= len(instrucciones) {
		http.Error(w, fmt.Sprintf("El PC %d está fuera del rango de instrucciones (PID: %d, TID: %d)", instructionReq.Pc, instructionReq.Pid, instructionReq.Tid), http.StatusBadRequest)
		return
	}

	// Devolver la instrucción solicitada
	instruccion := instrucciones[instructionReq.Pc]
	instructionResponse := InstructionResponse{Instruction: instruccion}

	// Log de obtención de instrucción
	log.Printf("## Obtener instrucción - (PID:TID) - (%d:%d) - Instrucción: %s", instructionReq.Pid, instructionReq.Tid, instruccion)

	// Envio la respuesta en formato JSON
	json.NewEncoder(w).Encode(instructionResponse)
	w.Write([]byte(instruccion))
}

func buscarTCBPorPid(pid int) map[estructuraHilo][]string {

	for pcb, tcbMap := range mapPCBPorTCB {
		if pcb.Pid == pid {
			return tcbMap
		}
	}
	return nil
}

func obtenerPCBPorPID(PID int) (PCB, error) {
	for pcb := range mapPCBPorTCB {
		if pcb.Pid == PID {
			return pcb, nil
		}
	}
	log.Printf("No se encontró el PID: %d", PID)
	return PCB{}, fmt.Errorf("no se encontró el PID: %d", PID)
}

// ------------------------------------ GET EXECUTION CONTEXT ---------------------------------------------
type GetExecutionContextResponse struct {
	Pcb PCB            `json:"pcb"`
	Tcb estructuraHilo `json:"tcb"`
}

func GetExecutionContext(w http.ResponseWriter, r *http.Request) {
	var solicitud KernelExeReq

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&solicitud)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("PCB : %d TID : %d - me llegaron estos valores", solicitud.Pid, solicitud.Tid)

	time.Sleep(time.Duration(MemoriaConfig.Delay_Respuesta) * time.Millisecond)

	// Usar la función `buscarTCBPorPid` para obtener el tidMap
	tidMap := buscarTCBPorPid(solicitud.Pid)
	if tidMap == nil {
		http.Error(w, fmt.Sprintf("No se encontró el PID %d", solicitud.Pid), http.StatusNotFound)
		log.Printf("error: no se encontró el PID %d", solicitud.Pid)
		return
	}

	// Buscar el TCB dentro del tidMap
	for tcb := range tidMap {
		if tcb.Tid == solicitud.Tid {
			// Obtener valores de base y limit desde otro mapa
			valores := mapPIDxBaseLimit[solicitud.Pid]
			var respuesta GetExecutionContextResponse

			respuesta.Pcb.Pid = solicitud.Pid
			respuesta.Pcb.Base = valores.Base
			respuesta.Pcb.Limit = valores.Limit
			respuesta.Tcb = tcb

			log.Printf("Pid %d y Tid %d encontrados", solicitud.Pid, tcb.Tid)

			// Codificar la respuesta como JSON
			respuestaJson, err := json.Marshal(respuesta)
			if err != nil {
				http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write(respuestaJson)

			// Log de obtener el contexto de ejecución
			log.Printf("## Contexto <Solicitado> - (PID:TID) - (%d:%d)", solicitud.Pid, solicitud.Tid)
			return
		}
	}

	// Si no se encuentra el TID
	http.Error(w, "No se encontró el TID", http.StatusNotFound)
	log.Printf("error: no se encontró el TID %d para el PID %d", solicitud.Tid, solicitud.Pid)
}

//-------------------------------- UPDATE EXECUTION CONTEXT-----------------------------------------------

func UpdateExecutionContext(w http.ResponseWriter, r *http.Request) {
	var actualizadoContexto GetExecutionContextResponse

	time.Sleep(time.Duration(MemoriaConfig.Delay_Respuesta) * time.Millisecond)

	if err := json.NewDecoder(r.Body).Decode(&actualizadoContexto); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("Respuesta codificada PID = %d , TID = %d", actualizadoContexto.Pcb.Pid, actualizadoContexto.Tcb.Tid)

	// Usar la función `buscarTCBPorPid` para obtener el tidMap
	tidMap := buscarTCBPorPid(actualizadoContexto.Pcb.Pid)
	if tidMap == nil {
		http.Error(w, fmt.Sprintf("No se encontró el PID %d", actualizadoContexto.Pcb.Pid), http.StatusNotFound)
		log.Printf("error: no se encontró el PID %d", actualizadoContexto.Pcb.Pid)
		return
	}

	// Buscar el TCB dentro del tidMap
	for tcb := range tidMap {
		log.Printf("TID actualizar : %d", tcb.Tid)
		if tcb.Tid == actualizadoContexto.Tcb.Tid {
			// Modificar contexto y valores
			ModificarContexto(actualizadoContexto.Pcb, tcb, actualizadoContexto.Tcb)
			ModificarValores(actualizadoContexto.Pcb.Pid, actualizadoContexto.Pcb.Base, actualizadoContexto.Pcb.Limit)

			// Log de contexto de ejecución actualizado
			log.Printf("## Contexto Actualizado - (PID:TID) - (%d:%d)", actualizadoContexto.Pcb.Pid, actualizadoContexto.Tcb.Tid)
			log.Printf("Contexto = %v", mapPCBPorTCB[actualizadoContexto.Pcb])

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("El contexto de ejecución ha sido actualizado"))
			return
		}
	}

	// Si no se encuentra el TID
	http.Error(w, "TID no ha sido encontrado", http.StatusNotFound)
	log.Printf("error: no se encontró el TID %d para el PID %d", actualizadoContexto.Tcb.Tid, actualizadoContexto.Pcb.Pid)
}

//-----------------MODIFICAR CONTEXTO----------(NUEVA FUNCION)----

func ModificarContexto(pcbEncontrado PCB, tcbEncontrada estructuraHilo, nuevoTCB estructuraHilo) {

	instrucciones := mapPCBPorTCB[pcbEncontrado][tcbEncontrada]

	delete(mapPCBPorTCB[pcbEncontrado], tcbEncontrada)

	mapPCBPorTCB[pcbEncontrado][nuevoTCB] = instrucciones
}

//-----------------------------MODIFICAR VALORES(BASE Y LIMITE)-------------------------------------

func ModificarValores(pid int, base uint32, limit uint32) {

	valor := Valor{Base: base, Limit: limit}

	mapPIDxBaseLimit[pid] = valor
}

//-----------------------------------------CREATE PROCESS-------------------------------------------

const (
	HayEspacio   int = 1
	Compactar        = 2
	NoHayEspacio     = 3
)

func CreateProcess(w http.ResponseWriter, r *http.Request) { //recibe la pid y el size
	var process Process
	var limitEnInt int
	var estado estadoMemoria

	time.Sleep(time.Duration(MemoriaConfig.Delay_Respuesta) * time.Millisecond)

	if err := json.NewDecoder(r.Body).Decode(&process); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("Pid: %d", process.Pid)
	log.Printf("size: %d", process.Size)
	pcb := PCB{ //creo la estructura necesaria
		Pid:   process.Pid,
		Base:  0,
		Limit: 0,
	}

	if esquemaMemoria == "FIJAS" {

		log.Printf("esquema: FIJAS")
		numeroDeParticion := asignarPorAlgoritmo(algoritmoBusqueda, process.Size) //asigno por algoritmo

		if numeroDeParticion == -1 {

			http.Error(w, "No hay espacio en la memoria", http.StatusConflict)
			estado.Estado = NoHayEspacio

		} else {
			estado.Estado = HayEspacio

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

			mapPIDxBaseLimit[process.Pid] = Valor{Base: pcb.Base, Limit: pcb.Limit}

			//marcar particion como ocupada
			if err := guardarPCBenMapConRespectivaParticion(pcb, numeroDeParticion); err != nil { //GUARDO EN EL MAP pcb, y el numero de particion
				http.Error(w, err.Error(), http.StatusInternalServerError) //MII MAP DE PCB X NMRO DE PARTICION
				return
			}

			if err := guardarPCBEnElMap(pcb); err != nil { //ACA ESTOY GUARDANDO LA PCB EN MI MAP PRINCIPAL EL MAS IMPORTANTE DE TODOS
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			log.Printf("particion %d mapespacio: %v", numeroDeParticion, mapParticiones)

			// Log de creación de proceso
			log.Printf("## Proceso Creado - PID: %d - Tamaño: %d", process.Pid, process.Size)

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Ok"))

		}
	} else if esquemaMemoria == "DINAMICAS" {

		numeroDeParticion := asignarPorAlgoritmo(algoritmoBusqueda, process.Size)

		//SI NO HAY PARTICION DISPONIBLE
		if numeroDeParticion == -1 {
			//aca deberia de alguna manera verificar si puede o no compactar
			if espacioLibreSuficiente(process.Size) { //funcion que me devuelve true o false si hay espacio suficiente sumando todas las particiones libres
				estado.Estado = Compactar
				log.Printf("COMPACTE")
				//compactarLasParticiones() //compacto las particiones libres
				//actualizarBasesYLímites() //actualizo las bases y limites
			} else {
				log.Printf("NO HAY ESPACIO")
				http.Error(w, "No hay espacio en la memoria", http.StatusConflict)
				estado.Estado = NoHayEspacio
			}
		} else {
			estado.Estado = HayEspacio
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

			mapPIDxBaseLimit[process.Pid] = Valor{Base: pcb.Base, Limit: pcb.Limit}

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
		}

	}

	respuesta, err := json.Marshal(&estado)

	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK) // Esto da superfluos error ya que si arriba retorna http error al no tener un return genera un error, hay que ver como solucionarlo
	w.Write(respuesta)
}

func guardarPCBenMapConRespectivaParticion(pcb PCB, numeroDeParticion int) error {
	mapPCBPorParticion[pcb] = numeroDeParticion
	return nil
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
	log.Printf("Particiones: %v Espacio libre: %d Tamaño: %d", particiones, espacioLibre, Size)
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

func actualizarBasesYLímites() {
	baseAcumulada := 0

	for i := 0; i < len(particiones); i++ {
		if mapParticiones[i] { // Si la partición está ocupada
			for pcb, particion := range mapPCBPorParticion {
				if particion == i {
					// Actualizar la base y el límite
					pcb.Base = uint32(baseAcumulada)
					pcb.Limit = uint32(baseAcumulada + particiones[i] - 1)

					// Actualizar en el mapa PID -> Base/Limit
					mapPIDxBaseLimit[pcb.Pid] = Valor{
						Base:  pcb.Base,
						Limit: pcb.Limit,
					}

					// Incrementar la base acumulada
					baseAcumulada += particiones[i]
				}
			}
		}
	}
}

//--------------------------------------------------------------------

var mapPCBPorParticion = make(map[PCB]int)

func guardarPCBEnElMap(pcb PCB) error {
	if _, found := mapPCBPorTCB[pcb]; !found {
		mapPCBPorTCB[pcb] = make(map[estructuraHilo][]string)
	}
	return nil
}

func asignarPorAlgoritmo(tipoDeAlgoritmo string, size int) int {
	switch tipoDeAlgoritmo {
	case "FIRST":
		log.Printf("algoritmo: FIRST") //BORRAR
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
			mapParticiones[i] = true                                                  //Bloquea la particion ya que fue asignada
			log.Printf("espacio de: %d tomado %v", particiones[i], mapParticiones[i]) //BORRRAR
			return i
		}
	}
	log.Printf("no hay espacio")
	return -1
}

func bestFit(processSize int) int {
	bestIndex := -1
	particionesConSuficienteTamaño := 0 //n
	minDifference := math.MaxInt32

	for i, size := range particiones {
		if size >= processSize { //n
			if !mapParticiones[i] { //n
				difference := size - processSize
				if difference < minDifference {
					minDifference = difference
					bestIndex = i
				}
			}
			particionesConSuficienteTamaño++ //n
		}
	}
	if bestIndex != -1 {
		mapParticiones[bestIndex] = true                     //Bloquea la particion ya que fue asignada
		log.Printf("espacio de: %d", particiones[bestIndex]) //BORRRAR
	}
	if bestIndex != -1 && particionesConSuficienteTamaño == 0 {
		log.Printf("Estas intentando crear un proceso con un tamaño mayor a todos los espacios de memoria")
		panic("Imposible crear proceso")
	}

	return bestIndex
}

func worstFit(processSize int) int {
	worstIndex := -1
	maxDifference := -1 //ARREGLAR IGUAL QUE BEST
	particionesConSuficienteTamaño := 0

	for i, size := range particiones {
		if size >= processSize {
			if !mapParticiones[i] {
				difference := size - processSize
				if difference > maxDifference {
					maxDifference = difference
					worstIndex = i
				}
			}
			particionesConSuficienteTamaño++
		}
	}

	if worstIndex != -1 {
		mapParticiones[worstIndex] = true
	}
	if worstIndex != -1 && particionesConSuficienteTamaño == 0 {
		log.Printf("Estas intentando crear un proceso con un tamaño mayor a todos los espacios de memoria")
		panic("Imposible crear proceso")
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

type KernelProcessTerminateReq struct {
	Pid int `json:"pid"`
}

func TerminateProcess(w http.ResponseWriter, r *http.Request) {
	log.Printf("Entra a terminate process")
	var kernelReq KernelProcessTerminateReq
	if err := json.NewDecoder(r.Body).Decode(&kernelReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	pid := kernelReq.Pid
	numeroDeParticion, err := encontrarParticionPorPID(pid)
	tamanio := particiones[numeroDeParticion]
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if esquemaMemoria == "FIJAS" { //PARA FIJAS
		mapParticiones[numeroDeParticion] = false // libero el map booleano que indicaba si la particion esta libre o no

		delete(mapPCBPorParticion, PCB{Pid: pid}) // elimino la estructura del pcb en el map de particiones
		delete(mapPCBPorTCB, PCB{Pid: pid})       // elimino el pcb del map anidado
		delete(mapPIDxBaseLimit, pid)             // elimino el pid del map de base y limit
	} else if esquemaMemoria == "DINAMICAS" {
		mapParticiones[numeroDeParticion] = false
		consolidarParticiones(numeroDeParticion) //consolido las particiones libres
		delete(mapPCBPorParticion, PCB{Pid: pid})
		delete(mapPCBPorTCB, PCB{Pid: pid})
		delete(mapPIDxBaseLimit, pid)
	}
	// Log de destrucción de proceso
	log.Printf("## Proceso Destruido - PID: %d - Tamaño: %d", pid, tamanio)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Proceso finalizado exitosamente"))
}

func encontrarParticionPorPID(pid int) (int, error) {
	particionEncontrada := -1
	err := fmt.Errorf("PID no encontrado")

	if len(mapPCBPorParticion) == 1 {
		log.Printf("uno") // Borrar
		pcb := PCB{Pid: pid}
		particion, ok := mapPCBPorParticion[pcb]
		if ok {
			particionEncontrada = particion
			err = nil
		}
	} else {
		log.Printf("varios") // Borrar
		for pcb, particion := range mapPCBPorParticion {
			if pcb.Pid == pid {
				log.Printf("encuentro") // CAMBIAR
				particionEncontrada = particion
				err = nil
				break //si haces un return dentro de este genera un error, por eso lo gestione asi
			}
		}
	}

	if err != nil {
		log.Printf("no encuentro") // CAMBIAR
	}

	return particionEncontrada, err
}

func consolidarParticiones(numeroDeParticion int) {
	mapeoOriginalANuevo := make(map[int]int)

	//CONSOLIDAR IZQUIERDA
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

	//CONSOLIDAR DERECHA
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
	var thread Thread
	time.Sleep(time.Duration(MemoriaConfig.Delay_Respuesta) * time.Millisecond)

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

	log.Printf("## Hilo Creado - (PID:TID) - (%d:%d)", thread.Pid, thread.Tid)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ok"))
	return
}

func guardarTodoEnElMap(pid int, TCB estructuraHilo, path string) error {
	log.Printf("Cargando archivo desde: %s", path)

	readFile, err := os.Open(path)
	if err != nil {
		log.Printf("Error al abrir el archivo en PATH: %s", path)
		return err
	}
	defer readFile.Close()
	time.Sleep(time.Duration(MemoriaConfig.Delay_Respuesta) * time.Millisecond)

	fileScanner := bufio.NewScanner(readFile)
	instrucciones := make([]string, 0)

	for fileScanner.Scan() {
		instrucciones = append(instrucciones, fileScanner.Text())
	}

	// Buscar PCB asociado al PID
	var pcbEncontrado PCB
	for pcb := range mapPCBPorTCB {
		if pcb.Pid == pid {
			pcbEncontrado = pcb
			break
		}
	}

	if _, found := mapPCBPorTCB[pcbEncontrado]; !found {
		return fmt.Errorf("PID no encontrado")
	}

	log.Println("Actualizando instrucciones en el mapa")
	mapPCBPorTCB[pcbEncontrado][TCB] = instrucciones
	log.Println("Carga de instrucciones finalizada")
	return nil
}

//---------------------------------------TERMINATE THREAD--------------------------------------------

type Req struct {
	Pid int `json:"pid"`
	Tid int `json:"tid"`
}

func TerminateThread(w http.ResponseWriter, r *http.Request) {
	log.Printf("ENTRO A TERMINATE THREAD")

	var req Req
	time.Sleep(time.Duration(MemoriaConfig.Delay_Respuesta) * time.Millisecond)

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		log.Printf("ENTRO A TERMINATE THREAD")
		return
	}

	/*if _, exists := mapPCBPorTCB[PCB{Pid: req.Pid}]; !exists {
		http.Error(w, "No se pudo encontrar el proceso", http.StatusNotFound)
		log.Printf("ENTRO A TERMINATE THREAD8")
		return
	}*/

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
	log.Println("Iniciando handler para lectura de memoria")
	time.Sleep(time.Duration(MemoriaConfig.Delay_Respuesta) * time.Millisecond)

	var memReq MemoryRequest
	if err := json.NewDecoder(r.Body).Decode(&memReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Println("Leyendo memoria...")
	data, err := ReadMemory(memReq.PID, memReq.TID, memReq.Address)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Datos leídos con éxito")
	respuestaJson, err := json.Marshal(DataRead{Data: data})
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuestaJson)
	log.Println("Datos enviados al CPU")
}

var mu sync.Mutex

// tener en cuenta lo de que si me dan para leer y en vez de leer 4 voy a llegar a leer 2 porque se me termino la particion
// tener en cuenta lo de que si me dan para leer desde 4 y leo hasta el 8 pero mi particion termina en 12.
// tengo que leer y escribir pero en mi slice de memoria, las particiones corte el limite y base lo voy a utilizar para calcular todo lo anterior

func ReadMemory(PID int, TID int, address uint32) ([]byte, error) { //size capaz sacarlo y poner directamente 4
	log.Printf("Accediendo a readMemory")
	mu.Lock()
	defer mu.Unlock()

	// Buscar PCB asociado al PID
	var pcbEncontrado PCB
	encontrado := false

	for pcb := range mapPCBPorTCB {
		if pcb.Pid == PID {
			pcbEncontrado = pcb
			encontrado = true
			break
		}
	}

	if !encontrado {
		log.Printf("PID no encontrado")
		return nil, fmt.Errorf("no se encontró el PID")
	}

	valor, err := BuscarBaseLimitPorPID(PID)
	if err != nil {
		return nil, fmt.Errorf("error al buscar base y límite: %v", err)
	}

	pcbEncontrado.Base, pcbEncontrado.Limit = valor.Base, valor.Limit

	// Verificar si la dirección está dentro del rango
	if address < pcbEncontrado.Base || address > pcbEncontrado.Limit {
		return nil, fmt.Errorf("dirección fuera de rango")
	}

	solocuatro := uint32(4)
	if address+solocuatro > pcbEncontrado.Limit {
		solocuatro = pcbEncontrado.Limit - address // Ajustar tamaño si excede el límite
	}

	// Leer los bytes en la memoria
	data := make([]byte, solocuatro)
	copy(data, globals.MemoriaUsuario[address:address+solocuatro])

	// Completar con ceros si no se leyeron 4 bytes
	if len(data) < 4 {
		data = append(data, make([]byte, 4-len(data))...) // Padding
	}

	return data, nil
}

// func sendDataToCPU(content []byte) error {

// 	CPUurl := fmt.Sprintf("http://%s:%d/receiveDataFromMemory", IpCpu, PuertoCpu)
// 	ContentResponseTest, err := json.Marshal(content)
// 	if err != nil {
// 		log.Fatalf("Error al serializar el Input: %v", err)
// 	}

// 	resp, err := http.Post(CPUurl, "application/json", bytes.NewBuffer(ContentResponseTest))
// 	if err != nil {
// 		log.Fatalf("Error al enviar la solicitud al módulo de memoria: %v", err)
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK {
// 		log.Fatalf("Error en la respuesta del módulo de memoria: %v", resp.StatusCode)
// 	}

// 	return nil
// }

//----------------------------------------------WRITE MEMORY-------------------------------------------------

// primero me tiene que llegar el pid y el tid, la direccion de memoria y los datos a escribir
// el pid el tid la direccion de memoria en la cual voy a comenzar a escribir los datos que me llegan,
// Y los datos me llegan en un string, y eso lo voy a convertir a un slice de bytes y lo voy a escribir en la memoria
// otra vez fundamentalmente escribir sobre la memoria "grande"

func WriteMemoryHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Enters to Write Memory Handler")
	var memReq MemoryRequest
	time.Sleep(time.Duration(MemoriaConfig.Delay_Respuesta) * time.Millisecond)

	if err := json.NewDecoder(r.Body).Decode(&memReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("Intenta escribir memoria")
	if err := WriteMemory(memReq.PID, memReq.TID, memReq.Address, memReq.Data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func WriteMemory(PID int, TID int, address uint32, data []byte) error {
	log.Printf("Enters to Write Memory")
	mu.Lock()
	defer mu.Unlock()

	var pcbEncontrado PCB //LO HAGO PARA PODER ENTRAR AL MAPA ANIDADO Y AGARRAR LA PCB DE ESE PID
	encontrado := false

	for pcb := range mapPCBPorTCB {
		if pcb.Pid == PID {
			pcbEncontrado = pcb
			encontrado = true
			break
		}
	}
	if !encontrado {
		log.Printf("no se encontró el PID : %d", PID)
		return fmt.Errorf("no se encontró el PID: %d", PID)
	}

	valor, err := BuscarBaseLimitPorPID(PID)
	if err != nil {
		return fmt.Errorf("error al buscar base y límite: %v", err)
	}

	pcbEncontrado.Base = valor.Base
	pcbEncontrado.Limit = valor.Limit
	/*
		tcbMap := mapPCBPorTCB[pcbEncontrado]
		var tcbEncontrada estructuraHilo
		encontrado = false
		for tcb := range tcbMap {
			if tcb.Tid == TID {
				tcbEncontrada = tcb
				encontrado = true
				break
			}
		}
		if !encontrado {
			log.Printf("no se encontró el TID para el PID: %d", PID)
			return fmt.Errorf("no se encontró el TID para el PID: %d", PID)
		}
	*/
	//primero tengo que ver si la direccion que me dieron esta dentro del rango de la particion del pid
	if address < pcbEncontrado.Base || address > pcbEncontrado.Limit {
		log.Printf("dirección fuera de rango para el PID: %d", PID)
		return fmt.Errorf("dirección fuera de rango para el PID: %d", PID)
	}

	espaciodisponible := pcbEncontrado.Limit - address // Espacio disponible desde la dirección hasta el límite
	lengthData := len(data)
	var dataNuevo []byte
	if lengthData >= 4 {
		dataNuevo = data[:4]
	} else {
		dataNuevo = data[:lengthData]
	}
	lengthDataNuevo := len(dataNuevo)
	if espaciodisponible >= 4 {
		copy(globals.MemoriaUsuario[address:address+uint32(lengthDataNuevo)], dataNuevo)
	} else if lengthDataNuevo <= int(espaciodisponible) {
		copy(globals.MemoriaUsuario[address:address+uint32(lengthDataNuevo)], dataNuevo)
	} else {
		copy(globals.MemoriaUsuario[address:address+espaciodisponible], dataNuevo[:espaciodisponible])
	}
	log.Printf("Gets out of Write Memory habiendo escrito %v", dataNuevo)
	return nil
}

//-------------------------------DUMP MEMORY------------------------------------------------

type TCBRequest struct {
	Pid int `json:"pid"`
	Tid int `json:"tid"`
}

func DumpMemory(w http.ResponseWriter, r *http.Request) {

	var tcbReq TCBRequest

	// Decodificar la solicitud JSON
	if err := json.NewDecoder(r.Body).Decode(&tcbReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("PCB: %d, TID: %d - Valores recibidos", tcbReq.Pid, tcbReq.Tid)

	time.Sleep(time.Duration(MemoriaConfig.Delay_Respuesta) * time.Millisecond)

	// Buscar base y límite del proceso
	valor, err := BuscarBaseLimitPorPID(tcbReq.Pid)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error al buscar base y límite: %v", err), http.StatusInternalServerError)
		return
	}

	// Leer datos de memoria
	data := globals.MemoriaUsuario[valor.Base:valor.Limit]
	tamanio := valor.Limit - valor.Base

	informacion := FsInfo{
		Data:          data,
		Tamanio:       tamanio,
		NombreArchivo: GenerarNombreArchivo(tcbReq.Pid, tcbReq.Tid),
	}

	// Convertir a JSON
	body, err := json.Marshal(informacion)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Enviando a memoria write_memory")
	if err := EnviarAModulo(MemoriaConfig.IpFs, MemoriaConfig.PuertoFs, bytes.NewBuffer(body), "dumpMemory"); err != nil {
		http.Error(w, fmt.Sprintf("Error al comunicar con FileSystem: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ok"))
}

type FsInfo struct {
	Data          []byte `json:"data"`
	Tamanio       uint32 `json:"tamanio"`
	NombreArchivo string `json:"nombreArchivo"`
}

func PasarDeUintAByte(num uint32) []byte {
	numEnString := strconv.Itoa(int(num))

	return []byte(numEnString)
}

func EnviarAModulo(ipModulo string, puertoModulo int, body io.Reader, endPoint string) error {

	url := fmt.Sprintf("http://%s:%d/%s", ipModulo, puertoModulo, endPoint)
	resp, err := http.Post(url, "application/json", body)
	if err != nil {
		log.Printf("error enviando mensaje al End point %s - IP:%s - Puerto:%d", endPoint, ipModulo, puertoModulo)
		return err
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("Error al recibir la respuesta del End point %s - IP:%s - Puerto:%d", endPoint, ipModulo, puertoModulo)
		err := fmt.Errorf("%s", resp.Status)
		return err
	}
	return nil
}

func GenerarNombreArchivo(pid int, tid int) string {

	timestamp := time.Now().Format("20060102-150405")

	return fmt.Sprintf("%d-%d-%s.dmp", pid, tid, timestamp)
}

///------------------------------------COMPACTACION--------------------------------------------------

func Compactacion(w http.ResponseWriter, r *http.Request) {

	compactarLasParticiones() //compacto las particiones libres
	actualizarBasesYLímites() //actualizo las bases y limites

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ok"))
	return
}
