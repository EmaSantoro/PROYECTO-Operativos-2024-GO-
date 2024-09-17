package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
	"bufio"

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


//-----------------------PROBAR COMO TENDRIA QUE HACER TIPO RECIBO PCB, TCB y PATH---------------------------

type PCB struct { 
   pid int 
   base int
   limit int 
}

type TCB struct {
	pid int
	tid int
	AX int
	BX int
	CX int
	DX int 
	EX int 
	FX int
	GX int
	HX int
	PC int 
} 

// Mapa anidado que almacena los contextos de ejecución
var mapPCBPorTCB = make(map[PCB]map[TCB][]string)

// func SetInstructionsFromFileToMap(w http.ResponseWriter, r *http.Request) {
// 	// Extraer los parámetros PCB, TCB y PATH del archivo

// 	queryParams := r.URL.Query()
// 	path := queryParams.Get("path")
// 	var pcb PCB
// 	var tcb TCB 
	
// 	// Abrir el archivo de pseudocódigo
// 	readFile, err := os.Open(path)
// 	if err != nil {
// 		http.Error(w, "Error opening file", http.StatusInternalServerError)
// 		return
// 	}
// 	defer readFile.Close()

	
//     time.Sleep(time.Duration(globals.ClientConfig.Delay_Respuesta) * time.Millisecond)

// 	// Crear un escáner para leer el archivo línea por línea
// 	fileScanner := bufio.NewScanner(readFile)
// 	fileScanner.Split(bufio.ScanLines)

// 	var instrucciones []string // Almacenar cada instrucción en un slice de strings
// 	for fileScanner.Scan() {
// 		instrucciones = append(instrucciones, fileScanner.Text()) //esta linea lee los codigos
// 	}

// 	// Verificar si el PCb ya existe en el mapa
// 	if _, found := mapPCBPorTCB[pcb]; !found {
// 		mapPCBPorTCB[pcb] = make(map[TCB][]string)
// 	}

// 	// Guardar las instrucciones en el mapa correspondiente al PID y TID
// 	mapPCBPorTCB[pcb][tcb] = instrucciones

// 	// Responder con éxito
// 	w.WriteHeader(http.StatusOK)
// 	w.Write([]byte("Instructions loaded successfully"))
// }

///--------------------------------------------GET INSTRUCTION---------------------------------------------

type InstructionResponse struct {
   Instruction string `json:"instruction"`
}

func GetInstruction(w http.ResponseWriter, r *http.Request) {
    queryParams := r.URL.Query() //para obtener los datos de la URL
	pid, _ := strconv.Atoi(queryParams.Get("pid")) // pid
	tid, _ := strconv.Atoi(queryParams.Get("tid")) // tid
	pc, _ := strconv.Atoi(queryParams.Get("pc")) // pc 

    time.Sleep(time.Duration(globals.ClientConfig.Delay_Respuesta) * time.Millisecond)

	// Buscar el PCB correspondiente al PID
	for pcb, tidMap := range mapPCBPorTCB {
		if pcb.pid == pid { // Encontramos el PCB que coincide con el PID

			// Buscar el TCB correspondiente al TID dentro de ese PCB
			for tcb, instrucciones := range tidMap {
				if tcb.tid == tid { // Encontramos el TCB que coincide con el TID

					// Verificar que el PC esté dentro del rango válido de instrucciones
					if pc >= 0 && pc < len(instrucciones) {
						// Obtener la instrucción correspondiente al PC
						instruccion := instrucciones[pc]

						// Simular el retardo configurado en el archivo de configuración
						time.Sleep(time.Duration(globals.ClientConfig.Delay_Respuesta) * time.Millisecond)

						// Construir la respuesta
						instructionResponse := InstructionResponse{
							Instruction: instruccion,
						}

						// Enviar la respuesta en formato JSON
						json.NewEncoder(w).Encode(instructionResponse)
						w.Write([]byte(instruccion)) // Escribir la instrucción
						return
					} else {
						// Si el PC está fuera del rango de instrucciones
						http.Error(w, "PC out of range", http.StatusBadRequest)
						fmt.Println("PC fuera del rango de instrucciones")
						return
					}
				}
			}
			// Si no se encuentra el TCB correspondiente
			http.Error(w, "TID not found", http.StatusNotFound)
			fmt.Println("No se encontró el TID")
			return
		}
	}
	// Si no se encuentra el PCB correspondiente
	http.Error(w, "PID not found", http.StatusNotFound)
	fmt.Println("No se encontró el PID")
}

