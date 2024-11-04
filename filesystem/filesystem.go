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

	http.HandleFunc("POST /dumpMemory", utils.DumpMemory)

	http.ListenAndServe(":"+strconv.Itoa(puerto), nil)

}
