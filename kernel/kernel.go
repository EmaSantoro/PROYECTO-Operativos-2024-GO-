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

	/*
		Al iniciar el módulo Kernel, se creará un proceso inicial para que éste lo planifique.
		Para poder inicializarlo se requerirá que el Kernel recibirá dos parámetros:
		el nombre del archivo de pseudocódigo que deberá ejecutar y el tamaño del proceso para ser inicializado en Memoria,
		el TID 0 creado por este proceso tendrá la prioridad máxima 0 (cero).
	*/

	// Funciones
	http.HandleFunc("PUT /proceso", utils.iniciarProceso) // buscar que es mux xd

	//Escuchar (bloqueante)
	http.ListenAndServe(":"+strconv.Itoa(puerto), nil)

}
