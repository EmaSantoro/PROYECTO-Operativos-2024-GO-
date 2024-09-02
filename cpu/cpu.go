package main

import (
	"log"

	"github.com/sisoputnfrba/tp-golang/cpu/globals"
	"github.com/sisoputnfrba/tp-golang/cpu/utils"
)

func main() {
	utils.ConfigurarLogger()
	log.Println("Hola soy un log")

	utils.InciarServidor(globals.ClientConfig.Puerto)
}