//------------------------------------ GET EXECUTION CONTEXT --------------------------------------------- 

func GetExecutionContext(w http.ResponseWriter, r *http.Request) {
	// Extraer los parámetros PID y TID de la URL
	queryParams := r.URL.Query()
	pid, _ := strconv.Atoi(queryParams.Get("pid"))
	tid, _ := strconv.Atoi(queryParams.Get("tid"))

    time.Sleep(time.Duration(globals.ClientConfig.Delay_Respuesta) * time.Millisecond)

	// Buscar el PCB correspondiente al PID
	for pcb, tidMap := range mapPCBPorTCB {
		if pcb.pid == pid { // Encontramos el PCB que coincide con el PID

			// Iterar sobre el mapa de TCBs y buscar por `tid`
			for tcb := range tidMap {
				if tcb.tid == tid { // Encontramos el TCB que coincide con el TID

					// Construir la respuesta con el contexto completo
					executionContext := struct {
						PCB
						TCB
					}{
						PCB: pcb,
						TCB: tcb,
					}

					// Enviar la respuesta en formato JSON
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(executionContext)
					return
				}
			}

			// Si no se encuentra el TCB correspondiente
			http.Error(w, "TID not found", http.StatusNotFound)
			return
		}
	}

	// Si no se encuentra el PCB correspondiente
	http.Error(w, "PID not found", http.StatusNotFound)
} 

//-------------------------------- UPDATE EXECUTION CONTEXT-----------------------------------------------

// UpdateExecutionContext: Actualizar el contexto de ejecución para un TID específico
func UpdateExecutionContext(w http.ResponseWriter, r *http.Request) {
	// Extraer los parámetros PID y TID de la URL
	queryParams := r.URL.Query()
	pid, _ := strconv.Atoi(queryParams.Get("pid"))
	tid, _ := strconv.Atoi(queryParams.Get("tid"))

	// Extraer los nuevos valores para el PCB y TCB desde el cuerpo de la petición
	var newContext struct {
		PCB struct {
		    Base  int `json:"base"`
		 	Limit int `json:"limit"`
		}
		TCB struct {
			AX int `json:"AX"`
			BX int `json:"BX"`
			CX int `json:"CX"`
			DX int `json:"DX"`
			EX int `json:"EX"`
			FX int `json:"FX"`
			GX int `json:"GX"`
			HX int `json:"HX"`
			PC int `json:"PC"`
		}
	}

	// Decodificar el JSON del cuerpo de la petición
	err := json.NewDecoder(r.Body).Decode(&newContext)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

    time.Sleep(time.Duration(globals.ClientConfig.Delay_Respuesta) * time.Millisecond)

	// Buscar el PCB correspondiente al PID
	for pcb, tidMap := range mapPCBPorTCB {
		if pcb.pid == pid { // Encontramos el PCB que coincide con el PID

			// Buscar el TCB correspondiente al TID dentro de ese PCB
			for tcb := range tidMap {
				if tcb.tid == tid { // Encontramos el TCB que coincide con el TID

					// Actualizar los valores del PCB (Base y Limit)
					pcb.base = newContext.PCB.Base
					pcb.limit = newContext.PCB.Limit

					// Actualizar los registros del TCB
					tcb.AX = newContext.TCB.AX
					tcb.BX = newContext.TCB.BX
					tcb.CX = newContext.TCB.CX
					tcb.DX = newContext.TCB.DX
					tcb.EX = newContext.TCB.EX
					tcb.FX = newContext.TCB.FX
					tcb.GX = newContext.TCB.GX
					tcb.HX = newContext.TCB.HX
					tcb.PC = newContext.TCB.PC
				
			
					// Responder con éxito
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("Execution context updated successfully"))
					return
				}
			}

			// Si no se encuentra el TCB correspondiente
			http.Error(w, "TID not found", http.StatusNotFound)
			return
		}
	}

	// Si no se encuentra el PCB correspondiente
	http.Error(w, "PID not found", http.StatusNotFound)
}


