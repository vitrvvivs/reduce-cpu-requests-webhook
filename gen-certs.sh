#!/bin/bash

mkdir -p certs && cd certs || exit 1

openssl genrsa -out ca.key 2048

openssl req -new -x509 -days 365 -key ca.key \
  -subj "/C=AU/CN=reduce-cpu-requests-webhook"\
  -out ca.crt

openssl req -newkey rsa:2048 -nodes -keyout server.key \
  -subj "/C=AU/CN=reduce-cpu-requests-webhook" \
  -out server.csr

openssl x509 -req \
  -extfile <(printf "subjectAltName=DNS:reduce-cpu-requests-webhook.kube-system.svc") \
  -days 365 \
  -in server.csr \
  -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out server.crt

# YAML parsing in sed; How horrifying!
sed -E -f - ../k8s-resources.yaml.template >../k8s-resources.yaml <<EOF
    /^kind: Secret$/,/---/{
        1,/^data:/!{ # skip until data:
            1,/^(  ){0,1}[^ ]/{ 
                s;tls.crt: .*$;tls.crt: $(base64 -w0 <server.crt);; 
                s;tls.key: .*$;tls.key: $(base64 -w0 <server.key);; 
            }
        }
    };
    /^kind: MutatingWebhookConfiguration$/,/---/{
        1,/^webhooks:/!{
        /^    clientConfig:/,/^(  ){0,2}[^ ]/{
                s;caBundle: .*$;caBundle: $(base64 -w0 <ca.crt);; 
        }}
    }
EOF

echo "Generated certificates and updated k8s-resources.yaml"
