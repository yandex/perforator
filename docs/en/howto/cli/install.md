# Install Perforator CLI

Build `perforator/cmd/cli` from the root of the repository.

# Configure Perforator CLI

* Set `PERFORATOR_URL` environment variable to specify the Perforator server URL once. Otherwise you need to use `--url` flag for each command.
* Set `PERFORATOR_SECURE` to specify the security level of the connection. Default is secure.


```console
export PERFORATOR_URL="https://perforator.example.com"
export PERFORATOR_SECURE=true
```

