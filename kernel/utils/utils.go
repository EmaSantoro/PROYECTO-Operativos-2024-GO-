package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
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

type KernelRequest struct {
	Size int    `json:"size"`
	PCB  PCB    `json:"pcb"`
}

type TCBRequestCpu struct {
	Pid       int `json:"pid"`
	Tid       int `json:"tid"`
}

type TCBRequestMemory struct {
	Pid       int `json:"pid"`
	Tid       int `json:"tid"`
	Path 	string `json:"path"`
}

type IniciarProcesoResponse struct {
	Path      string `json:"path"`
	Size      int    `json:"size"`
	Prioridad int    `json:"prioridad"`
}

type FinalizarProcesoResponse struct {
	Pid int `json:"pid"`
	Tid int `json:"tid"`
}

type PCBRequest struct {
	Pid int `json:"pid"`
	Tid []int `json:"tids"`
	Mutex []int `json:"mutex"`
}

type CrearHiloResponse struct {
	Pid int `json:"pid"`
	Path string `json:"path"`
	Prioridad int `json:"prioridad"`	
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

//var mutexProcesosActivos sync.Mutex

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
	slog.SetLogLoggerLevel(slog.LevelDebug)
	/*	slog.SetLogLoggerLevel(slog.LevelInfo)
		slog.SetLogLoggerLevel(slog.LevelWarn)
		slog.SetLogLoggerLevel(slog.LevelError)
	SE SETEA EL NIVEL MINIMO DE LOGS A IMPRIMIR POR CONSOLA*/

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
		slog.Debug("Algoritmo de planificacion no valido")
	}
}

func procesoInicial(path string, size int) {
	//CREAMOS PCB
	pcb := createPCB()
	// Verificar si se puede enviar a memoria, si hay espacio para el proceso

	if consultaEspacioAMemoria(size, pcb) {
		tcb := createTCB(pcb.Pid, 0)       // creamos hilo main
		pcb.Tid = append(pcb.Tid, tcb.Tid) // agregamos el hilo a la listas de hilos del proceso
		enviarTCBMemoria(tcb, path)
		PlanificacionProcesoInicial(pcb, tcb)
	} else {
		slog.Error("Error creando el proceso inicial")
		return
	}
}

func consultaEspacioAMemoria(size int, pcb PCB) bool {
	var memoryRequest KernelRequest
	memoryRequest.Size = size
	memoryRequest.PCB = pcb

	puerto := globals.ClientConfig.PuertoMemoria
	ip := globals.ClientConfig.IpMemoria

	body, err := json.Marshal(memoryRequest)

	if err != nil {
		slog.Error("Fallo el proceso: error codificando " + err.Error())
		return false
	}

	url := fmt.Sprintf("http://%s:%d/hayEspacioEnLaMemoria", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))

	if err != nil {
		slog.Error("error enviando tamanio del proceso", slog.Int("pid", pcb.Pid), slog.String("ip", ip), slog.Int("puerto", puerto))
		return false
	}
	if resp.StatusCode != http.StatusOK {
		slog.Warn("El tamaño del proceso es más grande que la memoria disponible", slog.Int("pid", pcb.Pid)) //Se hace aca, porque si lo ponemos como un else de la consulta a memoria, ante cualquier error responde esto
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

	slog.Info("Se crea el proceso - Estado: NEW", slog.Int("pid", pcb.Pid))
	slog.Info("Se crea el hilo - Estado: READY", slog.Int("pid", tcb.Pid), slog.Int("tid", tcb.Tid))
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

		slog.Info("Se ejecuta el hilo - Estado: EXEC", slog.Int("pid", Hilo.Pid), slog.Int("tid", Hilo.Tid))
		enviarTCBCpu(Hilo) // envio el hilo a la cpu para que ejecute sus instruciones
	}
}

