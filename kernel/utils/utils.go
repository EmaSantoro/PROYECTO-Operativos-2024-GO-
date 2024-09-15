package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

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
	Pid       int
	Tid       int
	Prioridad int
}

type KernelResponse struct {
	Size int    `json:"size"`
	Path string `json:"path"`
	PCB  PCB    `json:"pcb"`
}

type Hilo struct {
	Path Path `json:"path"`
	Tcb  TCB  `json:"tcb"`
}

/*-------------------- COLAS GLOBALES --------------------*/

var colaNewproceso []PCB
var colaExitproceso []PCB

var colaReadyHilo []TCB
var colaExecHilo []TCB
var colaBlockHilo []TCB
var colaExitHilo []TCB

/*-------------------- MUTEX GLOBALES --------------------*/

var mutexColaNewproceso sync.Mutex
var mutexColaExitproceso sync.Mutex

var mutexColaReadyHilo sync.Mutex
var mutexColaExecHilo sync.Mutex
var mutexColaBlockHilo sync.Mutex
var mutexColaExitHilo sync.Mutex

/*-------------------- VAR GLOBALES --------------------*/

var (
	nextPid = 1
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
	procesoInicial(0, Path{Path: "/procesoInicial"})
}

func procesoInicial(size int, path Path) {
	//CREAMOS PCB
	pcb := createPCB()
	// Verificar si se puede enviar a memoria, si hay espacio para el proceso

	if consultaEspacioAMemoria(size, path, pcb) {
		tcb := createTCB(0) // creamos hilo main
		tcb.Pid = pcb.Pid
		PlanificacionProcesoInicial(path, pcb, tcb)
	} else {
		fmt.Println("No hay espacio en memoria")
		return // obviamente el primer proceso tiene espacio en memoria salvo que sea mas grande que el tamaño de la memoria
	}
}

func consultaEspacioAMemoria(size int, path Path, pcb PCB) bool {
	var memoryRequest KernelResponse
	memoryRequest.Size = size
	memoryRequest.Path = path.Path
	memoryRequest.PCB = pcb

	puerto := globals.ClientConfig.PuertoMemoria
	ip := globals.ClientConfig.IpMemoria

	body, err := json.Marshal(memoryRequest)

	if err != nil {
		log.Printf("error codificando %s", err.Error())
		return false
	}

	url := fmt.Sprintf("http://%s:%d/hayEspacioEnLaMemoria", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))

	if err != nil {
		log.Printf("error enviando tamanio del proceso a ip:%s puerto:%d", ip, puerto)
		return false
	}
	if resp.StatusCode != http.StatusOK {
		return false
	}

	return true
}

func createPCB() PCB {
	nextPid++

	return PCB{
		Pid:   nextPid - 1, // ASIGNO EL VALOR ANTERIOR AL pid
		Tid:   []int{},     // TID
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

func PlanificacionProcesoInicial(path Path, pcb PCB, tcb TCB) {

	mutexColaNewproceso.Lock()
	colaNewproceso = append(colaNewproceso, pcb)
	mutexColaNewproceso.Unlock()

	mutexColaReadyHilo.Lock()
	colaReadyHilo = append(colaReadyHilo, tcb)
	mutexColaReadyHilo.Unlock()

	fmt.Printf(" ## (<PID>:%d) Se crea el proceso - Estado: NEW ##", pcb.Pid)
	fmt.Printf(" ## (<PID>:%d , <TID>:%d ) Se crea el hilo - Estado: READY ##", tcb.Pid, tcb.Tid)

	planificacionCortoPlazo() // envio el hilo main a execute y le mando a cpu su tcb para que ejecute sus instrucciones
	enviarTCB(path, tcb)

}

func planificacionCortoPlazo() {
	if len(colaReadyHilo) > 0 {
		mutexColaReadyHilo.Lock()
		tcb := colaReadyHilo[0]
		colaReadyHilo = colaReadyHilo[1:]
		mutexColaReadyHilo.Unlock()

		mutexColaExecHilo.Lock()
		colaExecHilo = append(colaExecHilo, tcb)
		mutexColaExecHilo.Unlock()

		fmt.Printf(" ## (<PID>:%d , <TID>:%d ) Se mueve a la cola de ejecucion ##", tcb.Pid, tcb.Tid)
	}
}

func enviarTCB(path Path, tcb TCB) error {
	var cpuRequest Hilo
	cpuRequest.Path = path
	cpuRequest.Tcb = tcb

	puerto := globals.ClientConfig.PuertoCpu
	ip := globals.ClientConfig.IpCpu

	body, err := json.Marshal(cpuRequest)

	if err != nil {
		return fmt.Errorf("error codificando %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/recibirTcb", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))

	if err != nil {
		return fmt.Errorf("error enviando tcb a ip:%s puerto:%d", ip, puerto)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error en la respuesta del módulo de cpu: %v", resp.StatusCode)
	}
	return nil
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
