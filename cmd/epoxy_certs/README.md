# ePoxy Certificate Management

The ePoxy server uses two certificate conventions:

* Let's Encrypt certificates for Linux USB images (port 443) stage1+
* Private CA certificates for iPXE firmware images (port 4430 (non-standard)) stage1-only.

The Let's Encrypt certificates are updated automatically. The private CA
certificate's expiration should outlast our hardware. However, because the iPXE
firmware embeds the CA certificate, the server certificates need to be rotated
periodically (every few years).

## Check current certificates

```sh
$ echo \
  | openssl s_client \
      -servername epoxy-boot-api.mlab-sandbox.measurementlab.net \
      -connect epoxy-boot-api.mlab-sandbox.measurementlab.net:4430 2>/dev/null \
  | openssl x509 -text
```

## Generate Certificates

Create new server certificates with 5 year expiration plus 30 extra days.

```sh
$ epoxy_certs server -hostname epoxy-boot-api.mlab-sandbox.measurementlab.net -duration $(( 5*8761 + 24*30 ))h
$ openssl x509 -noout -text -in ./server-cert.pem
Certificate:
    Data:
        ...
        Validity
            Not Before: Jan  2 22:09:15 2024 GMT
            Not After : Feb  5 03:09:15 2029 GMT
        ...
```
