package globals

type Config struct {
	IpMemoria              string `json:"ip_memoria"`              //IP a la cual se deberá conectar con la Memoria
	PuertoMemoria          int    `json:"puerto_memoria"`          //Puerto al cual se deberá conectar con la Memoria
	IpCpu                  string `json:"ip_cpu"`                  //IP a la cual se deberá conectar con el Kernel
	PuertoCpu              int    `json:"puerto_cpu"`              //Puerto al cual se deberá conectar con el Kernel
	AlgoritmoPlanificacion string `json:"algoritmo_planificacion"` //Algoritmo de planificación a utilizar
	Quantum                int    `json:"quantum"`                 //Quantum de tiempo a utilizar en el algoritmo de planificación
	LogLevel               string `json:"log_level"`               //Nivel de detalle máximo a mostrar.
}

var ClientConfig *Config
