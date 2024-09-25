package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/sisoputnfrba/tp-golang/memoria/globals"
	"github.com/sisoputnfrba/tp-golang/memoria/utils"
)

var MemoriaUsuario []byte

func main() {
	utils.ConfigurarLogger()

	globals.ClientConfig = utils.IniciarConfiguracion("configsMemoria/config.json")

	if globals.ClientConfig == nil {
		log.Fatalf("No se pudo cargar la configuración")
	}

	puerto := globals.ClientConfig.Puerto
	tamMemoria := globals.ClientConfig.Tamanio_Memoria 
    MemoriaUsuario = make([]byte, tamMemoria) //inicializar tamaño de la memoria

	// funciones que va a manejar el servidor (Kernel , cpu y filesystem)
	//mux.HandleFunc("Endpoint", Funcion a la que responde)
	mux := http.NewServeMux() // se crea el servidor
	mux.HandleFunc(" /mensaje", utils.RecibirMensaje)
	// mux.HandleFunc("POST /actualizarContextoDeEjecucion", utils.ActualizarContextoDeEjecucion)
	// http.HandleFunc("POST /setInstructionFromFileToMap", utils.SetInstructionsFromFileToMap) //guardo todo en un map
	http.HandleFunc("POST /createProcess", utils.CreateProcess) //creo un proceso cuando me pasan el pcb,tcb,path y size
	http.HandleFunc("POST /terminateProcess", utils.TerminateProcess) //borro un proceso cuando me pasan el pid
	http.HandleFunc("POST /createThread", utils.CreateThread) //creo un hilo cuando me pasan el pcb,tcb,path y size
	http.HandleFunc("POST /terminateThread", utils.TerminateThread) //borro un hilo cuando me pasan el pid-tid
	http.HandleFunc("GET /obtenerInstruccion", utils.GetInstruction) //me piden instrucciones y las paso 
	http.HandleFunc("GET /obtenerContextoDeEjecucion", utils.GetExecutionContext) //me piden el contexto de ejecucion y lo paso
    http.HandleFunc("POST /actualizarContextoDeEjecucion", utils.UpdateExecutionContext) //me mandan el contexto de ejecucion y lo actualizo

	http.ListenAndServe(":"+strconv.Itoa(puerto), mux)


}
