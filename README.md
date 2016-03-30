# sous

Sous is a tool for building, testing, and deploying applications, using
Docker, Mesos, and Singularity.

**UPDATE: You now require go 1.6+ to build this project.**

## Features

- Runs on Mac and Linux (Windows support not currently planned) -
Use the same tool for local development and in your CI pipeline
- Easily distribute shared configuration using the built-in sous
server - Automatically adds rich metadata to your Docker images - Run
executable contracts against any Docker image, to ensure it behaves
appropriately for your platform. - Define platform contracts in terms of
application interactions - Automatically build NodeJS and Go code using
a multi-stage build process that eliminates build-time dependencies from
your production containers. (Java, C#, Ruby, and other languages coming
soon.)

### Planned features

- Multi-datacentre deployment orchestration (coming very soon)
- Declarative YAML-based DSL to define deployments (coming very soon)
- Safely deploy source code to production Global event log HTTP API
- to interrogate, and instigate changes to global state Run projects
- locally in a simulated production environment

## Ethos

Sous is designed to work with existing projects, using data they already
contain to determine how to properly build Docker images. It is designed
to ease migrating existing projects onto The Mesos Platform, using
sensible defaults and stack-centric conventions. It is also designed
with operations in mind, tagging Docker images with lots of metadata to
ease discovery and clean-up of images.

Sous works on your local dev machine, and on CI servers like TeamCity,
Jenkins, etc., in the same way, so you can be sure whatever works
locally will also work in CI.

## Installation

Sous is written in Go. If you already have Go 1.6 set up on your
machine, and have your GOPATH set up correctly, you can install it by
typing

    $ go get -u -v github.com/opentable/sous

Alternatively, you can install the latest development version on your
Mac using homebrew:

 $ brew install --HEAD opentable/osx-tools/sous

We plan to begin releasing versioned pre-built binaries soon.

(for development guides, see [here](docs/contributing.md)

### Initial Setup

Currently, sous cannot do much without a sous server instance to provide
configuration. Therefore, the first command you'll need to issue is:

    sous config sous-server http://your.sous.server

More documentation on setting up the server will be coming soon, as well
as a better experience for working offline, before you have a server set
up.

## Requirements

Sous shells out to your system to interact with Git and Docker. This is
a design decision, as it enables you to easily repeat the commands Sous
issues. That means that when they fail, as they sometimes do, you have
the power to re-play what happened, and figure out the issue.

You will need:

- Git >=2.2 Docker >=1.10                                             .0

On Mac, I would recommend installing Docker by installing docker-machine
via the Docker Toolbox available at https://www.docker.com/toolbox

## Basic Usage

Sous is a CLI tool, with a subcommand-based interface inspired by Git.
All sous commands are of the form:

    sous [-v] <command> [command-options]

Where `-v` means "be verbose". This is a very useful option, especially
right now where the codebase is not stable, so things can frequently go
wrong. Being verbose means that you will see all the shell commands sous
issues, as well as other diagnostic information.

### Commands

```shell $ sous help Sous is a tool to help with building and testing
docker images, verifying your code against platform contracts, and
deploying to Singularity.

Commands: build build your project build-path build state directory
clean delete your project's containers and images config get/set config
properties contracts check project against platform contracts detect
detect available actions dockerfile print current dockerfile help show
this help image print last built docker image tag logs view stdout and
stderr from containers ls list images, containers and other artifacts
parse parse global state directory push push your project run run your
project server run the sous server stamp stamp labels onto docker images
state global deployment state task_host get task host task_port get
task port test test your project update update sous config version show
version info

Tip: for help with any command, use `sous help <COMMAND> ``            `


### Building

If you have a project written in NodeJS or Go, Sous may be able to build
that project automatically. The best place to start is to `cd` into your
project's directory, and type

    sous dockerfile

This will print to your terminal the dockerfile sous believes is
appropriate for your project. If you agree, and want to use that
dockerfile to build your project, it's as easy as:

    sous build

Sous build works by interrogating your Git repo to sniff out what kind
of project it is and some other info like its name, version, what
runtime version it needs etc. Using this data, it attempts to create
sensible Dockerfiles to perform various tasks like building and testing
your project. It also applies labels to the Dockerfile which propagate
through to the image, and finally the running containers, with data such
as which Git commit was built, what stack is running inside it, which
user and host it was built on, and a load more.

This approach is inspired by Heroku's buildpacks, but with a focus on
building efficient docker images.

#### Build targets

By default, sous buildpacks can specify a number of "build targets"
which are essentially specialised `Dockerfile`s. The most important of
these is the `app` target, which is your actual software, i.e. the thing
you would deploy to QA and Production.

`sous build` is shorthand for `sous build app` and will automatically
build any intermediary targets necessary to get from your source code to
a deployable application.

Commonly, buildpacks will specify a `compile` target as well. This is
used to build your project, and typically will be based on a heavier
Docker base image, which includes things like compilers, make, and other
tools which you only need at build time, not run time. Usually you would
not want to build this target by itself. However, you can build any of
the available targets by using:

    sous build <target>

### Contracts

If your configuration includes contracts, you can run them for your
current project by simply using:

    sous contracts

This will attempt to build your app, if changes are detected, and then
run the resultant docker image through the defined contracts.

If you want to run the contracts against an arbitrary docker image, you
can do this:

    sous contracts -image <image>

Replacing `<image>` with the name of the image you want to test.
