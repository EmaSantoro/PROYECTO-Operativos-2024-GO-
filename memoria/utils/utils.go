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

/// ARCHIVOS DE PSEUDOCODIGO PASANDOLO AL MAP 

func SetInstructionsFromFileToMap(w http.ResponseWriter, r *http.Request) {
	// Extraer los parámetros PID, TID y path del archivo
	queryParams := r.URL.Query()
	pid, _ := strconv.Atoi(queryParams.Get("pid"))
	tid, _ := strconv.Atoi(queryParams.Get("tid"))
	path := queryParams.Get("path")

	// Abrir el archivo de pseudocódigo
	readFile, err := os.Open(path)
	if err != nil {
		http.Error(w, "Error opening file", http.StatusInternalServerError)
		return
	}
	defer readFile.Close()

	// Crear un escáner para leer el archivo línea por línea
	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)

	var instrucciones []string // Almacenar cada instrucción en un slice de strings
	for fileScanner.Scan() {
		instrucciones = append(instrucciones, fileScanner.Text()) //esta linea lee los codigos
	}

	// Verificar si el PID ya existe en el mapa
	if _, found := mapPidPorHilos[pid]; !found {
		// Si no existe, crear un nuevo mapa para el PID
		mapPidPorHilos[pid] = make(map[int][]string)
	}

	// Guardar las instrucciones en el mapa correspondiente al PID y TID
	mapPidPorHilos[pid][tid] = instrucciones

	// Responder con éxito
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Instructions loaded successfully"))
}

///-------------FUNCIONES PARA DARLE LAS INSTRUCCIONES A CPU-------------------

type InstructionResponse struct {
	Instruction string `json:"instruction"`
}

var mapPidPorHilos = make(map[int]map[int][]string)

func GetInstruction(w http.ResponseWriter, r *http.Request) {
    
	queryParams := r.URL.Query() //para obtener los datos de la URL
	pid, _ := strconv.Atoi(queryParams.Get("pid")) // pid
	tid, _ := strconv.Atoi(queryParams.Get("tid")) // tid
	pc, _ := strconv.Atoi(queryParams.Get("pc")) // pc 

    // Acceder al mapa de instrucciones almacenadas por PID y TID
	if tidMap, foundPid := mapPidPorHilos[pid]; foundPid {
		// Acceder al mapa interno por TID
		if instrucciones, foundTid := tidMap[tid]; foundTid {
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
				// Escribe la instrucción directamente como bytes
				w.Write([]byte(instruccion))
			}
		} else {
			// Si no se encuentra el TID
			http.Error(w, "TID not found", http.StatusNotFound)
			fmt.Println("No se encontró el TID")
			return
		}
	} else {
		// Si no se encuentra el PID
		http.Error(w, "PID not found", http.StatusNotFound)
		fmt.Println("No se encontró el PID")
		return
	}

	// Si no se encuentra la instrucción, devolver error
	http.Error(w, "Instruction not found", http.StatusNotFound)
}