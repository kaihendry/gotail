# gotail

<https://youtu.be/XmCDji3t7eg>

# Nginx configuration

	server {
			server_name example.com;
			location / {
					proxy_pass http://localhost:8080;
					proxy_set_header Connection '';
					proxy_http_version 1.1;
					chunked_transfer_encoding off;
			}
	}
