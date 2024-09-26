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
	"time"

	"github.com/sisoputnfrba/tp-golang/memoria/globals"
)

type Mensaje struct {
	Mensaje string `json:"mensaje"`
}

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

func init() {

	//ConfigMemoria := IniciarConfiguracion("configsMemoria/config.json")
	//EnviarMensaje(ConfigMemoria.IpKernel, ConfigMemoria.PuertoKernel, "Hola Kernel, Soy Memoria")
	//EnviarMensaje(ConfigMemoria.IpFs, ConfigMemoria.PuertoFs, "Hola FS, Soy Memoria")
	//EnviarMensaje(ConfigMemoria.IpCpu, ConfigMemoria.PuertoCpu, "Hola CPU, Soy Memoria")
}

func ConfigurarLogger() {
	logFile, err := os.OpenFile("tp.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
}

func RecibirMensaje(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	var mensaje Mensaje
	err := decoder.Decode(&mensaje)

	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	log.Println("Conexion con Memoria")
	log.Printf("%+v\n", mensaje)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func EnviarMensaje(ip string, puerto int, mensajeTxt string) {
	mensaje := Mensaje{Mensaje: mensajeTxt}
	body, err := json.Marshal(mensaje)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/mensaje", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", ip, puerto)
	}
	/*defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return
	}*/
	log.Printf("respuesta del servidor: %s", resp.Status)
}

///////////////////////////////////////////////////////////////////////////////

type PCB struct {  //NO ES LA MISMA PCB QUE TIENE KERNEL DIGAMOS ES UNA PROPIA DE MEMORIA
	pid   int
	base uint32
	limit uint32
}

// // Mapa anidado que almacena los contextos de ejecución
// var mapPCBPorTCB = make(map[PCB]map[TCB][]string)

///--------------------------------------------GET INSTRUCTION---------------------------------------------

type InstructionResponse struct {
	Instruction string `json:"instruction"`
}

