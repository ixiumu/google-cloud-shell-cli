# Google Cloud Shell

```bash
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build
```

```bash
Host cloudshell
    User <username>
    HostName 127.0.0.1
    Port 22
    IdentityFile ~/.ssh/id_rsa
    ProxyCommand gcs -i ~/.ssh/id_rsa -W %h:%p
```
