# Security policy

## Supported versions

Security fixes are provided for the latest released version. The initial compatibility baseline is CPA v7.2.138 with CPAMP v1.12.2.

## Report a vulnerability

Please use GitHub's private vulnerability-reporting feature for this repository. Do not include real Management Keys, API keys, auth files, server addresses, or panel backups in a public issue.

## Security boundaries

- The plugin never requests, reads, stores, or forwards a CPA Management Key.
- Its unauthenticated browser resource exposes only the embedded Theme Studio landing page and loader JavaScript.
- Automatic injection accepts only regular `.html` files up to 64 MiB and rejects symbolic links.
- Panel changes are delimited by unique markers and replaced atomically.
- `panel_path` and `host_config_path` must be trusted local paths controlled by the operator.
- The plugin does not make Manager Server's embedded panel writable and does not bypass CPA or CPAMP authentication.