func GetInstruction(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	pid, _ := strconv.Atoi(queryParams.Get("pid")) // pid
	tid, _ := strconv.Atoi(queryParams.Get("tid")) // tid
	pc, _ := strconv.Atoi(queryParams.Get("pc"))   // pc

	time.Sleep(time.Duration(globals.ClientConfig.Delay_Respuesta) * time.Millisecond)

	for pcb, tidMap := range mapPCBPorTCB {
		if pcb.pid == pid {
			for tcb, instrucciones := range tidMap {
				if tcb.tid == tid {
					if pc >= 0 && pc < len(instrucciones) {

						instruccion := instrucciones[pc]

						instructionResponse := InstructionResponse{
							Instruction: instruccion,
						}

						// Log de obtención de instrucción
                        log.Printf("## Obtener instrucción - (PID:TID) - (%d:%d) - Instrucción: %s", pid, tid, instruccion)

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

//------------------------------------ GET EXECUTION CONTEXT ---------------------------------------------

func GetExecutionContext(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	pid, _ := strconv.Atoi(queryParams.Get("pid"))
	tid, _ := strconv.Atoi(queryParams.Get("tid"))

	time.Sleep(time.Duration(globals.ClientConfig.Delay_Respuesta) * time.Millisecond)

	for pcb, tidMap := range mapPCBPorTCB {
		if pcb.pid == pid {
			for tcb := range tidMap {
				if tcb.tid == tid {
					executionContext := struct {
						PCB
						estructuraHilo
					}{
						PCB:            pcb,
						estructuraHilo: tcb,
					}

					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(executionContext)
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
	queryParams := r.URL.Query()
	pid, _ := strconv.Atoi(queryParams.Get("pid")) //esto me parece que no va
	tid, _ := strconv.Atoi(queryParams.Get("tid")) //esto tampoco 

	var newContext struct {
		PCB struct {
			base  uint32 `json:"base"`
			limit uint32 `json:"limit"`
		}
		estructuraHilo struct {
			AX uint32 `json:"AX"`
			BX uint32 `json:"BX"`
			CX uint32 `json:"CX"`
			DX uint32 `json:"DX"`
			EX uint32 `json:"EX"`
			FX uint32 `json:"FX"`
			GX uint32 `json:"GX"`
			HX uint32 `json:"HX"`
			PC uint32 `json:"PC"`
		}
	}

	err := json.NewDecoder(r.Body).Decode(&newContext)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	time.Sleep(time.Duration(globals.ClientConfig.Delay_Respuesta) * time.Millisecond)

	for pcb, tidMap := range mapPCBPorTCB {
		if pcb.pid == pid {
			for tcb := range tidMap {
				if tcb.tid == tid {

					// Actualizar los valores del PCB (Base y Limit)
					pcb.base = newContext.PCB.base
					pcb.limit = newContext.PCB.limit

					// Actualizar los registros del TCB
					tcb.AX = uint32(newContext.estructuraHilo.AX)
					tcb.BX = uint32(newContext.estructuraHilo.BX)
					tcb.CX = uint32(newContext.estructuraHilo.CX)
					tcb.DX = uint32(newContext.estructuraHilo.DX)
					tcb.EX = uint32(newContext.estructuraHilo.EX)
					tcb.FX = uint32(newContext.estructuraHilo.FX)
					tcb.GX = uint32(newContext.estructuraHilo.GX)
					tcb.HX = uint32(newContext.estructuraHilo.HX)
					tcb.PC = uint32(newContext.estructuraHilo.PC)

					//CREO QUE NO SE ESTA ACTUALIZANDO EL MAPA ANIDADO
					tidMap[tcb] = tidMap[tcb]
					mapPCBPorTCB[pcb] = tidMap

					w.WriteHeader(http.StatusOK)
					w.Write([]byte("Execution context updated successfully"))
					return
				}
			}
			http.Error(w, "TID not found", http.StatusNotFound)
			return
		}
	}
	http.Error(w, "PID not found", http.StatusNotFound)
}

//-----------------------------------------CREATE PROCESS-------------------------------------------

// Mapa anidado que almacena los contextos de ejecución
var mapPCBPorTCB = make(map[PCB]map[estructuraHilo][]string) //ESTE ES EL PRINCIPAL DIGAMOS 
var mapParticiones []bool                          //estado de las particiones ocupada/libre
var particiones = globals.ClientConfig.Particiones //vector de particiones, aca tengo los tamaños en int
type Process struct {
	size int `json:"size"`
	pid  int `json:"pcb"`
}


func CreateProcess(w http.ResponseWriter, r *http.Request) { //recibe la pid y el size
	var process Process
	time.Sleep(time.Duration(globals.ClientConfig.Delay_Respuesta) * time.Millisecond)

	if err := json.NewDecoder(r.Body).Decode(&process); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pcb := PCB{ //creo la estructura necesaria
		pid: process.pid,
		base: 0,
		limit: 0,
	}

	if globals.ClientConfig.EsquemaMemoria == "FIJAS" {

		numeroDeParticion := asignarPorAlgoritmo(globals.ClientConfig.AlgoritmoBusqueda, process.size) //asigno por algoritmo

		if numeroDeParticion == -1 {
			http.Error(w, "No hay espacio en la memoria", http.StatusConflict)
			return
		}

		//BASE
		var baseEnInt int
		pcb.base = 0
        for i := 0; i < numeroDeParticion; i++ {
            baseEnInt += globals.ClientConfig.Particiones[i] //tengo que ver tema int y uint32
			pcb.base = uint32(baseEnInt)
        }

		//LIMIT
		var limitEnInt int
		limitEnInt = baseEnInt + globals.ClientConfig.Particiones[numeroDeParticion] - 1
		pcb.limit = uint32(limitEnInt)

		mapParticiones[numeroDeParticion] = true //marcar particion como ocupada

		if err := guardarPCBenMapConRespectivaParticion(pcb, numeroDeParticion); err != nil { //GUARDO EN EL MAP pcb, y el numero de particion
			http.Error(w, err.Error(), http.StatusInternalServerError) //MII MAP DE PCB X NMRO DE PARTICION 
			return
		}

		if err := guardarPCBEnElMap(pcb); err != nil { //ACA ESTOY GUARDANDO LA PCB EN MI MAP PRINCIPAL EL MAS IMPORTANTE DE TODOS
			http.Error(w, err.Error(), http.StatusInternalServerError) 
			return
		}

		// Log de creación de proceso
        log.Printf("## Proceso Creado - PID: %d - Tamaño: %d", process.pid, process.size)


		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Ok"))
		return
	} else {
		//PARTICION DINAMICA
	}
}

var mapPCBPorParticion = make(map[PCB]int)

func guardarPCBenMapConRespectivaParticion(pcb PCB, numeroDeParticion int) error {
	mapPCBPorParticion[pcb] = numeroDeParticion
	return nil
}

func guardarPCBEnElMap(pcb PCB)error{
    
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
// var particiones = globals.ClientConfig.Particiones //vector de particiones, aca tengo los tamaños en int

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

// 	time.Sleep(time.Duration(globals.ClientConfig.Delay_Respuesta) * time.Millisecond)

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

	if globals.ClientConfig.EsquemaMemoria == "FIJAS" { //PARA FIJAS
		var numeroDeParticion int
		encontrado := false
		for pcb, particion := range mapPCBPorParticion {
			if pcb.pid == pid {
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

		delete(mapPCBPorParticion, PCB{pid: pid}) // elimino la estructura del pcb en el map de particiones
		delete(mapPCBPorTCB, PCB{pid: pid})      // elimino el pcb del map anidado

		// Log de destrucción de proceso
        log.Printf("## Proceso Destruido - PID: %d - Tamaño: %d", pid, globals.ClientConfig.Particiones[numeroDeParticion])

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Proceso finalizado exitosamente"))
	}
	//FALTA PARA LA PARTICION DINAMICA
}

//-----------------------------------------CREATE THREAD--------------------------------------------

type Thread struct {
	Pid  int    `json:"pid"`
	Tid  int    `json:"tid"`
	Path string `json:"path"`
}

type estructuraHilo struct {
	pid int
	tid int
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

func CreateThread(w http.ResponseWriter, r *http.Request) {
	var thread Thread
	time.Sleep(time.Duration(globals.ClientConfig.Delay_Respuesta) * time.Millisecond)
	if err := json.NewDecoder(r.Body).Decode(&thread); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	TCB := estructuraHilo{ //creo la estructura necesaria
		pid: thread.Pid,
		tid: thread.Tid,
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

	if err := guardarTodoEnElMap(thread.Pid, TCB, thread.Path); err != nil { //GUARDO EN EL MAP
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Log de creación de hilo
    log.Printf("## Hilo Creado - (PID:TID) - (%d:%d)", thread.Pid, thread.Tid)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ok"))
	return
}


func guardarTodoEnElMap(pid int, TCB estructuraHilo, path string) error {

	// Abro el archivo de pseudocódigo
	readFile, err := os.Open(path)
	if err != nil {
		log.Printf("Error: PATH %s opening file", path)
		return err
	}
	defer readFile.Close()

	time.Sleep(time.Duration(globals.ClientConfig.Delay_Respuesta) * time.Millisecond)

	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)

	var instrucciones []string // Almaceno cada instrucción en un slice de strings
	for fileScanner.Scan() {
		instrucciones = append(instrucciones, fileScanner.Text()) //esta linea lee los codigos
	}

	var pcbEncontrado PCB //para encontrar el pcb y poder entrar al mapa anidado
	for pcb := range mapPCBPorTCB {
		if pcb.pid == pid {
			pcbEncontrado = pcb
			break
		}
	}

	if _, found := mapPCBPorTCB[pcbEncontrado]; !found {
		mapPCBPorTCB[pcbEncontrado] = make(map[estructuraHilo][]string)
	}

	mapPCBPorTCB[pcbEncontrado][TCB] = instrucciones

	return nil

}

//---------------------------------------TERMINATE THREAD--------------------------------------------

type Req struct {
	Pid int `json:"pid"`
	Tid int `json:"tid"`
}

func TerminateThread(w http.ResponseWriter, r *http.Request) {

	var req Req
	time.Sleep(time.Duration(globals.ClientConfig.Delay_Respuesta) * time.Millisecond)

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if _, exists := mapPCBPorTCB[PCB{pid: req.Pid}]; !exists {
		http.Error(w, "No se pudo encontrar el proceso", http.StatusNotFound)
		return
	}

	if tcbMap, found := mapPCBPorTCB[PCB{pid: req.Pid}]; found {
		delete(tcbMap, estructuraHilo{pid: req.Pid, tid: req.Tid})
		if len(tcbMap) == 0 {
			delete(mapPCBPorTCB, PCB{pid: req.Pid}) //por si llega a quedar vacio
		}
	}

	// Log de destrucción de hilo
    log.Printf("## Hilo Destruido - (PID:TID) - (%d:%d)", req.Pid, req.Tid)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

//-----------------------------------------READ MEMORY-------------------------------------------

// type MemoryRequest struct{
// 	PID int `json:"pid"`
// 	Address uint32 `json:"address"`
// }

// func ReadMemoryHandler(w http.ResponseWriter, r *http.Request) {
//     var memReq MemoryRequest
//     time.Sleep(time.Duration(globals.ClientConfig.Delay_Respuesta) * time.Millisecond)

//     if err := json.NewDecoder(r.Body).Decode(&memReq); err != nil {
//         http.Error(w, err.Error(), http.StatusBadRequest)
//         return
//     }

//     data, err := ReadMemory(memReq.PID, memReq.Address)
//     if err != nil {
//         http.Error(w, err.Error(), http.StatusInternalServerError)
//         return
//     }

//     if memReq.Type == "CPU" {
//         if err := sendDataToCPU(data); err != nil {
//             http.Error(w, "Error al enviar los datos a la CPU", http.StatusInternalServerError)
//             return
//         }
//     }
// }

// func ReadMemory()

// func sendDataToCPU()

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

// 	time.Sleep(time.Duration(globals.ClientConfig.Delay_Respuesta) * time.Millisecond)

// 	for pcb, tidMap := range mapPCBPorTCB {

	
// }