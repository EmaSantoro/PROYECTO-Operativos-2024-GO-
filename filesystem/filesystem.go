package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/sisoputnfrba/tp-golang/filesystem/globals"
	"github.com/sisoputnfrba/tp-golang/filesystem/utils"
)

func main() {
	utils.ConfigurarLogger()

	globals.ClientConfig = utils.IniciarConfiguracion("configsFS/config.json")

	if globals.ClientConfig == nil {
		log.Fatalf("No se pudo cargar la configuración")
	}
	puerto := globals.ClientConfig.Puerto


	// funciones que va a manejar el servidor (SOLO CON MEMORIA)
	//mux.HandleFunc("Endpoint", Funcion a la que responde)
	mux := http.NewServeMux() // se crea el servidor
	mux.HandleFunc("/mensaje", utils.RecibirMensaje)
	http.ListenAndServe(":"+strconv.Itoa(puerto), mux)

}
