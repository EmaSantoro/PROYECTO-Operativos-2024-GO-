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

var colaNew []PCB
var colaReady []PCB
var colaExec []PCB
var colaBlock []PCB
var colaExit []PCB


/*type Paquete struct {
	ID      string `json:"ID"` //de momento es un string que indica desde donde sale el mensaje.
	Mensaje string `json:"mensaje"`
	Size    int16  `json:"size"`
	Array   []rune `json:"array"`
}

var paquete Paquete = Paquete{
	ID:      "CPU", //de momento es un string que indica desde donde sale el mensaje.
	Mensaje: "Soy CPU",
	Size:    int16(len([]rune{'H', 'o', 'l', 'a'})),
	Array:   []rune{'H', 'o', 'l', 'a'},
}*/

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

func init() {
	ConfigKernel := IniciarConfiguracion("configsKERNEL/config.json")
	EnviarMensaje(ConfigKernel.IpMemoria, ConfigKernel.PuertoMemoria, "Hola Memoria, Soy Kernel")
	EnviarMensaje(ConfigKernel.IpCpu, ConfigKernel.PuertoCpu, "Hola CPU, Soy Kernel")
}

func ConfigurarLogger() {
	logFile, err := os.OpenFile("tp.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
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

/*func EnviarPaqueteACPU() {
	ip := globals.ClientConfig.IpCpu
	puerto := globals.ClientConfig.PuertoCpu

	body, err := json.Marshal(paquete)
	if err != nil {
		log.Printf("error codificando paquete: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/paquete", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", ip, puerto)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return
	}
	log.Printf("respuesta del servidor: %s", resp.Status)

}*/

func iniciarProceso() {

	var procesoInicial PCB // creo el proceso inicial (esto nose si esta bien porque se supone que hace esto con cada proceso que llega)
	procesoInicial.Pid = 1
	
	if(len(colaNew) == 0){  // si no hay procesos en colaNew se agrega y se intenta inicializar
		colaNew = append(colaNew, procesoInicial)
	//	if(/*hay espacio en memoria*/){
			
			colaNew = colaNew[1:]  // saco el proceso de la colaNew

			procesoInicial.Tid = append(procesoInicial.Tid , 0) // creo el hilo 0 y lo agrego a la lista de hilos del proceso
			
			colaReady = append(colaReady, procesoInicial)
	
		//else {
			//esperara a que haya espacio en memoria para inicializarlo
		//}

	}	
	//else si hay procesos en colaNew se lo encola 
		colaNew = append(colaNew, procesoInicial)
	
		
	

	 
}

func finalizarProceso(){
	// Liberar PCB asociado
	pcb := colaExec[0]
	colaExec = colaExec[1:]

	// Informar a la Memoria la finalización del proceso
	EnviarMensaje(globals.ClientConfig.IpMemoria, globals.ClientConfig.PuertoMemoria, fmt.Sprintf("Finalizar proceso %d", pcb.Pid))

	// Esperar confirmación de la Memoria
	// ...

	// Liberar PCB
	pcb = PCB{}

	// Intentar inicializar un proceso en estado NEW si los hubiere
	iniciarProceso()
}