// func HayEspacioEnLaMemoria(w http.ResponseWriter, r *http.Request){

// 	queryParams := r.URL.Query()
// 	size, _ := strconv.Atoi(queryParams.Get("size"))

// 	// Verificar si el esquema de particionamiento es FIJAS o VARIABLE
// 	if globals.ClientConfig.EsquemaMemoria == "FIJAS" {
// 		// Si es FIJAS, buscar un espacio libre en las particiones fijas
// 		for _, particion := range globals.ClientConfig.Particiones {
// 			if particion.Tamanio >= size && particion.Estado == 0 {
// 				// Si se encuentra un espacio libre, responder con éxito
// 				w.WriteHeader(http.StatusOK)
// 				w.Write([]byte("Ok"))
// 				return
// 			}
// 		}
// 	} else {
// 		// si es VARIABLE buscar un espacio libre en la memoria
// 	}

// 	// Si no se encontró espacio libre, responder con error
// 	http.Error(w, "Memoria sin Almacenamiento", http.StatusConflict)
// }

//------------------------HAY ESPACIO EN LA MEMORIA-------------------------------------------

func HayEspacioEnLaMemoria(w http.ResponseWriter, r *http.Request){

	queryParams := r.URL.Query()
	size, _ := strconv.Atoi(queryParams.Get("size"))
	path := queryParams.Get("path")
	var pcb PCB
	var tcb TCB

	time.Sleep(time.Duration(globals.ClientConfig.Delay_Respuesta) * time.Millisecond)

	// Verificar si el esquema de particionamiento es FIJAS o VARIABLE
	if globals.ClientConfig.EsquemaMemoria == "FIJAS" {
		// Si es FIJAS, buscar un espacio libre en las particiones fijas
		for _, particion := range globals.ClientConfig.Particiones {
			if particion.Tamanio >= size && particion.Estado == 0 {
				// AGINAR ESPACIO EN LA MEMORIA DE USUARIO (RESERVAR LUGAR)
			    if err := guardarTodoEnElMap(pcb, tcb, path); err != nil { //GUARDO EN EL MAP
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Ok"))
				return
			}
		}
	} else {
		// si es VARIABLE buscar un espacio libre en la memoria
	}

	// Si no se encontró espacio libre, responder con error
	http.Error(w, "Memoria sin Almacenamiento", http.StatusConflict)
}

func guardarTodoEnElMap(pcb PCB, tcb TCB, path string) error{
	
	// Abrir el archivo de pseudocódigo
	readFile, err := os.Open(path)
	if err != nil {
		log.Printf("Error: PATH %s opening file", path)
		return err
	}
	defer readFile.Close()
	
	time.Sleep(time.Duration(globals.ClientConfig.Delay_Respuesta) * time.Millisecond)

	// Crear un escáner para leer el archivo línea por línea
	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)

	var instrucciones []string // Almacenar cada instrucción en un slice de strings
	for fileScanner.Scan() {
		instrucciones = append(instrucciones, fileScanner.Text()) //esta linea lee los codigos
	}

	// Verificar si el PCb ya existe en el mapa
	if _, found := mapPCBPorTCB[pcb]; !found {
		mapPCBPorTCB[pcb] = make(map[TCB][]string)
	}

	// Guardar las instrucciones en el mapa correspondiente al PID y TID
	mapPCBPorTCB[pcb][tcb] = instrucciones

	return nil

}

//1. Primer paso: kernel me tendria que pasar pid, tid, base y limite
//2. tengo lugar? 
//2. Segundo paso: yo tendria que sacar del path de mi configuracion el path del archivo
//3. kernel abre el archivo y me pasa el path. 

//kernel me pasa path,pcb y tamaño 
