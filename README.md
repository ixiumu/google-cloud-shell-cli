# Google Cloud Shell

## Usage:

Create your own access [credentials](https://developers.google.com/workspace/guides/create-credentials) and save them to `~/.ssh/gcs_credentials.json`

Download the appropriate file for your platform from [releases](https://github.com/ixiumu/google-cloud-shell-cli/releases), rename it to `gcs`, and copy it to a directory in your system's `PATH`.

Create SSH key:

```bash
ssh-keygen -t ssh-rsa -f ~/.ssh/gcs_rsa
```

Add the public key to Google Cloud Shell:

```bash
gcs addPublicKey "ssh-rsa <your-public-key>"
```

Start an SSH connection:

```bash
gcs ssh
```

## VSCode Remote SSH

Edit `~/.ssh/config` and add the following content to enable connecting to Google Cloud Shell using VSCode Remote SSH.

Replace `<username>` with your Google ID. If your SSH key is password protected, you will need to enter it twice here.

```bash
Host cloudshell
    User <username>
    HostName 127.0.0.1
    Port 22
    IdentityFile ~/.ssh/gcs_rsa
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
    ProxyCommand gcs ssh -i ~/.ssh/gcs_rsa -W %h:%p
```
