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

	puerto := globals.ClientConfig.Puerto //posteriormente se va a asignar el puerto dentro de cada funcion

	//utils.EnviarPaqueteACPU()

	mux := http.NewServeMux()
	mux.HandleFunc("/mensaje", utils.RecibirMensaje) // buscar que es mux xd
	http.ListenAndServe(":"+strconv.Itoa(puerto), mux)

}
