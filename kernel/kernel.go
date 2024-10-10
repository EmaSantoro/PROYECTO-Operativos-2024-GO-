package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/sisoputnfrba/tp-golang/kernel/globals"
	"github.com/sisoputnfrba/tp-golang/kernel/utils"
)

func main() {
	utils.ConfigurarLogger()
	globals.ClientConfig = utils.IniciarConfiguracion("configsKERNEL/config.json")

	if globals.ClientConfig == nil {
		log.Fatalf("No se pudo cargar la configuración")
	}

	puerto := globals.ClientConfig.Puerto

	//mux := http.NewServeMux()

	http.HandleFunc("POST /crearProceso", utils.CrearProceso)
	http.HandleFunc("DELETE /finalizarProceso", utils.FinalizarProceso)

	http.HandleFunc("POST /crearHilo", utils.CrearHilo)
	http.HandleFunc("DELETE /finalizarHilo", utils.FinalizarHilo)
	http.HandleFunc("DELETE /cancelarHilo", utils.CancelarHilo)
	http.HandleFunc("POST /unirseAHilo", utils.EntrarHilo)

	http.HandleFunc("POST /crearMutex", utils.CrearMutex)
	http.HandleFunc("POST /bloquearMutex", utils.BloquearMutex)
	http.HandleFunc("POST /liberarMutex", utils.LiberarMutex)

	http.HandleFunc("POST /manejarIo", utils.ManejarIo)

	http.HandleFunc("POST /dumpMemory", utils.DumpMemory)

	//Escuchar (bloqueante)
	http.ListenAndServe(":"+strconv.Itoa(puerto), nil)

}
