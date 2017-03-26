gogenerate
==========
`gogenerate` is a tool that runs `go generate` in directories specified in the configuration. The configuration also
enables the specification of the files or directories that are expected to be produced by `go generate`, and can be run
in `verify` mode to verify whether any of the output files changed as a result of running the generator.

Usage
-----
Run `./gogenerate --config=generate.yml` to run the `go generate` command in the directories specified by the
configuration.

Run `./gogenerate --config=generate.yml --verify` to verify that running the `go generate` command for the specified
configuration did not change any of the files or directories specified by the configuration. If any of the matching
paths did change, the program prints the differences and exits with a non-0 exit code.

Configuration
-------------
The configuration file specifies the "generate" configurations, which consist of the relative path to the directory in
which `go generate` should be run and matcher configuration that specifies the paths or files that should be examined to
determine whether or not running `go generate` resulted in a change. All relative paths (both for the `go generate`
location and the matchers) are evaluated relative to the working directory of the command.

Here is an example configuration file:

```yml
generators:
  foo:
    go-generate-dir: gen
    gen-paths:
      paths:
        - "gen/output.txt"
```
