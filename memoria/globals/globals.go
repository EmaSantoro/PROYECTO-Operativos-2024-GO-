package globals

type Config struct {
	Puerto            int         `json:"port"`               // Puerto de escucha del servidor
	Tamanio_Memoria   int         `json:"tamanio_memoria"`    // Tamaño de la memoria expresado en bytes
	Path_Instruccion  string      `json:"path_instruccion"`   // Carpeta donde se encuentran los archivos de pseudocodigo
	Delay_Respuesta   int         `json:"delay_respuesta"`    // Tiempo de espera para responder una instruccion
	IpKernel          string      `json:"ip_kernel"`          // IP del kernel
	PuertoKernel      int         `json:"port_kernel"`        // Puerto del kernel
	IpCpu             string      `json:"ip_cpu"`             // IP de la CPU
	PuertoCpu         int         `json:"port_cpu"`           // Puerto de la CPU
	IpFs              string      `json:"ip_filesystem"`      // IP del filesystem
	PuertoFs          int         `json:"port_filesystem"`    // Puerto del filesystem
	EsquemaMemoria    string      `json:"esquema_memoria"`    // Esquema de particiones de memoria a utilizar
	AlgoritmoBusqueda string      `json:"algoritmo_busqueda"` // Algoritmo de busqueda de huecos en memoria
	Particiones       []int       `json:"particiones"`        // Lista ordenada con las particiones a generar en el algoritmo Particiones fijas
	Log_Level         string      `json:"log_level"`          // Nivel de loggeo
}

// type Particion struct {
// 	Tamanio int `json:"tamanio"` // Tamaño de la particion
// 	//Estado int `json:"estado"` // Estado de la particion //
// }

var ClientConfig *Config


