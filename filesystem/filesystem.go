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
	//IpMemoria := globals.ClientConfig.IpMemoria
	//PuertoMemoria := globals.ClientConfig.PuertoMemoria
	puerto := globals.ClientConfig.Puerto

	//utils.EnviarMensaje(IpMemoria, PuertoMemoria, "Hola Memoria, Soy FS")

	mux := http.NewServeMux() // se crea el servidor

	// funciones que va a manejar el servidor (SOLO CON MEMORIA)
	//mux.HandleFunc("Endpoint", Funcion a la que responde)

	mux.HandleFunc("/mensaje", utils.RecibirMensaje)

	http.ListenAndServe(":"+strconv.Itoa(puerto), mux)
	/* if err != nil {
		panic(err)
	} */

}