func enviarTCBCpu(tcb TCB) error {

	cpuRequest := TCBRequestCpu{}
	cpuRequest.Pid = tcb.Pid
	cpuRequest.Tid = tcb.Tid
	

	puerto := globals.ClientConfig.PuertoCpu
	ip := globals.ClientConfig.IpCpu

	body, err := json.Marshal(&cpuRequest)

	if err != nil {
		slog.Error("error codificando " + err.Error())
		return err
	}

	url := fmt.Sprintf("http://%s:%d/recibirTcb", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))

	if err != nil {
		slog.Error("Error enviando TCB", slog.String("ip", ip), slog.Int("puerto", puerto), slog.Any("error", err))
		return err
	}
	if resp.StatusCode != http.StatusOK {
		slog.Error("error en la respuesta del módulo de cpu:" + fmt.Sprintf("%v", resp.StatusCode))
		return err
	}
	return nil
}

func enviarTCBMemoria(tcb TCB, path string) error {

	memoryRequest := TCBRequestMemory{}
	memoryRequest.Pid = tcb.Pid
	memoryRequest.Tid = tcb.Tid
	memoryRequest.Path = path

	puerto := globals.ClientConfig.PuertoMemoria
	ip := globals.ClientConfig.IpMemoria

	body, err := json.Marshal(&memoryRequest)

	if err != nil {
		slog.Error("error codificando " + err.Error())
		return err
	}

	url := fmt.Sprintf("http://%s:%d/recibirTcb", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))

	if err != nil {
		slog.Error("Error enviando TCB", slog.String("ip", ip), slog.Int("puerto", puerto), slog.Any("error", err)) // err contiene el error que causo que no se envie la tcb
		return err
	}
	if resp.StatusCode != http.StatusOK {
		slog.Error("Error en la respuesta del módulo de CPU", slog.Int("status_code", resp.StatusCode))
		return err
	}
	return nil
}

func CrearProceso(w http.ResponseWriter, r *http.Request) {
	var proceso IniciarProcesoResponse
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
	if consultaEspacioAMemoria(size, pcb) {
		nextTid = 0
		tcb := createTCB(pcb.Pid, prioridad) // creamos hilo main
		pcb.Tid = append(pcb.Tid, tcb.Tid)   // agregamos el hilo a la listas de hilos del proceso
		enviarTCBMemoria(tcb, path)

		mutexColaNewproceso.Lock()
		colaNewproceso = append(colaNewproceso, pcb) // agregamos el proceso a la cola de new
		mutexColaNewproceso.Unlock()

		mutexColaReadyHilo.Lock()
		colaReadyHilo = append(colaReadyHilo, tcb) // agregamos el hilo a la cola de ready
		mutexColaReadyHilo.Unlock()

		slog.Info("Se crea el proceso - Estado: NEW", slog.Int("pid", pcb.Pid))

		slog.Info("Se crea el hilo - Estado: READY", slog.Int("pid", tcb.Pid), slog.Int("tid", tcb.Tid))

	} else {
		slog.Warn("El tamaño del proceso es mas grande que la memoria, esperando a que finalice otro proceso ....")
		// esperar a que finalize otro proceso y volver a consultar por el espacio en memoria para inicializarlo
		<-finProceso
		iniciarProceso(path, size, prioridad)
	}

	return nil
}

