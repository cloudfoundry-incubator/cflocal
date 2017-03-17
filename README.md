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
   stage    Build a droplet using the app in the current directory and the
               environment specified in local.yml.
               Droplet filename: <name>.droplet

   -b <url>    Buildpack URL (git or zip)
                  Default: (uses detection)
   -s <app>    Use the service credentials from the specified remote CF app instead
                  of the credentials in local.yml.
                  Default: (uses local.yml)
   -f <app>    Same as -s, but re-writes the service credentials to match what they
                  would be if they were tunneled through the app.
                  Default: (uses local.yml)

RUN OPTIONS:
   run      Run a droplet with using the environment specified in local.yml.
               Droplet filename: <name>.droplet

   -p <port>   Port on localhost for app to listen on 
                  Default: (arbitrary unused port)
   -d <dir>    Mount the specified directory into the app container at the app root.
                  If the directory is empty, the app is copied into the directory.
                  If the directory is not empty, the app is replaced by the directory.
                  Default: none
   -s <app>    Use the service credentials from the specified remote CF app instead
                  of the service credentials in local.yml.
                  Default: (uses local.yml or app provided by -f)
   -f <app>    Tunnel services through the specified remote CF app and rewrite service
                  credentials to use the tunnel. The service credentials from the
                  specified app will be used if -s is not also passed.
                  Default: (uses local.yml)

EXPORT OPTIONS:
   export   Export a standalone Docker image containing a droplet and the environment
               specified in local.yml.
               Droplet filename: <name>.droplet

   -r <ref>    Tag the exported image with the provided reference.
                  Default: none

PULL OPTIONS:
   pull     Download the droplet and the associated environment of a remote CF app.
               The local.yml file is updated with the environment.
               Droplet filename: <name>.droplet


PUSH OPTIONS:
   push     Push a droplet and (optionally) its associated environment from local.yml
               to a remote CF app.
               Droplet filename: <name>.droplet

   -e          Replace the remote app environment with the environment from local.yml.
                  This does not read or set environment variable groups.
                  Default: false
   -k          Do not restart the application after pushing the droplet.
                  The current droplet will continue to run until the next restart.
                  Default: false
```

NOTES:
 - For safety reasons:
    - Service forwarding tunnels are not active during staging
    - Containers are never exported with remote service credentials
    - Service credentials from remote apps are never stored in local.yml
