## Link del enunciado

https://docs.google.com/document/d/1HSZ14tk7IOfkOf-7ni0Wa6wnKZClEQA7zZyv-h0EZAY/edit?tab=t.0

## Issues

https://github.com/sisoputnfrba/foro/issues?q=is%3Aissue+


# Go Direction (GOD)

## Introducción
**Go Direction (GOD)** es un **proyecto de sistemas operativos** que simula un **sistema distribuido**, donde múltiples procesos interactúan con diferentes módulos del sistema operativo, como la CPU, la memoria y el sistema de archivos.

El objetivo principal es **comprender y aplicar conceptos clave de sistemas operativos y programación de sistemas**, implementando una arquitectura distribuida en **Golang**, con despliegue automatizado en entornos Linux.

## Objetivos

### Conceptuales
- Comprender el funcionamiento de un sistema operativo y su interacción con procesos, memoria, CPU y archivos.
- Aplicar conceptos de planificación de procesos e hilos.
- Manejar memoria y archivos en un entorno simulado.
- Interactuar con syscalls y mecanismos de sincronización como mutex.

### Prácticos
- Desarrollar **módulos desacoplados** con APIs bien definidas.
- **Automatizar el despliegue** en diferentes computadoras.
- Usar herramientas como **Makefiles, archivos de configuración y logs**.
- Implementar una **simulación eficiente y estructurada** de un sistema operativo.

---

## Arquitectura del Sistema
El sistema se divide en cuatro módulos principales:

### 1. Kernel
El **Kernel** administra la ejecución de procesos e hilos, planificándolos y gestionando sus interacciones con los otros módulos.

#### Principales Componentes
- **PCB (Process Control Block):** Almacena información de los procesos en ejecución.
- **TCB (Thread Control Block):** Contiene información de los hilos.

#### Planificación
- **Planificador de Largo Plazo:** Maneja la creación y finalización de procesos.
- **Planificador de Corto Plazo:** Controla la ejecución de hilos usando algoritmos como FIFO, Prioridades y Colas Multinivel.

#### Syscalls Implementadas
- **Procesos:** `PROCESS_CREATE`, `PROCESS_EXIT`
- **Hilos:** `THREAD_CREATE`, `THREAD_JOIN`, `THREAD_CANCEL`, `THREAD_EXIT`
- **Mutex:** `MUTEX_CREATE`, `MUTEX_LOCK`, `MUTEX_UNLOCK`
- **Memoria:** `DUMP_MEMORY`
- **Entrada/Salida:** `IO`

### 2. CPU
Simula el ciclo de instrucción de una CPU real.

#### Ciclo de Instrucción
1. **Fetch:** Obtiene la próxima instrucción.
2. **Decode:** Traduce direcciones lógicas a físicas.
3. **Execute:** Ejecuta la instrucción.
4. **Check Interrupt:** Verifica interrupciones del Kernel.

#### Registros Simulados
Incluye registros como **PC, AX, BX, CX, DX, EX, FX, GX, HX, Base y Límite**.

#### Instrucciones Soportadas
- Operaciones aritméticas: `SUM`, `SUB`
- Manejo de memoria: `READ_MEM`, `WRITE_MEM`, `DUMP_MEMORY`
- Control de flujo: `JNZ`
- Gestión de procesos e hilos: `PROCESS_CREATE`, `THREAD_CREATE`, `THREAD_JOIN`, `THREAD_CANCEL`, `THREAD_EXIT`
- Sincronización: `MUTEX_CREATE`, `MUTEX_LOCK`, `MUTEX_UNLOCK`
- Entrada/Salida: `IO`

### 3. Memoria
Gestiona la asignación y traducción de direcciones de memoria.

#### Esquema de Memoria
- **Memoria del Sistema:** Almacena contextos de ejecución.
- **Memoria de Usuario:** Define particiones **fijas o dinámicas**.

#### Funciones Principales
- **Asignación y liberación de memoria**.
- **Traducción de direcciones** lógicas a físicas.
- **Manejo de errores**, como **Segmentation Fault**.

### 4. File System
Proporciona almacenamiento estructurado para procesos.

#### Esquema de Archivos
- **Estructura jerárquica** de directorios.
- **Identificadores (FID)** para archivos y carpetas.
- **Metadata** con nombre, tamaño, permisos y timestamps.

#### Operaciones Soportadas
- **Creación de archivos**.
- **Lectura y escritura**.
- **Eliminación de archivos**.

#### Manejo de Errores
- **Permisos insuficientes**.
- **Espacio en disco insuficiente**.
- **Archivo no encontrado**.

---

## Deployment y Testing
- **Distribución:** Los procesos deben ejecutarse en distintas computadoras.
- **Automatización:** Desarrollo de scripts para desplegar procesos y archivos de configuración.
- **Documentación:** Ejemplos y archivos de prueba en el repositorio.

## Archivos de Configuración
Cada módulo tendrá un **archivo de configuración** que definirá:
- **IP y puertos** de comunicación.
- **Algoritmos de planificación**.
- **Tamaño de memoria y esquema de partición**.
- **Configuración de logs**.

## Autores

Este proyecto no podria haber sido posible sin el trabajo de cada uno de sus integrantes.

Nicolas Schkurko (@NicolasSchkurko)

Milagros Arce (@milagrosarce)

Milagros Lujan Salafia (@MilagrosLu)

Ramiro Di Santo (@ramidisanto)

Santoro Emanuel (@EmaSantoro)