# sous [![Build Status](https://secure.travis-ci.org/opentable/sous.png?branch=master)](http://travis-ci.org/opentable/sous) [![Report card](https://goreportcard.com/badge/github.com/opentable/sous)](https://goreportcard.com/report/github.com/opentable/sous)
Sous is a tool for building, testing, and deploying applications, using
Docker, Mesos, and Singularity.

(this is a dummy change)

If you just want to use Sous to build and deploy your code, follow the [installation instructions](./doc/install.md).

For contribution guidelines, see [here](./doc/contributions.md).

[View documentation in the doc/ directory.](https://github.com/opentable/sous/tree/master/doc)

# Using Sous

If you're looking to get started using Sous
to manage your service within a larger organization, read on.

### Bleeding edge development

Sous is written in Go.
Once you have [Go set up on your machine,](./doc/setting-up-go.md)
you can install it by typing:

```bash
$ go get -u -v github.com/opentable/sous
```

However, for normal use,
we recommend that you use a release,
rather than fritter away development time
on our QA.
Also, while we'll do our best,
we'll be most able to help with bugs on released versions.

## Client Configuration

To use Sous effectively,
you'll need to know the URL of at least one running Sous server.
(If you're looking to set up a new Sous deployment,
see [this guide.](./doc/first-deployment-of-sous.md))
That URL will be particular to your organization,
but someone should be able to provide it to you.

Once you have it, run:
```bash
$ sous config server <URL>
```

(...replacing `<URL>` with the URL you were given.)

You should be good to go, but if you're curious,
client configuration is documented [here](./doc/client-config.md).

The **Installation** and **Client Configuration** steps
should only need to be done once on any given workstation.

## Hello sous

Now that you have a Sous client set up,
let's add a project to Sous management.

The following commands will contact the Sous server and create your project in every known cluster:

```bash
# Enter the directory of your project.
$ cd <my-project>

# Connect to the Sous server and register the project's existence.
$ sous init
```

You can limit this to a single cluster by
replacing the last command with `sous init -cluster <name>`

Sous will provide a list of known clusters if you give it bad input.

To add or remove your project from available clusters, use `sous manifest get > manifest.yaml` to download the current state of deployments. After editing the returned yaml file, use `sous manifest set < manifest.yaml` to send the changes to your Sous server.

Since there's no Docker image that corresponds
to this project yet, Sous won't actually try to deploy
your project yet.

## Day to day

From then on, the process is similar,
but you don't need to do the `sous init`
since Sous already knows about the project.

Instead, you'll want to use `sous deploy`, like this:

```bash
# Sous requires that projects be tagged with a
# semantic version.
$ git tag -a 1.2.4 -m "sous build tag" && git push --tags

# Actually do the docker build, push and registration steps
$ sous build

# Update Sous' view of the world so that it knows you want to
# deploy the version you built.
$ sous deploy -tag 1.2.4 -cluster <name>
```

As soon as you've completed the `deploy` step,
Sous will be ready to deploy your service.
You should see it deployed and running in a few seconds.

Note that these steps can be easily adapted to work
on continuous integrations servers, as well.

## Requirements

Sous shells out to your system to interact with Git and Docker. This is
a design decision, as it enables you to easily repeat the commands Sous
issues. That means that when they fail, as they sometimes do, you have
the power to re-play what happened, and figure out the issue.

You will need:

- Git >=2.2
- Go >= 1.7
- Docker >=1.10

On Mac, we recommend installing Docker by installing docker-machine
via the Docker Toolbox available at https://www.docker.com/toolbox

## LICENSE

MIT
