# TLS Certificate Generation and Installation Howto

This HOWTO will show how to generate TLS certificates that can be used with nginx ingresses in kubernetes.  The certificates will be created for the fictitious FQDN `pocstore.local`.

The steps outlined below apply to local generation on macOS, but can probably be adapted for other systems. 

## Prerequisites ##
You must have `openssl` installed on your system

## Configuration File Used for pocstore.local Certificate Generation ##

A configuration file, `pocstore.conf`, will be used when running openssl to generate the TLS certificate for the fictitious FQDN `pocstore.local`.  

If you are using this project structure, create a directory named `configuration` under the `resources/security/tls_certificates` directory and save the file there with the name `pocstore.conf`. *The directory and 2 conf files may already exist.  You can edit them as needed for your situation*  

The contents of `pocstore.conf` are as follows:

```[ req ]
default_bits       = 4096
distinguished_name = req_distinguished_name
req_extensions     = req_ext

[ req_distinguished_name ]
countryName                 = Country Name (2 letter code)
countryName_default         = US
stateOrProvinceName         = State or Province Name (full name)
stateOrProvinceName_default = Your state
localityName                = Locality Name (eg, city)
localityName_default        = Your city
organizationName            = Organization Name (eg, company)
organizationName_default    = Your company
commonName                  = Common Name (e.g. server FQDN or YOUR name)
commonName_max              = 64
commonName_default          = pocstore.local

[ req_ext ]
subjectAltName = @alt_names

[alt_names]
DNS.1   = pocstore.local
DNS.2   = pocstore
```

## Generate TLS Certificate Using openssl ##

The following commands generate a TLS certificate using the configuration file created above.  The certificate will be created in the folder in which you run the openssl commands (if you use the options outlined below). 

If you are using this project structure, navigate to the `resources/security/tls_certificates/certificates` directory and run the commands from there.

```
openssl genrsa -out pocstore.key 4096  

openssl req -new -sha256 -out pocstore.csr -key pocstore.key -config ../configuration/pocstore.conf 

openssl req -text -noout -in pocstore.csr 

openssl x509 -req -days 3650 -in pocstore.csr -signkey pocstore.key -out pocstore.crt -extensions req_ext -extfile ../configuration/pocstore.conf
```

## Add the Certificate to macOS Keychain ##

This step allows you to test locally with Safari and not receive security warnings.

```sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain pocstore.crt```

## Script ##

All of these steps are included in the script named `tls_certificates_generate.sh` in the `scripts/tls_certificates` directory.  That script creates certificates for the FQDN poctore.local (described here) and keycloak.local.

I suppose that I could have mentioned the script in the beginning of this HOWTO, but then you would have never have read it!

## Deploy Certificates as Kubernetes Secrets ##

The script named `tls_certificates_install.sh` in the `scripts/kubernetes/install` directory shows the steps needed to deploy the certificates as secrets in a kubernetes cluster.

## Create and Deploy an Nginx Ingress Configured for TLs ##

Please see any of the *-ingress.yaml files in the `deployment/k8s/<service>/` directories for examples of how to create an ingress that uses TLS using the kubernetes secrets created for the certificates.

## Credits ##

The following site helped immensely with understanding the steps needed to generate and install certificates into the keychain oin macOS:  

https://gist.github.com/leevigraham/e70bc5c0a585f40536252abab61875d8