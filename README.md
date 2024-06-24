# Google Cloud Shell

Download the appropriate file for your platform from [releases](https://github.com/ixiumu/google-cloud-shell-cli/releases)

Create your own access [credentials](https://developers.google.com/workspace/guides/create-credentials) and save them to `~/.ssh/gcs_credentials.json`

Edit `~/.ssh/config` and add the following content to enable connecting to Google Cloud Shell using VSCode Remote SSH

```bash
Host cloudshell
    User <username>
    HostName 127.0.0.1
    Port 22
    IdentityFile ~/.ssh/id_rsa
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
    ProxyCommand gcs -i ~/.ssh/id_rsa -W %h:%p
```