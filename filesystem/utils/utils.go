package utils

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"time"

	"net/http"

	"github.com/sisoputnfrba/tp-golang/filesystem/globals"
)

/*---------------------- ESTRUCTURAS ----------------------*/
type FSmemoriaREQ struct {
	Data          []byte `json:"data"`
	Tamanio       uint32 `json:"tamanio"`
	NombreArchivo string `json:"nombreArchivo"`
}

type Bitmap struct {
	bits            []int
	contadorBloques int
	tamanioBloques  int
}

/*-------------------- VAR GLOBALES --------------------*/
var ConfigFS *globals.Config

/*---------------------- FUNCIONES CONFIGURACION Y LOGGERS ----------------------*/

func IniciarConfiguracion(filePath string) *globals.Config {
	var config *globals.Config
	configFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)

	return config
}

func ConfigurarLogger() {
	logFile, err := os.OpenFile("tp.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
}

/*---------------------- FUNCION INIT ----------------------*/

func init() {
	ConfigFs := IniciarConfiguracion("configsFS/config.json")

	//Al iniciar el modulo se debera validar que existan los archivos bitmap.dat y bloques.dat. En caso que no existan se deberan crear. Caso contrario se deberan tomar los ya existentes.
	if ConfigFs != nil {
		pathFS := ConfigFS.Mount_dir
		iniciarArchivo(pathFS, "bitmap.dat")
		iniciarArchivo(pathFS, "bloques.dat")
	}

}

/*---------------------- FUNCIONES DE ARCHIVOS ----------------------*/

func iniciarArchivo(path string, nombreArchivo string) {
	_, err := os.Stat(nombreArchivo)

	if os.IsNotExist(err) {
		crearArchivo(path, nombreArchivo)
	} else {
		log.Printf("El archivo '%s' ya existe, no hace falta crearlo", nombreArchivo)
	}

	time.Sleep(time.Duration(ConfigFS.Block_access_delay) * time.Millisecond)
}

func crearArchivo(path string, nombreArchivo string) {
	nombre := path + "/" + nombreArchivo
	archivo, err := os.Create(nombre)
	if err != nil {
		log.Fatalf("Error al crear el archivo '%s': %v", path, err)
	}
	defer archivo.Close()
	log.Println("Archivo creado:", nombreArchivo)
}

//Bloques dat almacena los datos de bloques, y bitmap almacena los bloques que estan ocupados

//Lo que reciben es el contenido de la memoria del usuario, cosa que en su caso como estan en Go,
// si, es un array de bytes y lo único que tienen que hacer es copiar el contenido en los bloques

func DumpMemory(w http.ResponseWriter, r *http.Request) {
	dumpReq := FSmemoriaREQ{}
	// Decodificar la solicitud JSON
	if err := json.NewDecoder(r.Body).Decode(&dumpReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	time.Sleep(time.Duration(ConfigFS.Block_access_delay) * time.Millisecond)
}
