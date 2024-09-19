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

	mux := http.NewServeMux()

	mux.HandleFunc("/crearProceso", utils.CrearProceso)
	mux.HandleFunc("/finalizarProceso", utils.FinalizarProceso)

	mux.HandleFunc("/crearHilo", utils.CrearHilo)
	mux.HandleFunc("/finalizarHilo", utils.FinalizarHilo)
	mux.HandleFunc("/cancelarHilo",  utils.CancelarHilo)
	mux.HandleFunc("/unirseAHilo", utils.EntrarHilo)
	

	mux.HandleFunc("/crearMutex", utils.CrearMutex)
	mux.HandleFunc("/bloquearMutex", utils.BloquearMutex)
	mux.HandleFunc("/liberarMutex", utils.LiberarMutex)

	//Escuchar (bloqueante)
	http.ListenAndServe(":"+strconv.Itoa(puerto), mux)

}
