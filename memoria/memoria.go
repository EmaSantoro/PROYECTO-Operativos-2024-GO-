package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/sisoputnfrba/tp-golang/memoria/globals"
	"github.com/sisoputnfrba/tp-golang/memoria/utils"
)

func main() {
	utils.ConfigurarLogger()

	globals.ClientConfig = utils.IniciarConfiguracion("configsMemoria/config.json")

	if globals.ClientConfig == nil {
		log.Fatalf("No se pudo cargar la configuración")
	}

	puerto := globals.ClientConfig.Puerto

	// funciones que va a manejar el servidor (Kernel , cpu y filesystem)
	//mux.HandleFunc("Endpoint", Funcion a la que responde)
	mux := http.NewServeMux() // se crea el servidor
	mux.HandleFunc(" /mensaje", utils.RecibirMensaje)
	// mux.HandleFunc("POST /actualizarContextoDeEjecucion", utils.ActualizarContextoDeEjecucion)
	http.HandleFunc("POST /setInstructionFromFileToMap", utils.SetInstructionsFromFileToMap) //guardo todo en un map
	http.HandleFunc("GET /obtenerInstruccion", utils.GetInstruction) //me piden instrucciones y las paso 
	http.HandleFunc("GET /obtenerContextoDeEjecucion", utils.GetExecutionContext) //me piden el contexto de ejecucion y lo paso


	http.ListenAndServe(":"+strconv.Itoa(puerto), mux)


}
