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
	Pid       int
	Tid       int
	Prioridad int
}

var procesoInicial PCB

var colaNewproceso []PCB
var colaReadyproceso []PCB
var colaExecproceso []PCB
var colaBlockproceso []PCB
var colaExitproceso []PCB

var colaReadyHilo []TCB
var colaExecHilo []TCB
var colaBlockHilo []TCB
var colaExitHilo []TCB

/*-------------------- VAR GLOBALES --------------------*/

var (
	nextPid = 1
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

//	Iniciar modulo
func init() {
	ConfigKernel := IniciarConfiguracion("configsKERNEL/config.json")

	iniciarProcesoInicial("./procesoBasico", TAMANIO)
	
}

func iniciarProceso(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var path Path
	err := decoder.Decode(&path)

	if err != nil {
		http.Error(w, "Error decoding JSON data", http.StatusInternalServerError)
		return
	}

	//CREAMOS PCB
	pcb := createPCB()

	log.Printf("%+v\n", path)


	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))

	// iniciarProcesoInicial( path , tamanioMemoria ) ver de donde sacar el tamanio del proceso inicial?????

}

func createPCB() PCB {
	nextPid++

	return PCB{
		Pid: nextPid - 1 // ASIGNO EL VALOR ANTERIOR AL pid
		//Tid
		//Mutex
	}
}


func iniciarProcesoInicial(path string, tamanioMemoria int){
	//inicializar PCB
 
	procesoInicial.Pid = 1
	procesoInicial.Tid = append(procesoInicial.Tid, 0)

	colaNewproceso = append(colaNewproceso, procesoInicial)

	tcb := TCB{Pid: procesoInicial.Pid , Tid: 0, Prioridad: 0}

	colaReadyHilo = append(colaReadyHilo, tcb)
	fmt.Println(" ## (<PID>:%d) Se crea el proceso - Estado: NEW ##", procesoInicial.Pid) // se tendria que hacer esto con cada proceso y hilo que llega
	fmt.Println(" ## (<PID>:%d) Se crea el hilo - Estado: READY ##", tcb.Pid)  			// el valor de sus estructura se obtiene de los parametros que llegan de las instrucciones

	// enviar path a cpu


	// enviar tamanio a memoria
	

}
