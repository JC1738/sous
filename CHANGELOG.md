# Sous Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/)
with respect to its command line interface and HTTP interface.

## [Unreleased](//github.com/opentable/sous/compare/0.5.62...HEAD)

### Added
* CLI: If no image is present in runspec, return a fatal flaw in build.
* Server: adding duplex state storage, to keep DB in sync until ready to switch over
* Server: Update logging to a more structured format: server, resource, Generic Msg, handle_gdm, volume, local_config, router

### Changed
* All: error parsing repo from SourceLocation now more informative.

### Fixed
* All: Data race in rectification queue.

## [0.5.62](//github.com/opentable/sous/compare/0.5.61...0.5.62)

### Added
* Server: If default log level is invalid, use Extreme level and print out a message.
* Logging: sous-diff-resolution logs include two new fields 'sous-diff-source-type' and
  'sous-diff-source-user', hard-coded to 'global rectifier' and 'unknown'.

### Fixed
* All: Setting a manifest with Owners field not alphabetically sorted no longer prevents updates
  to other manifests. (Owners are always stored in alphabetical order now.)
* Server: Kafka config validation now correctly checks if brokers were specified when Kafka is enabled.
* Server: Changing the Owners list now correctly causes the Singularity request to be updated.

## Internal
* Server: Storage module for Postgres uses placeholders correctly. As yet, not connected up.

## [0.5.61](//github.com/opentable/sous/compare/0.5.60...0.5.61)

### Added
* Server: Ability to load sous sibiling urls from environment variable

## [0.5.60](//github.com/opentable/sous/compare/0.5.59...0.5.60)

