# puppet-environment-cache-invalidate

![ci-build](https://github.com/riton/puppet-environment-cache-invalidate/workflows/ci-build/badge.svg)

## Status

This code is still under heavy development and subject to change.

## Description

CLI that invalidates the PuppetServer environment cache on multiple pre-configured servers simultaneously.

It calls the [Puppet Server: Admin API: Environment Cache](https://puppet.com/docs/puppetserver/latest/admin-api/v1/environment-cache.html#delete-puppet-admin-apiv1environment-cache)
API endpoint.

## Usage

```
Usage:
  puppet-environment-cache-invalidate [flags]

Flags:
      --config string   config file (default is $HOME/.puppet-environment-cache-invalidate.yaml)
      --debug           enable debug
  -h, --help            help for puppet-environment-cache-invalidate
      --log-json        log in JSON format
      --log-syslog      log to syslog
```

## Installation

### Build from source

```
$ go build .
```

### Pre-compiled binaries

This project is investigating _Github actions_ to build its packages.

You can find:
* pre-compiled binary for Linux
* pre-built RPM package

in [the action tab of this project](https://github.com/riton/puppet-environment-cache-invalidate/actions)

## Configuration

Configuration file is searched for in the following places:
* `/etc/puppet-environment-cache-invalidate.yaml`
* `$HOME/.puppet-environment-cache-invalidate/puppet-environment-cache-invalidate.yaml`

This _CLI_ relies on [viper](https://github.com/spf13/viper) so any configuration file format (_JSON, TOML, YAML, HCL, envfile and Java properties config files_) should be supported.

### Sample

```yaml
---
puppetservers:
  - 'puppetserver-01.example.org'
  - 'puppetserver-02.example.org'
  - 'puppetserver-03.example.org'

auth:
  certfile: '/etc/puppetlabs/puppet/certs/mynode.pem'
  pkfile: '/etc/puppetlabs/puppet/ssl/private_keys/mynode.key'
  ca-bundle: '/etc/puppetlabs/puppet/ssl/certs/ca.pem'
```

### Detail

#### puppetservers

* Type: `[]string`
* Description: List of puppetservers we should trigger environment cache invalidation for. **Required**.

#### auth.certfile

* Type: `string`
* Description: X509 client certificate file used to authenticate the HTTP request to the puppetserver admin API. **Required**.

#### auth.pkfile

* Type: `string`
* Description: Client private key used to authenticate the HTTP request to the puppetserver admin API. **Required**.

#### auth.ca-bundle

* Type: `string`
* Description: CA-Bundle file used to validate the puppetserver certificate. **Required**.
