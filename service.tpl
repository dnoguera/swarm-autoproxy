{{range $index, $element := .}}
#################################################################
########### Vhost configuration of {{$element.ServiceDomain}}{{$element.ServicePath}}
#################################################################
server {

	access_log off;
	error_log  /var/log/nginx/error.log warn;
	listen    {{$element.ServiceExternalPort}};
	server_name  {{$element.ServiceDomain}} ;

	location {{$element.ServicePath}} {
	 	 log_not_found  off;
	 	 client_max_body_size    2000m;
	 	 client_body_buffer_size 512k;
	 	 proxy_send_timeout   600;
	 	 proxy_read_timeout   600;
	 	 proxy_buffer_size    32k;
	 	 proxy_buffers     16 32k;
	 	 proxy_busy_buffers_size 64k;
	 	 proxy_temp_file_write_size 64k;
	 	 proxy_connect_timeout 600s;

        	proxy_pass   http://{{$element.ServiceName}}:{{$element.ServiceInternalPort}};
        	proxy_set_header   Host   $host;
        	proxy_set_header   X-Real-IP  $remote_addr;
        	proxy_set_header   X-Forwarded-For $proxy_add_x_forwarded_for;
          }

}
{{end}}
