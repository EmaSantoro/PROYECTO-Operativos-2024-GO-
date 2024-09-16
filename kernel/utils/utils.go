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

type IniciarProceso struct {
	Path      string `json:"path"`
	Size      int    `json:"size"`
	Prioridad int    `json:"prioridad"`
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

var mutexProcesosActivos sync.Mutex

/*-------------------- VAR GLOBALES --------------------*/

var (
	nextPid = 1
	nextTid = 0
)

/*---------------------- CANALES ----------------------*/

var finProceso = make(chan bool)

//var procesosActivos = make(map[int]PCB) // mapa que gestiona los procesos activos y se puede acceder a travez de una clave
// en este caso seria por el pid

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

	if ConfigKernel != nil {

		EnviarMensaje(ConfigKernel.IpMemoria, ConfigKernel.PuertoMemoria, "Hola Memoria, Soy Kernel")
		EnviarMensaje(ConfigKernel.IpCpu, ConfigKernel.PuertoCpu, "Hola CPU, Soy Kernel")

		//Cuando levanto kernel se inicia un proceso ppal y luego se ejecutan syscalls?
		procesoInicial("procesoInicial", 0)

		if ConfigKernel.AlgoritmoPlanificacion == "FIFO" {
			go ejecutarHilosFIFO()
		} else if ConfigKernel.AlgoritmoPlanificacion == "PRIORIDADES" {
			// go ejecutarHilosPrioridades()
		} else if ConfigKernel.AlgoritmoPlanificacion == "COLASMULTINIVEL" {
			// go ejecutarHilosColasMultinivel()
		}
	} else {
		log.Printf("Algoritmo de planificacion no valido")
	}
}

func procesoInicial(path string, size int) {
	//CREAMOS PCB
	pcb := createPCB()
	// Verificar si se puede enviar a memoria, si hay espacio para el proceso

	if consultaEspacioAMemoria(size, path, pcb) {
		tcb := createTCB(pcb.Pid, 0)       // creamos hilo main
		pcb.Tid = append(pcb.Tid, tcb.Tid) // agregamos el hilo a la listas de hilos del proceso
		enviarTCBMemoria(tcb)
		PlanificacionProcesoInicial(pcb, tcb)
	} else {
		fmt.Println("El tamaño del proceso inicial es mas grande que la memoria")
		return // obviamente el primer proceso tiene espacio en memoria salvo que sea mas grande que el tamaño de la memoria
	}
}

func consultaEspacioAMemoria(size int, path string, pcb PCB) bool {
	var memoryRequest KernelResponse
	memoryRequest.Size = size
	memoryRequest.Path = path
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
		Pid:   nextPid - 1,
		Tid:   []int{},
		Mutex: []int{},
	}
}

func createTCB(pid int, prioridad int) TCB {
	nextTid++

	return TCB{
		Pid:       pid,
		Tid:       nextTid - 1,
		Prioridad: prioridad,
	}
}

func PlanificacionProcesoInicial(pcb PCB, tcb TCB) {

	mutexColaNewproceso.Lock()
	colaNewproceso = append(colaNewproceso, pcb)
	mutexColaNewproceso.Unlock()

	mutexColaReadyHilo.Lock()
	colaReadyHilo = append(colaReadyHilo, tcb)
	mutexColaReadyHilo.Unlock()

	fmt.Printf(" ## (<PID>:%d) Se crea el proceso - Estado: NEW ##", pcb.Pid)
	fmt.Printf(" ## (<PID>:%d , <TID>:%d ) Se crea el hilo - Estado: READY ##", tcb.Pid, tcb.Tid)
}

func ejecutarHilosFIFO() {
	var Hilo TCB
	for {
		if len(colaReadyHilo) > 0 && len(colaExecHilo) == 0 {
			Hilo = colaReadyHilo[0]
			ejecutarInstruccion(Hilo)
		}
	}
}

