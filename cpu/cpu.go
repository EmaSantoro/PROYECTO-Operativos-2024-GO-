package main

import (
	"log"

	"github.com/sisoputnfrba/tp-golang/cpu/globals"
	"github.com/sisoputnfrba/tp-golang/cpu/utils"
)

func main() {
	utils.ConfigurarLogger()
	log.Println("Hola soy un log")

	globals.ClientConfig = utils.IniciarConfiguracion("cpu/config.json")

	if globals.ClientConfig == nil {
		log.Fatalf("No se pudo cargar la configuración")
	}
}
