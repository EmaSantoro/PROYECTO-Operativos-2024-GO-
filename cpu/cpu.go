package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/sisoputnfrba/tp-golang/cpu/globals"
	"github.com/sisoputnfrba/tp-golang/cpu/utils"
)

func main() {
	utils.ConfigurarLogger()

	globals.ClientConfig = utils.IniciarConfiguracion("configsCPU/config.json")

	if globals.ClientConfig == nil {
		log.Fatalf("No se pudo cargar la configuración")
	}
	puerto := globals.ClientConfig.Puerto

	mux := http.NewServeMux()
	mux.HandleFunc("/mensaje", utils.RecibirMensaje)
	//mux.HandleFunc("/paquete", utils.RecibirPaquete)
	mux.HandleFunc("/recibirTcb", utils.RecibirPIDyTID)
	//mux.HandleFunc("/interrupcion", utils.Interruption)
	mux.HandleFunc("/receiveDataFromMemor", utils.RecieveDataFromMemory)
	mux.HandleFunc("/interrupcion", utils.RecieveInterruption)
	http.ListenAndServe(":"+strconv.Itoa(puerto), mux)

}
