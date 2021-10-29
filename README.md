# csi-driver-cacerts

CSI driver that add ca certificates to a the OS trusted certificate issuers (eg, /etc/ssl/certs/ca-certificates.crt, /etc/ssl/certs/java/cacerts) so that users don't need to pass ca certificates to individual applications or use insecure mode (eg, curl -k).

## Example

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: curl-debian
  namespace: demo
spec:
  containers:
  - name: main
    image: appscode/curl:debian
    command:
    - sleep
    - "3600"
    volumeMounts:
    - name: cacerts
      mountPath: /etc/ssl/certs
  volumes:
  - name: cacerts
    csi:
      driver: cacerts.csi.cert-manager.io
      readOnly: true
      volumeAttributes:
        os: debian
        caProviderClasses: ca-provider
```

## OS Distro Support

Different OS uses different files for trusted ca certificates. This driver has been tested against the following Linux distributions.

- alpine
- centos6
- centos7
- centos8
- debian
- fedora
- opensuse
- oraclelinux8
- oraclelinux7
- oraclelinux6
- rockylinux
- ubuntu