func FinalizarProceso(w http.ResponseWriter, r *http.Request) {
	var hilo FinalizarProcesoResponse
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
		slog.Warn("El hilo no es el principal, no se puede ejecutar esta instruccion")
		return //Ver como hacer para que no finalice el kernel y el hilo continue con su siguiente instruccion
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
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
			slog.Info("Finaliza el proceso - Estado: EXIT", slog.Int("pid", pcb.Pid))

			// Eliminar todos los hilos del proceso
			for i := len(colaReadyHilo) - 1; i >= 0; i-- { // recorremos la lista de hilos ready
				if colaReadyHilo[i].Pid == pid { // si el pid del hilo coincide con el pid del proceso
					mutexColaReadyHilo.Lock()
					colaReadyHilo = append(colaReadyHilo[:i], colaReadyHilo[i+1:]...) // sacamos el hilo de la cola de ready
					mutexColaReadyHilo.Unlock()

					mutexColaExitHilo.Lock()
					colaExitHilo = append(colaExitHilo, colaReadyHilo[i]) // agregamos el hilo a la cola de exit
					mutexColaExitHilo.Unlock()

					slog.Info(" ## finaliza el hilo - Estado: EXIT ##", slog.Int("pid", colaReadyHilo[i].Pid), slog.Int("tid", colaReadyHilo[i].Tid))
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
					slog.Info(" ## finaliza el hilo - Estado: EXIT ##", slog.Int("pid", colaReadyHilo[i].Pid), slog.Int("tid", colaReadyHilo[i].Tid))
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

					slog.Info(" ## finaliza el hilo - Estado: EXIT ##", slog.Int("pid", colaReadyHilo[i].Pid), slog.Int("tid", colaReadyHilo[i].Tid))
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

	memoryRequest := PCBRequest{}
	memoryRequest.Pid = pcb.Pid
	memoryRequest.Tid = pcb.Tid
	memoryRequest.Mutex = pcb.Mutex

	puerto := globals.ClientConfig.PuertoMemoria
	ip := globals.ClientConfig.IpMemoria

	body, err := json.Marshal(&memoryRequest)

	if err != nil {
		slog.Error("error codificando" + err.Error())
		return err
	}

	url := fmt.Sprintf("http://%s:%d/finalizacionProceso", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))

	if err != nil {
		slog.Error("Error enviando TCB", slog.String("ip", ip), slog.Int("puerto", puerto))
		return err
	}
	if resp.StatusCode != http.StatusOK {
		slog.Error("Error en la respuesta del módulo de CPU", slog.Int("status_code", resp.StatusCode))
		return err
	}
	return nil

}

func CrearHilo(w http.ResponseWriter, r *http.Request) {
	 var hilo CrearHiloResponse
	 decoder := json.NewDecoder(r.Body)
	 err := decoder.Decode(&hilo)
	 if err != nil {
		 http.Error(w, err.Error(), http.StatusBadRequest)
		 return
	 }

	 pid := hilo.Pid
	 path := hilo.Path
	 prioridad := hilo.Prioridad

	 err = iniciarHilo(pid, path, prioridad)

	 if err != nil {
		 http.Error(w, err.Error(), http.StatusInternalServerError)
		 return
	 }

	 w.WriteHeader(http.StatusOK)	 
	
}

func iniciarHilo(pid int, path string, prioridad int) error {
	
	tcb := createTCB(pid, prioridad) // creo hilo
	enviarTCBMemoria(tcb, path)     // envio hilo a memoria con el path que le corresponde ejecutar
	pcb := getPCB(pid)            // obtengo el proceso al que pertenece el hilo
	pcb.Tid = append(pcb.Tid, tcb.Tid) // agrego el hilo a la lista de hilos del proceso

	mutexColaReadyHilo.Lock()
	colaReadyHilo = append(colaReadyHilo, tcb) // agrego hilo a la cola de ready
	mutexColaReadyHilo.Unlock()

	slog.Info("Se crea el hilo - Estado: READY", slog.Int("pid", tcb.Pid), slog.Int("tid", tcb.Tid))

	return nil

}

func getPCB(pid int) PCB {
	for _, pcb := range colaNewproceso {
		if pcb.Pid == pid {
			return pcb
		}
	}
	return PCB{}
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
		slog.Error("error codificando mensaje: " + err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/mensaje", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		slog.Error("Error enviando mensaje", slog.String("ip", ip), slog.Int("puerto", puerto))
	}

	slog.Info("respuesta del servidor: " + resp.Status)
}

func RecibirMensaje(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje Mensaje
	err := decoder.Decode(&mensaje)
	if err != nil {
		slog.Error("Error al decodificar mensaje: " + err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	slog.Info("Conexion con Kernel" + fmt.Sprintf("%+v\n", mensaje))

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
