package main

import (
	"flag"
	"fmt"
	"strings"
	"github.com/fsouza/go-dockerclient"
	"os"
	"os/exec"
	"reflect"
	"text/template"
	"time"
)

// Modo de empleo: ./proxyRebuild -servicetpl service.tpl -destination destination.conf
// Para compilar desde OSX para arquitectura Linux AMD64: env GOOS=linux GOARCH=amd64 go build -v

func main() {
	endpoint := "unix:///var/run/docker.sock"
	client, err := docker.NewClient(endpoint)
	if err != nil {
		panic(err)
	}

	templateFile := flag.String("servicetpl", "/etc/service.tpl", "Fichero de plantilla del que hacer rebuild")
	endCommand := flag.String("cmd", "/etc/init.d/nginx reload", "Comando que se ejecutara al finalizar el rebuild de los ficheros de config")
	outputFile := flag.String("destination", "/etc/nginx/conf.d/templateresult.conf", "Fichero resultante despues de aplicar rebuild de config")
	serviceNetwork := flag.String("network", "any", "Red en la que han de estar los servicios que se añadiran al proxy")

	flag.Parse()

	const delay = 30000 * time.Millisecond

	var serviceTemplate = template.Must(template.ParseFiles(*templateFile))

	type ServiceEntry struct {
		ServiceName         string
		ServiceDomain       string
		ServicePath         string
		ServiceInternalPort string
		ServiceExternalPort string
	}

	// https://docs.docker.com/engine/api/v1.24/#services

	arrayDataOld := []ServiceEntry{}
	arrayData := []ServiceEntry{}

	for {

		services, err := client.ListServices(docker.ListServicesOptions{})
		if err != nil {
			panic(err)
		}
		arrayDataOld = make([]ServiceEntry, len(arrayData))
		copy(arrayDataOld, arrayData)
		arrayData = arrayData[:0]

		var isOnSameNetwork bool
		//var servicePath string
		//var extPort string


		for _, srv := range services {
			//fmt.Println("ID: ", srv.ID)
			//fmt.Println("CreatedAt: ", srv.CreatedAt)
			fmt.Println(" - Procesando servicio: Spec.Name=", srv.Spec.Name)
			//fmt.Println("Spec.Labels: %v ", srv.Spec.Labels)

			if val1, ok := srv.Spec.Labels["com.autoproxy.proxy"]; !ok {
				
				if val2, ok := srv.Spec.TaskTemplate.ContainerSpec.Labels["com.autoproxy.proxy"]; !ok {
					fmt.Println(" - NO El servicio  ", srv.Spec.Name, " no está en proxy ", string(val1) + string(val2) )
					continue
				}
			} //else{
			//    fmt.Println(" - SI El servicio " , srv.Spec.Name , " si está en proxy: " , val)
			//}

			isOnSameNetwork = false

			for _, network := range srv.Spec.Networks {
				if network.Target == string(*serviceNetwork) {
					//fmt.Println("SI El servicio  " , srv.Spec.Name , "  está en la red ", string(*serviceNetwork))
					isOnSameNetwork = true
				}else{
					fmt.Println(" - Red no igual ", "==", network.Target )
				}
			}
			if isOnSameNetwork == false && string(*serviceNetwork) != "any" {
				fmt.Println(" - El servicio  ", srv.Spec.Name, " NO esta en la misma red ", string(*serviceNetwork) )
				continue
			}

			
			serviceDomain := ""
			if val, ok := srv.Spec.Labels["com.autoproxy.domain"]; !ok {
				if val2, ok := srv.Spec.TaskTemplate.ContainerSpec.Labels["com.autoproxy.domain"]; !ok {
					fmt.Println(" - El servicio  ", srv.Spec.Name, " no tiene un dominio configurado", val, val2)
					continue
				}else{
					 serviceDomain = srv.Spec.TaskTemplate.ContainerSpec.Labels["com.autoproxy.domain"]
				}
			}else{
				serviceDomain = srv.Spec.Labels["com.autoproxy.domain"]
			}

			servicePath := "/"
			if val, ok := srv.Spec.Labels["com.autoproxy.path"]; ok {
				servicePath = val
			} else {
                if val2, ok := srv.Spec.TaskTemplate.ContainerSpec.Labels["com.autoproxy.path"]; ok {
                    servicePath = val2
                }else{
					servicePath = "/"
				}
			}

			extPort := "80"
			if val, ok := srv.Spec.Labels["com.autoproxy.extPort"]; ok {
				extPort = val
			} else {
				if val2, ok := srv.Spec.TaskTemplate.ContainerSpec.Labels["com.autoproxy.extPort"]; !ok {
					extPort = "80"
				}else{
					extPort = val2
				}
			}

			intPort := "80"
			if val, ok := srv.Spec.Labels["com.autoproxy.intPort"]; !ok {
				if val2, ok := srv.Spec.TaskTemplate.ContainerSpec.Labels["com.autoproxy.intPort"]; !ok {
					fmt.Println(" - No se ha especificado el puerto interno del servicio ", srv.Spec.Name, " ", val)
					continue
				}else{
					intPort =  val2
				}
			}else{
				intPort = srv.Spec.Labels["com.autoproxy.intPort"]
			}

			serviceName := srv.Spec.Name
			if val, ok := srv.Spec.Labels["com.docker.stack.namespace"]; ok {
				serviceName = strings.Replace(srv.Spec.Name, srv.Spec.Labels["com.docker.stack.namespace"] + "_", "", 1 )
				fmt.Println(" - Dentro de un Stack, usando prefijo: ", val)
			}

			// Vamos creando estructura
			entry := &ServiceEntry{
				ServiceName:         serviceName,
				ServiceDomain:       serviceDomain,
				ServicePath:         servicePath,
				ServiceInternalPort: intPort,
				ServiceExternalPort: extPort,
			}

			fmt.Println(" - Adding service " + srv.Spec.Name + " to array ")
			arrayData = append(arrayData, *entry)

			//if srv.Spec.Labels["domain"] != nil {
			//fmt.Println("Label Domain: " , srv.Spec.Labels["domain"])
			//}
			//if srv.Spec.Labels["proxy"] != nil {
			//fmt.Println("Label Proxy: " , srv.Spec.Labels["proxy"])
			//}

			//for _, lbl := range srv.Spec.Labels  {
			//fmt.Printf("%v", lbl)
			//}
			// fmt.Println("EndpointSpec.Mode: ", srv.EndpointSpec.Mode)
			//fmt.Printf("%v", srv)

		}

		// fmt.Println("DeepEqual::: ", reflect.DeepEqual(arrayData, arrayDataOld), " Len:", len(arrayData), " LenOld:", len(arrayDataOld))

		sonIguales := reflect.DeepEqual(arrayData, arrayDataOld)

		if !sonIguales {
			if _, err := os.Stat(string(*outputFile)); err == nil {
				// path/to/whatever exists
				err = os.Remove(string(*outputFile))
				if err != nil {
					fmt.Println(err)
					return
				}
			}

			f, err := os.Create(string(*outputFile))
			if err != nil {
				fmt.Println("Create file Error: ", err)
				return
			}
			if err := serviceTemplate.Execute(f, arrayData); err != nil {
				fmt.Println(err)
			}

			outputCmd, err := exec.Command("/bin/sh", "-c", string(*endCommand)).Output()
			if err != nil {
				fmt.Println("Failed to execute command: %s", outputCmd)
				fmt.Println(err)
			} else {
				fmt.Println("RESULTADO Comando::::: ", string(outputCmd))
			}
		}else{
                    fmt.Println(" - * No se han detectado cambios en los servicios de Swarm")
                }

		time.Sleep(delay)
	} // End infinite loop
}
