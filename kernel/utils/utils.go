package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/sisoputnfrba/tp-golang/kernel/globals"
)

/*---------------------- ESTRUCTURAS ----------------------*/

type Mensaje struct {
	Mensaje string `json:"mensaje"`
}

type Path struct {
	Path string `json:"path"`
}

type PCB struct {
	Pid   int
	Tid   []int
	Mutex []int
}

type TCB struct {
	Pid 	 int
	Tid       int
	Prioridad int
}



var colaNewproceso []PCB
var colaExitproceso []PCB

var colaReadyHilo []TCB
var colaExecHilo []TCB
var colaBlockHilo []TCB
var colaExitHilo []TCB

/*-------------------- VAR GLOBALES --------------------*/

var (
	nextPid = 1
)
var (
	nextTid = 0
)

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

// Iniciar modulo
func init() {
	ConfigKernel := IniciarConfiguracion("configsKERNEL/config.json")
	EnviarMensaje(ConfigKernel.IpMemoria, ConfigKernel.PuertoMemoria, "Hola Memoria, Soy Kernel")
	EnviarMensaje(ConfigKernel.IpCpu, ConfigKernel.PuertoCpu, "Hola CPU, Soy Kernel")

	//Cuando levanto kernel se inicia un proceso ppal y luego se ejecutan syscalls?

}

func iniciarProceso(w http.ResponseWriter, r *http.Request) {

	var path Path

	decoder := json.NewDecoder(r.Body)

	err := decoder.Decode(&path)

	if err != nil {
		http.Error(w, "Error decoding JSON data", http.StatusInternalServerError)
		return
	}

	//CREAMOS PCB
	pcb := createPCB()
	// Verificar si se puede enviar a memoria, si hay espacio para el proceso
	// como averiguo el tamanio del archivo
	tcb := createTCB(0) // creamos hilo main
	tcb.Pid = pcb.Pid
	pcb.Tid = append(pcb.Tid, tcb.Tid)
	//enviarPathMemoria(path , tamanioProceso)
	//enviarPcbTcbCPU(pcb , tcb)

	iniciarPlanificacion(path, pcb, tcb)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))

}

func createPCB() PCB {
	nextPid++

	return PCB{
		Pid:   nextPid - 1, // ASIGNO EL VALOR ANTERIOR AL pid
		Tid:   []int{},    // TID
		Mutex: []int{},     // Mutex
	}
}

func createTCB(prioridad int) TCB {
	nextTid++

	return TCB{
		Pid:       0,
		Tid:       nextTid - 1,
		Prioridad: prioridad,
	}
}

func iniciarPlanificacion(path Path, pcb PCB, tcb TCB) { // preguntar si colas de los distintos estados son para los procesos o hilos o ambos


	colaNewproceso = append(colaNewproceso, pcb)

	colaReadyHilo = append(colaReadyHilo, tcb)

	fmt.Printf(" ## (<PID>:%d) Se crea el proceso - Estado: NEW ##", pcb.Pid)
	fmt.Printf(" ## (<PID>:%d , <TID>:%d ) Se crea el hilo - Estado: READY ##", tcb.Pid, tcb.Tid)

	//enviarPathMemoria(proceso , hilo)

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

	log.Printf("respuesta del servidor: %s", resp.Status)
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

	log.Println("Conexion con Kernel")
	log.Printf("%+v\n", mensaje)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
