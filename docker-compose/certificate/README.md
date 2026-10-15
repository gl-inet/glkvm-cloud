# Web certificate (optional)

Put your own certificate here to serve the web UI with it:

- `glkvm.cer` — certificate (full chain)
- `glkvm.key` — private key

If they are missing, GLKVM Cloud generates a self-signed certificate on first
start, so browsers show a warning when you open the platform by IP.

Devices never use this certificate. The device ports use one signed by the
platform's own CA, which is generated in `database/pki` — keep that directory
and back it up: devices that joined trust that CA, and lose access if it is
replaced.
