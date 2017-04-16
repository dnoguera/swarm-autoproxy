# swarm-autoproxy
Generic autorebuild tool to create ingress proxy to swarm cluster mainly for nginx

## Modo de empleo:

Creamos un servicio con los labels necesarios para que se queden registrados correctamente:
```
docker service create --replicas 1 --name helloworld \
  --network proxy-http \
  --label proxy=True \
  --label domain=menganito.com \
  --label intPort=80 \
  alpine ping docker.com
```

Ejecutamos el binario compilado que cada 30 segundos consulta el API de Docker y hace rebuild del servicio indicado:
```
./proxyRebuild -servicetpl service.tpl -destination "/etc/nginx/conf.d/destination.conf" -cmd "service nginx reload"
```

## Help:
```
Usage of ./proxyRebuild:
  -cmd string
    	Comando que se ejecutara al finalizar el rebuild de los ficheros de config (default "/etc/init.d/nginx reload")
  -destination string
    	Fichero resultante despues de aplicar rebuild de config (default "/etc/nginx/conf.d/templateresult.conf")
  -network string
    	Red en la que han de estar los servicios que se añadiran al proxy (default "proxy-http")
  -servicetpl string
    	Fichero de plantilla del que hacer rebuild (default "/etc/service.tpl")
```


