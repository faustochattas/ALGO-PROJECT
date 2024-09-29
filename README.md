# A.L.G.O-TP-GO

Este repositorio contiene el código fuente y los archivos relacionados con el trabajo práctico de Sistemas Operativos, implementado en el lenguaje de programación Go. El proyecto está diseñado para simular y demostrar conceptos fundamentales de sistemas operativos.

[Enunciado A.L.G.O](https://docs.google.com/document/d/1w0PK_ZCUHsvGrVER_rKW7HkqLjHdkC6H0IBsmHUqhS4/edit)

[Documento de Pruebas Preliminares](https://faq.utnso.com.ar/tp-c-comenta-pruebas)

> Los resultados seran los mismos que en la implementacion de C


Instrucciones de instalacion: 

```terminal
> git clone https://github.com/adriangilt/A.L.G.O-TP-GO.git
> cd A.L.G.O-TP-GO/scripts
> ./build_modulos.sh
```

Configurar IP de cada Modulo. Por ejemplo: 

```terminal
> cd kernel
> vim config.json
```

Posteriormente, levantar modulos en el siguiente orden: 

1. Memoria
2. CPU
3. Kernel
4. Entrada/salida

Por ultimo ejecutar pruebas. Por ejemplo:
```terminal
> cd algo-pruebas
> cd scripts_kernel
> ./PRUEBA_PLANI.sh
```

Espero que les sirva como guia y cualquier tipo de contribucion es bienvenida.

#### Aclaración: Esta es solo una de las diferentes formas en las que se puede desarrollar este TP.

 