func ejecutarInstruccion(Hilo TCB) {
	if len(colaReadyHilo) > 0 && len(colaExecHilo) == 0 {

		mutexColaReadyHilo.Lock()
		colaReadyHilo = colaReadyHilo[1:] // saco el hilo de la cola de ready
		mutexColaReadyHilo.Unlock()

		mutexColaExecHilo.Lock()
		colaExecHilo = append(colaExecHilo, Hilo) // agrego el hilo a la cola de ejecucion
		mutexColaExecHilo.Unlock()

		fmt.Printf(" ## (<PID>:%d , <TID>:%d ) Se ejecuta el hilo - Estado: EXEC ##", Hilo.Pid, Hilo.Tid)

		enviarTCBCpu(Hilo) // envio el hilo a la cpu para que ejecute sus instruciones
	}
}

func enviarTCBCpu(tcb TCB) error {

	cpuRequest := TCB{}
	cpuRequest = tcb

	puerto := globals.ClientConfig.PuertoCpu
	ip := globals.ClientConfig.IpCpu

	body, err := json.Marshal(&cpuRequest)

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

func enviarTCBMemoria(tcb TCB) error {

	memoryRequest := TCB{}
	memoryRequest = tcb

	puerto := globals.ClientConfig.PuertoMemoria
	ip := globals.ClientConfig.IpMemoria

	body, err := json.Marshal(&memoryRequest)

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

func CrearProceso(w http.ResponseWriter, r *http.Request) {
	var proceso IniciarProceso
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&proceso)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	path := proceso.Path
	size := proceso.Size
	prioridad := proceso.Prioridad

	go iniciarProceso(path, size, prioridad) // go routine para que no bloquee el hilo principal en caso de que no haya espacio en memoria para iniciar proceso. CONSULTAR!!

	w.WriteHeader(http.StatusOK)

}

func iniciarProceso(path string, size int, prioridad int) error {

	//CREAMOS PCB
	pcb := createPCB()

	// Verificar si se puede enviar a memoria, si hay espacio para el proceso
	if consultaEspacioAMemoria(size, path, pcb) {
		nextTid = 0
		tcb := createTCB(pcb.Pid, prioridad) // creamos hilo main
		pcb.Tid = append(pcb.Tid, tcb.Tid)   // agregamos el hilo a la listas de hilos del proceso
		enviarTCBMemoria(tcb)

		mutexColaNewproceso.Lock()
		colaNewproceso = append(colaNewproceso, pcb) // agregamos el proceso a la cola de new
		mutexColaNewproceso.Unlock()

		mutexColaReadyHilo.Lock()
		colaReadyHilo = append(colaReadyHilo, tcb) // agregamos el hilo a la cola de ready
		mutexColaReadyHilo.Unlock()

		fmt.Printf(" ## (<PID>:%d) Se crea el proceso - Estado: NEW ##", pcb.Pid)
		fmt.Printf(" ## (<PID>:%d , <TID>:%d ) Se crea el hilo - Estado: READY ##", tcb.Pid, tcb.Tid)

	} else {
		fmt.Println("El tamaño del proceso es mas grande que la memoria, esperando a que finalice otro proceso ....")
		// esperar a que finalize otro proceso y volver a consultar por el espacio en memoria para inicializarlo
		<-finProceso
		iniciarProceso(path, size, prioridad)
	}

	return nil
}

func FinalizarProceso(w http.ResponseWriter, r *http.Request) {
	var hilo TCB
	decoder := json.NewDecoder(r.Body)

	err := decoder.Decode(&hilo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pid := hilo.Pid
	tid := hilo.Tid

	if tid == 0 {
		err = exitProcess(pid)
	} else {
		fmt.Printf("El hilo no es el principal, no se puede ejecutar esta instruccion")
		return //Ver como hacer para que no finalice el kernel y el hilo continue con su siguiente instruccion
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func exitProcess(pid int) error {

	for i, pcb := range colaNewproceso { // lo que hace range es recoreer la lista de procesos y devuelve la posicion en la que esta y el proceso

		if pcb.Pid == pid { // si coincide el pid del proceso con el pid que se quiere finalizar

			mutexColaNewproceso.Lock()
			colaNewproceso = append(colaNewproceso[:i], colaNewproceso[i+1:]...) // sacamos el proceso de la cola de new
			mutexColaNewproceso.Unlock()

			mutexColaExitproceso.Lock()
			colaExitproceso = append(colaExitproceso, pcb) // agregamos el proceso a la cola de exit
			mutexColaExitproceso.Unlock()

			fmt.Printf(" ## finaliza el proceso (<PID>:%d) - Estado: EXIT ##", pcb.Pid)

			// Eliminar todos los hilos del proceso
			for i := len(colaReadyHilo) - 1; i >= 0; i-- { // recorremos la lista de hilos ready
				if colaReadyHilo[i].Pid == pid { // si el pid del hilo coincide con el pid del proceso
					mutexColaReadyHilo.Lock()
					colaReadyHilo = append(colaReadyHilo[:i], colaReadyHilo[i+1:]...) // sacamos el hilo de la cola de ready
					mutexColaReadyHilo.Unlock()

					mutexColaExitHilo.Lock()
					colaExitHilo = append(colaExitHilo, colaReadyHilo[i]) // agregamos el hilo a la cola de exit
					mutexColaExitHilo.Unlock()

					fmt.Printf(" ## finaliza el hilo (<PID>:%d , <TID>:%d ) - Estado: EXIT ##", colaReadyHilo[i].Pid, colaReadyHilo[i].Tid)
				}
			}

			for i := len(colaExecHilo) - 1; i >= 0; i-- { // recorremos la lista de hilos en ejecucion
				if colaExecHilo[i].Pid == pid { // si el pid del hilo coincide con el pid del proceso
					mutexColaExecHilo.Lock()
					colaExecHilo = append(colaExecHilo[:i], colaExecHilo[i+1:]...) // sacamos el hilo de la cola de ejecucion
					mutexColaExecHilo.Unlock()

					mutexColaExitHilo.Lock()
					colaExitHilo = append(colaExitHilo, colaReadyHilo[i]) // agregamos el hilo a la cola de exit
					mutexColaExitHilo.Unlock()

					fmt.Printf(" ## finaliza el hilo (<PID>:%d , <TID>:%d ) - Estado: EXIT ##", colaReadyHilo[i].Pid, colaReadyHilo[i].Tid)
				}
			}

			for i := len(colaBlockHilo) - 1; i >= 0; i-- { // recorremos la lista de hilos bloqueados
				if colaBlockHilo[i].Pid == pid { // si el pid del hilo coincide con el pid del proceso
					mutexColaBlockHilo.Lock()
					colaBlockHilo = append(colaBlockHilo[:i], colaBlockHilo[i+1:]...) // sacamos el hilo de la cola de bloqueados
					mutexColaBlockHilo.Unlock()

					mutexColaExitHilo.Lock()
					colaExitHilo = append(colaExitHilo, colaReadyHilo[i]) // agregamos el hilo a la cola de exit
					mutexColaExitHilo.Unlock()

					fmt.Printf(" ## finaliza el hilo (<PID>:%d , <TID>:%d ) - Estado: EXIT ##", colaReadyHilo[i].Pid, colaReadyHilo[i].Tid)
				}
			}

			resp := enviarProcesoFinalizadoAMemoria(pcb)

			if resp == nil {
				//mutexProcesosActivos.Lock()
				//defer mutexProcesosActivos.Unlock()

				// eliminar el PCB del proceso terminado
				//delete(procesosActivos, pid)

				// Notificar a través del canal
				finProceso <- true
			}
		}
	}

	return nil
}

func enviarProcesoFinalizadoAMemoria(pcb PCB) error {

	memoryRequest := PCB{}
	memoryRequest = pcb

	puerto := globals.ClientConfig.PuertoMemoria
	ip := globals.ClientConfig.IpMemoria

	body, err := json.Marshal(&memoryRequest)

	if err != nil {
		return fmt.Errorf("error codificando %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/finalizacionProceso", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))

	if err != nil {
		return fmt.Errorf("error enviando tcb a ip:%s puerto:%d", ip, puerto)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error en la respuesta del módulo de cpu: %v", resp.StatusCode)
	}
	return nil

}

func crearHilo(w http.ResponseWriter, r *http.Request) {

}

func finalizarHilo(w http.ResponseWriter, r *http.Request) {

}

func unirHilo(w http.ResponseWriter, r *http.Request) {

}

func cancelarHilo(w http.ResponseWriter, r *http.Request) {

}

func crearMutex(w http.ResponseWriter, r *http.Request) {

}

func bloquearMutex(w http.ResponseWriter, r *http.Request) {

}

func liberarMutex(w http.ResponseWriter, r *http.Request) {

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