This release contains internal changes only; external behaviour is unmodified.
See [the set of commits in this release](//github.com/opentable/sous/compare/0.5.59...0.5.60)
for details of developer-facing changes.

## [0.5.59](//github.com/opentable/sous/compare/0.5.58...0.5.59)
### Fixed
* Server: Docker repository HTTP metrics collected and logged.
* Server: Sizes of *response* bodies logged and reported to Graphite.
* Server: Backtrace on HTTP logging excludes our libraries.

## [0.5.58](//github.com/opentable/sous/compare/0.5.57...0.5.58)

### Added
* Server: HTTP client requests to Singularity log to Kafka
* Server: HTTP client requests to the Docker registry log to Kafka
* Server: HTTP server responses log to Kafka
* Server: Additional metrics reported per request kind, per host and per kind,host pair.

## [0.5.57](//github.com/opentable/sous/compare/0.5.56...0.5.57)
### Fixed
* CLI + Server: No longer panics when printing a nil Deployment.

* Server: Profiling configuration - accepts SOUS_PROFILING environment variable correctly.

### Changed
* Developer: Server action extracted.

## [0.5.56](//github.com/opentable/sous/compare/0.5.55...0.5.56)
### Fixed
* CLI: No longer panics under normal operation (e.g. when trying to run 'sous deploy'
  outside of a git repo, or when server connection fails etc).
* CLI + Server: Logging goes to Kafka and Graphite when configured to again.

## [0.5.55](//github.com/opentable/sous/compare/0.5.54...0.5.55)
### Changed
* CLI: Quieter output for local operators. Previously many log messages were
  emitted to stderr which made CLI use difficult due to information overload.

### Fixed
* CLI: -s -q -v and -d (silent, quiet, verbose, and debug) flags now set the
  logging level appropriately. You'll now see a lot more output when using
  -d and -v flags, and only critical errors from -s, critical and console output
  from -q.

## [0.5.54](//github.com/opentable/sous/compare/0.5.53...0.5.54)
### Added
* All: a Schedule field on Manifests, which should publish to Singularity to allow for scheduled tasks.

### Fixed
* All: a bug in the Kafka subcomponent meant that the reverse sense was being applied to log entries.
  Furthermore, the severity was wrong, and the messages were being omitted.

## [0.5.53](//github.com/opentable/sous/compare/0.5.52...0.5.53)

### Fixed
* Server: the resolve complete message is properly emitted and produces stats

## [0.5.52](//github.com/opentable/sous/compare/0.5.51...0.5.52)

### Changed
* All: Logging to Kafka no longer goes through logrus.

### Added
* Client: console output related to deploys. Reports the number of instances
  intended, and reflects the successfully deployed source ID and target
  cluster.

## [0.5.51](//github.com/opentable/sous/compare/0.5.50...0.5.51)

### Added
* Server: max concurrent HTTP requests per Singularity server now configurable
  using `MaxHTTPConcurrencySingularity` in config file or
  `MAX_HTTP_CONCURRENCY_SINGULARITY` env var. Default is 10 was previously
  hard-coded to 10, so this is an opt-in change.

### Changed
* All: some logging behaviors - there may be more output than we'd like
* All: logging - capture the CLI output to Kafka, test that diff logs are generated.
* All: backtrace selection more accurate (i.e. the reported source of log entries is generally where they're actually reported)

## [0.5.50](//github.com/opentable/sous/compare/0.5.49...0.5.50)
### Fixed
* All: log entries weren't conforming to the requirements of our schemas.

## [0.5.49](//github.com/opentable/sous/compare/0.5.48...0.5.49)
### Fixed
* Server: at least one botched log message type has been caught and corrected.

### Added
* Client: more descriptive output from 'sous deploy' and 'sous update' commands.

## [0.5.48](//github.com/opentable/sous/compare/0.5.47...0.5.48)

### Fixed
* Erroneous output on stdout breaking some cli consumers.


## [0.5.47](//github.com/opentable/sous/compare/0.5.46...0.5.47)

### Fixed
Both: DI interaction causing panic on boot.

## [0.5.46](//github.com/opentable/sous/compare/0.5.44...0.5.46)

### Added
* Client: a separate status for "API requests are broken".
* Client: many new log messages during `sous update`, `sous deploy` and `sous plumbing status`.
* Client: general log message for start and end of command execution.

### Fixed
* Client: a panic would occur if the remote server wasn't available or responded with a 500.
* Attempting to fix invalid config using 'sous config' was not possible because we
  were validating config as a precondition to executing the command. We now only
  validate config for commands that rely on a valid config, so 'sous config' can be
  used to correct issues.

## [0.5.44](//github.com/opentable/sous/compare/0.5.43...0.5.44)

### Fixed

* Server: diff messages could panic when logging them if the diff didn't resolve correctly.
* All: logging panics would crash the app.
* Client: 'sous deploy' now waits for a complete resolution to take place before
  reporting failure. This avoids a race condition where earlier failures could
  be misreported as failures with the current deployment.
* Client: 'sous deploy' now bails out if no changes are detected after the present
  resolve cycle has completed, or if the latest version in the GDM does not match that
  expected. This solves an issue where deployments would appear to hang for a long time
  and eventually fail with a confusing error message, often due to conflicting updates.

## [0.5.43](//github.com/opentable/sous/compare/0.5.42...0.5.43)

### Added
* Server: deployment diffs are now logged as structured messages.
* Client: `sous init` command now has `-dryrun` flag, so you can generate a manifest
  without sending it to the server. This flag interacts with the `-flavor`,
  `-use-otpl-deploy` and `-ignore-otpl-deploy` flags as well, so you can check sous'
  intentions in all these scenarios without accidentally creating manifests you don't want.

### Fixed
* Server: Changing Startup.SkipCheck now correctly results in a re-deploy with the
  updated value.
* Client: commands 'sous deploy', 'sous manifest get' and 'sous manifest set' now receive the correct auto-detected offset
  so you no longer require the -offset flag in most cases (unless you need to override it).

## [0.5.42](//github.com/opentable/sous/compare/0.5.41...0.5.42)
### Fixed
* All: Graphite output was like `sous.sous.ci.sfautoresolver.fullcycle-duration.count`, now `sous.ci.sf.auto...`

## [0.5.41](//github.com/opentable/sous/compare/0.5.40...0.5.41)

### Fixed

* All: ot_* fields are actually populated - also instance-no
* All: metrics are scoped to env and region so that multiple sous instances don't
  clobber each other's metrics.
* All: component-id refers to the whole "sous" application; scoped loggers goes in logger-name

## [0.5.40](//github.com/opentable/sous/compare/0.5.39...0.5.40)

### Fixed

* Server: mismatches with logging schemas

## [0.5.39](//github.com/opentable/sous/compare/0.5.38...0.5.39)

### Fixed
* Server: logging configuration actually gets applied,
  so we can get graphite and kafka feeds.
* Server: Various places that log entries were tweaked to conform to ELK schema.

## [0.5.38](//github.com/opentable/sous/compare/0.5.37...0.5.38)

### Fixed
* All: restores -d and -v flags

## [0.5.37](//github.com/opentable/sous/compare/0.5.36...0.5.37)

### Fixed
* All: Cloned DI providers ("psyringe") were resulting in 2+ NameCaches, and
  uncontrolled access to the Docker registry cachce DB. A race condition led to
  errors that prevented deployment of Sous, and then blocked use of the CLI
  client. A stopgap was set up to force a NameCache to be provided early.

## [0.5.36](//github.com/opentable/sous/compare/0.5.35...0.5.36)

### Fixed
- Client: If only one otpl config found with no flavor, and a flavor is specified the found config was used.
- Client: If only one otpl config found for only one flavor, and no flavor or different flavor specified the found config was used.
- Client: `sous plumbing normalizegdm` broke the DI rules and added
  `DryrunNeither` an extra time, which led to a panic.
- Server: Initial database grooming had a race condition. Solved by ensuring NameCache is singular.

### Changed
- Client: Improve testability of default OT_ENV_FLAVOR logic and test.

## [0.5.35](//github.com/opentable/sous/compare/0.5.34...0.5.35)

### Fixed
- Client and server: various logging output is clearer and more correct.
- Client: Flavor flag wasn't being passed to otpl deploy logic.
- Client: Panic when setting OT_ENV_FLAVOR env variable if Env was unset.

## [0.5.34](//github.com/opentable/sous/compare/0.5.33...0.5.34)
### Added
- Client: When calling `sous init -flavor X` automatically add OT_ENV_FLAVOR value to the Env variables

### Fixed
- Client: `sous init -use-otpl-deploy` wasn't handling flavors properly.
- Client: `sous init -ignore-otpl-deploy` caused panic.
- Server: Logging wasn't being properly initialized and crashed on boot.

## [0.5.33](//github.com/opentable/sous/compare/0.5.32...0.5.33)
### Added
- Structured logging. Default logging will be structured (and colorful!)
  because of using logrus. Configuration should allow delivery of metrics to
  graphite and log entries to ELK.

### Fixed
- Server: Flavor metadata wasn't being pulled back into the ADS during rectify, which led to bad behaviors in deploy.

## [0.5.32](//github.com/opentable/sous/compare/0.5.31...0.5.32)
### Changed
- Developer: Refactors of filters and logging. Mostly to the good.

### Fixed
- Client: `sous build` for split containers was adding a path component,
  which broke the resulting deploy container.

## [0.5.31](//github.com/opentable/sous/compare/0.5.30...0.5.31)
### Fixed
- Client: `sous init` defaults resources correctly in the absence of other input.
## [0.5.30](//github.com/opentable/sous/compare/0.5.29...0.5.30)
### Fixed
- Client: certain commands were missing DI for a particular value. These are
  fixed, and tests added for the omission.
- Client: new values for optional fields no longer elided on `manifest set` -
  if the value was missing, the new value would be silently dropped.

## [0.5.29](//github.com/opentable/sous/compare/0.5.28...0.5.29)
### Changed
- Developer: DI system no longer used on a per-request basis.

## [0.5.28](//github.com/opentable/sous/compare/0.5.27...0.5.28)
### Fixed
- Client & Server: rework of DI to contain scope of variable assignment,
  and retain scope from CLI invocation to server.
- Client: multiple target builds weren't getting their offsets recorded correctly.
- Developer: dependency injection provider now an injected dependency of the CLI object.

## [0.5.27](//github.com/opentable/sous/compare/0.5.26...0.5.27)
### Fixed
- Client: omitting query params on update.
- Client: Failed to merge maps to null values (which came from original JSON).

## [0.5.26](//github.com/opentable/sous/compare/0.5.25...0.5.26)
### Fixed
- Client: using wrong name for the /manifest endpoint

## [0.5.25](//github.com/opentable/sous/compare/0.5.24...0.5.25)
### Fixed
- Client: simple dockerfile builds were crashing

## [0.5.24](//github.com/opentable/sous/compare/0.5.23...0.5.24)
### Fixed
- Client: the `sous manifest get` and `set` commands correctly accept a `-flavor` switch.

## [0.5.23](//github.com/opentable/sous/compare/0.5.22...0.5.23)
### Added:
- Client: Split buildpacks can now provide a list of targets,
  and produce all their build products in one `sous build`.
### Changed:
- Client: Client commands now have a "local server" available if no server is
  configurated. This is the start of the path to using HTTP client/server
  interactions for everything, as opposed to strategizing storage modules.

## [0.5.22](//github.com/opentable/sous/compare/0.5.21...0.5.22)
### Fixed
- Server: Validation checks didn't consider default values.

## [0.5.21](//github.com/opentable/sous/compare/0.5.20...0.5.21)
### Added
- Server: Startup field "SkipCheck" added - rather than omitting a
  healthcheck URI, services must set this field `true` in order to signal that
  they don't make use of a "ready" endpoint.
- Server: Full set of Singularity 0.15 HealthcheckOptions now supported.
  Previous field names on Startup retained for compatibility,
  although there may be nuanced changes in how they're interpreted
  (because Singularity no longer supports the old version.)
- Server: Logging extended to collect metrics.
- Server: Metrics exposed on an HTTP endpoint.
- Developer: LogSet extracted to new util/logging package, some refiguring of
  the types and interfaces there with an eye to pulling in a structured logger.
- Client: Command `sous plunbing normalizegdm` is a utility to round-trip the
  Sous storage format. With luck, running this command after manual changes to
  the GDM repo will correct false conflicts.
### Changed
- Server: Startup DeployConfig part fields now have cluster-based default
  values. (These should be configued in the GDM before this version is
  deployed!)
  Most notably, the CheckReadyProtocol needs a default ("HTTP" or "HTTPS")
  because Singularity validates the value on its side.

## [0.5.20](//github.com/opentable/sous/compare/0.5.19...0.5.20)
### Fixed
- Client: `sous deploy` wasn't recognizing its version if there was a prefix supplied.

## [0.5.19](//github.com/opentable/sous/compare/0.5.18...0.5.19)

Only changes in this release are related to deployment.

### Fixed
- Developer: deployment key changed to a machine account.
## [0.5.18](//github.com/opentable/sous/compare/0.5.16...0.5.18)
### Added
- Client: `sous query clusters` will enumerate all the logical clusters sous currently handles, for ease of manifest editing and deployment.

### Changed
- Developer: Makefile directives to build a Sous .deb and upload it to an Artifactory server.
- Server: /gdm now sets Etag header based on current revision of the GDM. Clients attempting
  to update with a stale Etag will have update rejected appropriately.
- Server: Updated Singularity API consumer to work with Singularity v0.15.

### Fixed
- Client: Increased buffer size in shell to handle super-long tokens.
- Client: Conflicting updates to the same manifest now fail appropriately (e.g. during 'sous deploy').
  This should mean better performance when running 'sous deploy' concurrently on the same manifest,
  as conflicting updates won't require multiple passes to resolve.


## [0.5.17]
__(extra release because of mechanical difficulties)__

## [0.5.16](//github.com/opentable/sous/compare/0.5.15...0.5.16)
- Fixed prefixed versions for update and deploy. At this point, git "package" tags should work everywhere they're used.

## [0.5.15](//github.com/opentable/sous/compare/0.5.14...0.5.15)
### Changed
- Panics during rectify print the stack trace along with the error message in the logs.
  Previously the stack trace was printed earlier in the log, making correlation
  difficult.
- Server: All read/write access to the GDM now serialised.
  This is to partially address and issue where concurrent calls to 'sous deploy'
  could result in one of them finishing in the ResolveNotIntended state.

### Fixed
- Client: 'sous build' was failing when using a semver tag with a non-numeric prefix.
  Validation logic is now shared, so 'sous build' succeeds with these tags.
- Non-destructive updates: clients won't clobber fields in the API they don't recognize. Result should be more stable, less coupled client-server relationship.

## [0.5.14](//github.com/opentable/sous/compare/0.5.13...0.5.14)

### Fixed
- A change to the Singularity API was breaking JSON unmarshaling. We now handle those errors as a "malformed" request - i.e. not Sous's to manage.

## [0.5.13](//github.com/opentable/sous/compare/0.5.12...0.5.13)

### Added
- Git tags with a non-numeric prefix and a semver suffix (e.g. 'release-1.2.3' or 'v2.3.4')
  are now considered a "semver" tag, and Sous will extract the version from them.
- Static analysis of important core data model calculations to ensure that all the components of those structures are at least "touched" during diff calculation.
- For developers only, there are 2 new build targets: `install-dev` and
  `install-brew`. These allow developers on a Mac to quickly switch between having
  a personal dev build, or the latest release from homebrew installed locally.

### Fixed
- Operations that change more than one manifest will now be rejected with an
  error. We do not believe there are any such legitimate operations, and
  there's a storage anomoly that surfaces as multiple manifests changing at
  once which we hope this will correct.
- 'sous manifest get' wrongly returned YAML with all lower-cased field names.
  Now it correctly returns YAML with upper camel-cased field names.
  Note that this does not apply to map keys, only struct fields.
- Deployment processing wasn't properly waited on, which could cause problems.

## [0.5.12](//github.com/opentable/sous/compare/0.5.11...0.5.12)
### Fixed
- Issue where deployments constantly re-deployed due to spurious Startup.Timeout diff.

## [0.5.11](//github.com/opentable/sous/compare/0.5.10...0.5.11)
### Fixed
- Singularity now accepts changes to Startup options in manifest.
- Off-by-one error with Singularity deploy IDs, fixed in 0.5.9, re-introduced in
  0.5.10. Now includes better tests surrounding edge cases.

## [0.5.10](//github.com/opentable/sous/compare/0.5.9...0.5.10)
### Fixed
- Off-by-one error with long request IDs.
- Startup information not recovered from Singularity, so not compared for deployment.

## [0.5.9](//github.com/opentable/sous/compare/0.5.8...0.5.9)
### Fixed
- Long version strings resulted in Singularity deploy IDs longer than the max
  allowed length of 49 characters. Now they are always limited to 49.

## [0.5.8](//github.com/opentable/sous/compare/0.5.7...0.5.8)
### Fixed
- Now builds and runs on Go 1.8 (one small change to URL parsing broke Sous for go 1.8).
- New Startup configuration section in manifests now correctly round-trips via 'sous
  manifest get|set' and takes part in manifest diffs.

## [0.5.7](//github.com/opentable/sous/compare/0.5.6...0.5.7)
### Changed
- Images built with Sous get a pinning tag that now includes the timestamp of
  the build, so that multiple builds on a single revision won't clobber labesls
  and make images inaccessible.

## [0.5.6](//github.com/opentable/sous/compare/0.5.5...0.5.6)
### Fixed
- Sous server was unintentionally filtering out manifests with non-empty offsets or flavors.

## [0.5.5](//github.com/opentable/sous/compare/0.5.4...0.5.5)

### Fixed
- Resolution cycles allocate much less memory, which hopefully keeps the memory headroom of Sous much smaller over time.

## [0.5.4](//github.com/opentable/sous/compare/0.5.3...0.5.4)

### Added

- Sous server now returns CORS headers so that the Sous SPA (forthcoming) can consume its data.

### Fixed

- Crashing bug on GDM updates.

## [0.5.3](//github.com/opentable/sous/compare/0.5.2...0.5.3)

### Added
- Profiling endpoints, gated with a `server` flag, or the SOUS_PROFILING env variable.

### Fixed
- Environment variable defaults from cluster definitions
  no longer elide identical variables on manifests,
  which means that common values can be added to the defaults
  without undue concern for manifest environment variables.

## [0.5.2](//github.com/opentable/sous/compare/0.5.1...0.5.2)

### Added
- Extra debug logging about how build strategies are selected.
- Startup options in manifest to set healthcheck timeout and relative
  URI path of healthcheck endpoint.

### Changed
- Singularity RequestIDs are generated with a suffix of the MD5 sum of
  pre-slug data instead of a random UUID.
- Singularity RequestIDs are shortened to no longer include FQDN or
  organization of Git repo URL.

### Fixed
- Calls to `docker build` now have a `--pull` flag so that stale cached FROM
  images don't confuse builds.

## [0.5.1](//github.com/opentable/sous/compare/0.5.0...0.5.1)

### Fixed
- Singularity RequestIDs retrieved from Singularity are reused when updating deploys,
  instead of recomputing fresh unique ones each time.

### Minor
- Added a tool called "danger" to do review of PRs.

## [0.5.0](//github.com/opentable/sous/compare/0.4.1...0.5.0)

### Added
* Split image build strategy: support for using a build image to produce artifacts to be run
  in a separate deploy image.

### Changed
* Sous detects the tasks in its purview based on metadata it sets when the task
  is created, rather than inspecting request or deploy ids.

### Fixed
* Consequent to detecting tasks based on metadata,
  Sous's requests are now compatible
  with Singularity 0.14,
  and the resulting Mesos Task IDs are suitable to use as Kafka client ids.

## [0.4.1](//github.com/opentable/sous/compare/0.4.0...0.4.1)

### Fixed
- Status for updated deploys was being reported as if they were already stable.
  The stable vs. live statuses reported by the server each now have their own
  GDM snapshot so that this determination can be made properly.

## [0.4.0](//github.com/opentable/sous/compare/0.3.0...0.4.0)

### Added
- Conflicting GDM updates now retry, up to the number of deployments in their manifest.

### Changed
- Failed deploys to Singularity are now retried until they succeed or the GDM
  changes.

## [0.3.0](//github.com/opentable/sous/compare/0.2.1...0.3.0)

### Added
- Extra metadata tagged onto the Singularity deploys.
- `sous server` now treats its configured Docker registry as canonical, so
  that, e.g. regional mirrors can be used to reduce deploy latency.

### Changed

- Digested Docker image names no longer query the registry, which should reduce
  our requests count there.

## [0.2.1](//github.com/opentable/sous/compare/0.2.0...0.2.1)

### Added

- Adds Sous related-metadata to Singularity deploys for tracking and visibility purposes.

### Fixed

- In certain conditions, Sous would report a deploy as failed before it had completed.

## [0.2.0](//github.com/opentable/sous/compare/0.1.9...0.2.0) - 2017-03-06

### Added

- 'sous deploy' now returns a nonzero exit code when tasks for a deploy fail to start
  in Singularity. This makes it more suitable for unattended contexts like CD.

### Fixed

- Source locations with empty offsets and flavors no longer confuse 'sous plumbing status'.
  Previously 'sous plumbing status' (and 'sous deploy' which depends on it) were
  failing because they matched too many deployments when treating the empty
  offset as a wildcard. They now correctly treat it as a specific value.
- 'sous build' and 'sous deploy' previously appeared to hang when running long internal
  operations, they now inform the user there will be a wait.


## [0.1.9](//github.com/opentable/sous/compare/0.1.8...0.1.9) - 2017-02-16

### Added

- 'sous init -use-otpl-deploy' now supports flavors
  defined by otpl config directories in the `<cluster>.<flavor>` format.
  If there is a single flavor defined, behaviour is like before.
  Otherwise you may supply a -flavor flag to import configurations of a particular flavor.
- config.yaml now supports a new `User` field containing `Name` and `Email`.
  If set this info is sent to the server alongside all requests,
  and is used when committing state changes (as the --author flag).
- On first run (when there is no config file),
  and when a terminal is attached
  (meaning it's likely that a user is present),
  the user is prompted to provide their name and email address,
  and the URL of their local Sous server.
- `sous deploy`
  (and `sous plumbing status`)
  now await Singularity marking the indended deployment as active before returning.

### Fixed
- Deployment filters (which are used extensively) now treat "" dirs and flavors
  as real values, rather than wildcards.

## [0.1.8](//github.com/opentable/sous/compare/v0.1.7...0.1.8) - 2017-01-17

### Added
- 'sous init' -use-otpl-config now imports owners from singularity-request.json
- 'sous update' no longer requires -tag or -repo flags
  if they can be sniffed from the git repo context.
- Docs: Installation document added at doc/install.md

### Changed

- Logging, Server: Warn when artifacts are not resolvable.
- Logging: suppress full deployment diffs in debug (-d) mode,
  only print them in verbose -v mode.
- Sous Version outputs lines less than 80 characters long.

### Fixed

- Internal error caused by reading malformed YAML manifests resolved.
- SourceLocations now return more sensible parse errors when unmarshaling from JSON.
- Resolve errors now marshalled correctly by server.
- Server /status endpoint now returns latest status from AutoResolver rather than status at boot.

## [0.1.7](//github.com/opentable/sous/compare/v0.1.6...v0.1.7) 2017-01-19

### Added

- We are now able to easily release pre-built Mac binaries.
- Documentation about Sous' intended use for driving builds.
- Change in 'sous plumbing status' to support manifests that deploy to a subset of clusters.
- 'sous deploy' now waits by default until a deploy is complete.
  This makes it much more useful in unattended CI contexts.

### Changed

- Tweaks to Makefile and build process in general.

## [0.1.6](//github.com/opentable/sous/compare/v0.1.5...v0.1.6) 2017-01-19

Not documented.

## [0.1.5](//github.com/opentable/sous/compare/v0.1.4...v0.1.5) 2017-01-19

Not documented.

## [0.1.4](//github.com/opentable/sous/compare/v0.1.3...v0.1.4) 2017-01-19

Not documented.

## Sous 0.1

Sous 0.1 adds:
- A number of new features to the CLI.
  - `sous deploy` command (alpha)
  - `-flavor` flag, and support for flavors (see below)
  - `-source` flag which can be used instead of `-repo` and `-offset`
- Automatic migrations for the Docker image name cache.
- Consistent identifier parse and print round-tripping.
- Updates to various pieces of documentation.
- Nicer Singularity request names.

## Consistency

- Changes to the schema of the local Docker image name cache database no longer require user
  intervention and re-building the entire cache from source. Now, we track the schema, and
  migrate your cache as necessary.
- SourceIDs and SourceLocations now correctly round-trip from parse to string and back again.
- SourceLocations now have a single parse method, so we always get the same results and/or errors.
- We somehow abbreviated "Actual Deployment Set" "ADC" ?! That's fixed, we're now using "ADS".

### CLI

#### New command `sous deploy` (alpha)

Is intended to provide a hook for deploying single applications from a CI context.
Right now, this command works with a local GDM, modifying it, and running rectification
locally.
Over time, we will migrate how this command works whilst maintaining its interface and
semantics.
(The intention is that eventually 'sous deploy' will work by making an API call and
allowing the server to handle updating the GDM and rectifying.)

#### New flag `-flavor`

Actually, this is more than a flag, it affects the underlying data model, and the way
we think about how deployments are grouped.

Previously, Sous enabled at most a single deployment configuration per SourceLocation
per cluster. This model covers 90% of our use cases at OpenTable, but there are
exceptions.

We added "flavor" as a core concept in Sous, which allows multiple different deployment
configurations to be defined for a single codebase (SourceLocation) in each cluster. We
don't expect this feature to be used very much, but in those cases where configuration
needs to be more granular than per cluster, you now have that option available.

All commands that accept the `-source` flag (and/or the `-repo` and `-offset` flags) now
also accept the `-flavor` flag. Flavor is intended to be a short string made of
alphanumerics and possibly a hyphen, although we are not yet validating this string.
Each `-flavor` of a given `-source` is treated as a separate application, and has its
own manifest, allowing that application to be configured globally by a single manifest,
just like any other.

To create a new flavored application, you need to `sous init` with a `-flavor` flag. E.g.:

    sous init -flavor orange

From inside a repository would initiate a flavored manifest for that repo. Further calls
to `sous update`, `sous deploy`, etc, need to also specify the flavor `orange` to
work with that manifest. You can add an arbitrary number of flavors per SourceLocation,
and using a flavored manifest does not preclude you from also using a standard manifest
with no explicit flavor.

#### New flag `-source`

The `-source` flag is intended to be a replacement for the `-repo` and
`-offset` combination, for use in development environments. Note that we do not have
any plans to remove `-repo` and `-offset` since they may still be useful, especially
in scripting environments.

Source allows you to specify your SourceLocation in a single flag.
Source also performs additional validation,
ensuring that the source you pass can be handled by Sous end-to-end.
At present, that
means the repository must be a GitHub-based, in the form:

    github.com/<user>/<repo>

If your source code is not based in the root of the repository, you can add the offset
by separating it with a comma, e.g.:

    github.com/<user>/<repo>,<offset>

Because GitHub repository paths have a fixed format that Sous understands, you can
optionally use a slash instead of a comma, so the following is equivalent:

    github.com/<user>/<repo>/<offset>

(and offset can itself contain slashes if necessary, just like before).

### Documentation

We have made various documentation improvements, but there are definitely some that are
still out of date, which we will look to resolve in the coming weeks. Improvements made
in this release include:

- Better description of networking setup for Singularity deployments.
- Update to the deployment workflow documentation.
- Some fixes to the getting started document.

### Singularity request names

Up until now, Singularity request names looked something like this:

    github.comopentablereponameclustername

Which is not a great user experience, and has a large chance of causing naming collisions.
This version of Sous changes these names to use the form:

    <SourceLocation>:<Flavor>:<ClusterName>

E.g. for a simple repo with no offset or flavor, it looks like this:

    github.com>opentable>sous::cluster-name

With an offset and flavor, it expands to something like this:

    github.com>opentable>sous,offset:flavor-name:cluster-name


### Other

There have been numerous other small tweaks and fixes, mostly at code level to make our
own lives easier. We are also conscientiously working on improving test coverage, and this
cycle hit 54%, we expect to see that rise quickly now that we fail CI when it falls. You
can track test coverage at https://codecov.io/gh/opentable/sous.

For more gory detail, check out the [full list of commits between 0.0.1 and 0.1](https://github.com/opentable/sous/compare/v0.0.1...v0.1).
