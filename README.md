# CF Local - cf CLI Plugin

[![Linux Build Status](https://travis-ci.org/cloudfoundry-incubator/cflocal.svg?branch=master)](https://travis-ci.org/cloudfoundry-incubator/cflocal)
[![Windows Build Status](https://ci.appveyor.com/api/projects/status/tbaf399k1d60q78j/branch/master?svg=true)](https://ci.appveyor.com/project/sclevine/cflocal/branch/master)
[![GoDoc](https://godoc.org/code.cloudfoundry.org/cflocal?status.svg)](https://godoc.org/code.cloudfoundry.org/cflocal)

![CF Local Demo](https://raw.githubusercontent.com/cloudfoundry-incubator/cflocal/master/assets/cflocal-demo.gif) \
*Note: Image download/build only occurs when a new rootfs is available.*

CF Local is a Cloud Foundry CLI plugin that enables you to:

* Stage and run Cloud Foundry apps using Docker.
* Pull running apps from a remote Cloud Foundry and run them with Docker.
* Stage apps with Docker and push them to a remote Cloud Foundry.
* Seamlessly inherit the service bindings of remotely running Cloud Foundry apps.
* Seamlessly re-write service bindings to use persistent SSH tunnels through remote apps.
* Develop Cloud Foundry apps in Docker using live-reload functionality backed by Docker volumes.
* Rapidly iterate on Cloud Foundry apps without Cloud Foundry.
* Convert Cloud Foundry apps into Docker images that only require Docker to run.

Notably, CF Local:

* Does not require the Docker CLI
* Can run against a remote Docker daemon
* Uses the latest official Cloud Foundry buildpack releases by default
* Always uses the latest Cloud Foundry rootfs (cflinuxfs2) release
* Includes multi-buildpack support
* Supports specifying buildpacks by name, zip URL, git URL, and local zip path

```
USAGE:
   cf local stage   <name> [ (-b <name> | -b <URL> | -b <zip>)... -e ]
                           [ (-p <dir> | -p <zip>) (-s <app> | -f <app>) ]
                           [ (-s <app> | -f <app>) ]
   cf local run     <name> [ (-i <ip>) (-p <port>) (-s <app>) (-f <app>) ]
                           [ (-d <dir> [-w] | (-d <dir>) [-t]) ]
   cf local export  <name> [ (-r <ref>) ]
   cf local pull    <name>
   cf local push    <name> [-e -k]
   cf local help
   cf local version

STAGE OPTIONS:
   stage <name>   Build a droplet using the app in the current directory and
                     the environment variables and service bindings specified
                     in local.yml.
                     Droplet filename: <name>.droplet

   -b <name>      Use one or more official CF buildpacks (specified by name).
                     Default: (uses detection)
   -b <url>       Use one or more buildpacks specified by git repository URL
                     or zip file URL (HTTP or HTTPS).
                     Default: (uses detection)
   -b <zip>       Use one or more buildpacks specified by local zip file path.
                     Default: (uses detection)
   -e             If buildpacks are explicitly specified then select one of
                     them using the buildpack detection process instead of
                     applying all of them using the multi-buildpack process.
                     Default: false
   -p <dir>       Use the specified directory as the app directory.
                     Default: current working directory
   -p <zip>       Use the specified ZIP file contents as the app directory.
                     Note that JAR and WAR files use ZIP file format.
                     Default: current working directory
   -s <app>       Use the service bindings from the specified remote CF app
                     instead of the service bindings in local.yml.
                     Default: (uses local.yml)
   -f <app>       Same as -s, but re-writes the service bindings to match
                     what they would be if they were tunneled through the app
                     with: cf local run <name> -f <app>
                     Default: (uses local.yml)

RUN OPTIONS:
   run <name>     Run a droplet with the configuration specified in local.yml.
                     Droplet filename: <name>.droplet

   -i <ip>        Listen on the specified interface IP
                     Default: localhost
   -p <port>      Listen on the specified port
                     Default: (arbitrary free port)
   -d <dir>       Replace the app directory with the specified directory.
                     The app directory from the droplet is ignored.
                     Default: (not mounted)
   -w             When used with -d, restart the app when the contents of the
                     specified directory are changed.
                     Default: false, Invalid: with -t, without -d
   -t             Start a shell (Bash) with the same environment as the app.
                     Default: false, Invalid: with -w
   -s <app>       Use the service bindings from the specified remote CF app
                     instead of the service bindings in local.yml.
                     Default: (uses local.yml or app provided by -f)
   -f <app>       Tunnel service connections through the specified remote CF
                     app. This re-writes the service bindings in the container
                     environment in order to use the tunnel. The service
                     bindings from the specified app will be used if -s is not
                     also passed.
                     Default: (uses local.yml)

EXPORT OPTIONS:
   export <name>  Export a standalone Docker image using the specified droplet
                     and configuration from local.yml.
                     Droplet filename: <name>.droplet

   -r <ref>       Tag the exported image with the provided reference.
                     Default: none

PULL OPTIONS:
   pull <name>    Download the droplet, environment variables, environment
                     variable groups, and start command of the named remote
                     CF app. The local.yml file is updated with the downloaded
                     configuration.
                     Droplet filename: <name>.droplet

PUSH OPTIONS:
   push <name>    Push a droplet to a remote CF app and restart the app.
                     Droplet filename: <name>.droplet

   -e             Additionally replace the remote app environment variables
                     with the environment variables from local.yml. This does
                     not read or replace environment variable groups.
                     Default: false
   -k             Do not restart the application after pushing the droplet.
                     The current droplet will continue to run until the next
                     restart.
                     Default: false

ENVIRONMENT:
   CFL_USE_PROXY  Always use or never use the environment's proxy settings.
                     Default: (use only when DOCKER_HOST is not set)
   DOCKER_HOST    Docker daemon address
                     Default: /var/run/docker.sock

SAMPLE: local.yml

applications:
- name: first-app
  buildpacks:
  - some_buildpack
  - some_other_buildpack
  command: "some start command"
  memory: 2G
  disk_quota: 4G
  staging_env:
    SOME_STAGING_VAR: "some staging value"
  running_env:
    SOME_RUNNING_VAR: "some running value"
  env:
    SOME_VAR: "some value"
  services:
    (( VCAP_SERVICES object in YAML ))
```

## Install

### From a Downloaded Release
```bash
$ ./cflocal-v0.19.0-macos
Plugin successfully installed. Current version: 0.19.0
```
***Or***
```bash
$ cf install-plugin cflocal-0.19.0-macos
Attention: Plugins are binaries written by potentially untrusted authors.
Install and use plugins at your own risk.
Do you want to install the plugin cflocal-0.19.0-macos? [yN]: y
Installing plugin cflocal...
OK
Plugin cflocal 0.19.0 successfully installed.
```

### From the Community Plugin Repository
```bash
$ cf install-plugin cflocal
Searching CF-Community for plugin cflocal...
Plugin cflocal 0.19.0 found in: CF-Community
Attention: Plugins are binaries written by potentially untrusted authors.
Install and use plugins at your own risk.
Do you want to install the plugin cflocal? [yN]: y
Starting download of plugin binary from repository CF-Community...
 14.35 MiB / 14.35 MiB [=====================================] 100.00% 2s
Installing plugin cflocal...
OK
Plugin cflocal 0.19.0 successfully installed.
```
Note: This version is occasionally out of date.
Furthermore, if you are using a version of the CF CLI prior to `v6.27.0`,
you will need to specify the repository where the plugin is located:
```bash
$ cf install-plugin cflocal -r CF-Community
```

## Uninstall

```
$ cf uninstall-plugin cflocal
Uninstalling plugin cflocal...
OK
Plugin cflocal successfully uninstalled.
```

## Security Notes

* Forwarded services (`-f`) are not reachable during staging.
* Images are never exported with remote service credentials.
* Service credentials from remote apps are never stored in local.yml.
* CF Local should not be used to download untrusted Cloud Foundry applications.
* CF Local is not intended for production use and is offered without warranty.
* CF Local distribution archives are [signed by me](https://keybase.io/sclevine).

