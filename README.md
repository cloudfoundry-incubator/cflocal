# CF Local - cf CLI Plugin

```
NAME:
   local - Stage, launch, push, pull, and export CF apps -- in Docker

USAGE:
   cf local stage  <name> [(-b <URL>)] [-s <app> | -f <app>]
   cf local run    <name> [(-p <port>) (-d <dir>) (-s <app>) (-f <app>)]
   cf local export <name> [(-r <ref>)]
   cf local pull   <name>
   cf local push   <name> [-e] [-k]
   cf local help
   cf local version

STAGE OPTIONS:
   stage <name>   Build a droplet using the app in the current directory and
                     the environment variables and service bindings specified
                     in local.yml.
                     Droplet filename: <name>.droplet

   -b <url>       Use the specified buildpack URL (git or zip).
                     Default: (uses detection)
   -s <app>       Use the service bindings from the specified remote CF app
                     instead of the service bindings in local.yml.
                     Default: (uses local.yml)
   -f <app>       Same as -s, but re-writes the service bindings to match
                     what they would be if they were tunneled through the app
                     with cf local run <name> -f <app>.
                     Default: (uses local.yml)

RUN OPTIONS:
   run <name>     Run a droplet with the configuration specified in local.yml.
                     Droplet filename: <name>.droplet

   -p <port>      Listen on the specified port (localhost only)
                     Default: (arbitrary free port)
   -d <dir>       Mount the specified directory into the app at the app root.
                     If empty, the app root is copied into the directory.
                     If not empty, the app root is replaced by the directory.
                     Default: none
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
   DOCKER_HOST    Docker daemon address
                     Default: /var/run/docker.sock

SAMPLE: local.yml

applications:
- name: first-app
  command: "some start command"
  staging_env:
    SOME_STAGING_VAR: "some staging value"
  running_env:
    SOME_RUNNING_VAR: "some running value"
  env:
    SOME_VAR: "some value"
  services:
    (( contents of VCAP_SERVICES ))

- name: second-app
  ...
```

INSTALL:
```
$ ./cflocal-v0.2.0-macos
Plugin successfully installed. Current version: 0.2.0
```

NOTES:
 - CF Local does not require the Docker CLI. It communicates directly with the Docker daemon.
 - For safety reasons:
    - Service forwarding tunnels are not active during staging
    - Containers are never exported with remote service credentials
    - Service credentials from remote apps are never stored in local.yml

TODO:
 - Permit specification of cflinuxfs2 version
 - Permit offline usage