# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repository is

A native Go client for HDFS. It speaks the Hadoop protobuf RPC protocol directly to the namenode and the data-transfer protocol directly to datanodes — no JVM or Hadoop install is needed at runtime. It also ships the `hdfs` command-line client.

The module path is `github.com/colinmarc/hdfs/v2` (this checkout is the `wgzhao/hdfs-go` fork; upstream is `colinmarc/hdfs`). Use that module path for all internal imports. The protocol is HDFS "Version 9", so it targets Hadoop 2.2.x+ and 3.x.

## Build and test

Go 1.22+ is required (`go.mod`); CI uses the version declared there.

```bash
make                 # same as `make hdfs`: build the ./hdfs CLI
make test            # build CLI, run `go test -v -race -timeout 30s ./...`, then all bats CLI tests
make release         # package gohdfs-<tag>-<os>-<arch>.tar.gz for the host platform
```

The library and CLI are pure Go and cross-compile without cgo; CI verifies `darwin/amd64`, `darwin/arm64`, `windows/amd64`, and `windows/arm64`:

```bash
GOOS=windows GOARCH=amd64 go build ./...
GOOS=darwin  GOARCH=arm64 go build ./...
```

Note that the Makefile and shell scripts are Unix-oriented. On Windows, build directly with `go build ./cmd/hdfs` rather than `make`. The release workflow produces Linux and macOS `.tar.gz` archives and Windows `.zip` archives.

Important: the root `hdfs` package tests are end-to-end tests against a live HDFS cluster, not hermetic unit tests. They load `HADOOP_CONF_DIR`/`HADOOP_HOME` via `hadoopconf.LoadFromEnvironment` and hard-fail if no namenode is configured. They also expect the fixtures created by `.github/scripts/fixtures.sh` (`/_test/foo.txt`, `/_test/mobydick.txt`) and the users `gohdfs1`/`gohdfs2`.

```bash
# single integration test (still needs a live cluster + fixtures)
go test -race -run '^TestReadFile$' .

# packages that run without a cluster
go test ./hadoopconf ./internal/rpc ./internal/sasl ./internal/transfer

# one bats file after building the CLI
make hdfs && bats ./cmd/hdfs/test/ls.bats
```

Cluster setup used by CI (Linux, requires sudo):

- `.github/scripts/install-hdfs.sh` — downloads/starts Hadoop (matrix includes 3.3.6 and 3.4.3, with optional Kerberos, RPC protection, data-transfer protection, and AES). It writes logs to `/tmp/hdfs/{namenode,datanode}.log` and files test ccaches at `/tmp/krb5cc_gohdfs_<user>`.
- `.github/scripts/fixtures.sh` — populates `/_test`.
- `.github/scripts/install-bats.sh` — installs bats.

`minicluster.sh` is legacy (CDH 5 download, references a missing `./fixtures.sh`); prefer the CI scripts.

### Protobuf generation

`.pb.go` files under `internal/protocol/hadoop_common` and `internal/protocol/hadoop_hdfs` are generated from the vendored `.proto` files with `protoc` + `protoc-gen-go`. Do not hand-edit generated files. If a `.proto` changes:

```bash
make clean-protos   # deletes all *.pb.go
make                # regenerates via protoc, then builds
```

CI runs `find -name '*.pb.go' -exec touch {} \;` before `make` so a missing `protoc` does not trigger regeneration.

## Architecture

The code is layered; operations generally flow top → bottom:

1. **Public API (`package hdfs`, repository root).** `Client` (`client.go`) wraps one `rpc.NamenodeConnection`, carries `ClientOptions`, and caches server defaults and the data encryption key. It deliberately mirrors `os`: `Open`, `Create`, `Append`, `Stat`, `ReadDir`, `Mkdir`/`MkdirAll`, `Remove`/`RemoveAll`, `Rename`, `Walk`, `Chmod`, `Chown`, etc. `FileReader` implements `io.Reader`, `io.ReaderAt`, `io.Seeker`, `io.Closer`, and `Readdir*`; `FileWriter` implements buffered block writes (calling `Close` is mandatory); `FileInfo` implements `os.FileInfo`. There are HDFS-specific extras too: xattrs (`xattr.go`), snapshots (`snapshot.go`), `Truncate`, `StatFs`, `GetContentSummary`, and `Checksum`. Every operation builds a protobuf request and calls `c.namenode.Execute("<rpcMethod>", req, resp)`.

2. **`hadoopconf`** parses `core-site.xml`, `hdfs-site.xml`, and `mapred-site.xml` from `HADOOP_CONF_DIR`, else `$HADOOP_HOME/conf`. `ClientOptionsFromConf` translates those properties into `ClientOptions`: namenode addresses, datanode hostname usage, Kerberos settings, and data-transfer protection.

3. **`internal/rpc`** implements namenode RPC. Wire format: a 4-byte big-endian length followed by varint-prefixed `RpcRequestHeaderProto`, `RequestHeaderProto`, and the request message; the connection starts with the `hrpc` handshake and optional Kerberos SASL negotiation. `NamenodeConnection.Execute` serializes calls with a mutex, fails over across configured namenode addresses (backing off failed ones for 5s), retries only on `StandbyException`, and runs a background goroutine renewing file leases every second. `basicTransport` handles plain RPC; `saslTransport` handles `integrity`/`privacy` wrapping.

4. **`internal/transfer`** implements the datanode data-transfer protocol: read block (`0x51`), write block (`0x50`), and checksum block (`0x55`) ops, all framed with data-transfer version `0x1c`. `BlockReader` transparently fails over between replicas of a block, verifying CRC32/CRC32C checksums, and records failures in a process-global cache so later operations avoid bad nodes. `BlockWriter` constructs the write pipeline; it intentionally has **no** pipeline-recovery support yet (see the TODOs in `block_writer.go`), so a write failure can leave the writer unusable. `SaslDialer` wraps datanode connections for block-token auth and DIGEST-MD5, with integrity or AES-CTR privacy.

5. **`internal/sasl` + `internal/transfer/digest_md5*`** implement SASL challenge parsing and the DIGEST-MD5 handshake used for datanode authentication/encryption.

6. **`internal/protocol/*`** contains the vendored HDFS `.proto` files and generated Go code; the generated types are used directly throughout the layers above.

7. **`cmd/hdfs`** is the CLI. One file per command; flags use `pborman/getopt`; clients are cached per namenode. `paths.go` handles `hdfs://host:port/path` URLs, maps relative paths to `/user/<user>`, and expands globs. `bash_completion` and the hidden `complete` command provide shell completion.

### Error handling contract

Remote Java exceptions arrive as `*rpc.NamenodeError` (implementing the public `hdfs.Error` interface). `error.go` maps well-known exception classes to Go/stdlib errors — `os.ErrNotExist`, `os.ErrPermission`, `os.ErrExist`, `os.ErrInvalid`, and `syscall.ENOTEMPTY` — and public methods wrap failures in `*os.PathError` with an operation name, matching `os` semantics. When adding operations, follow this pattern; tests assert on `*os.PathError` equality and on `os.IsNotExist`/`os.IsPermission`.

`FileWriter.Close` may return `ErrReplicating` (wrapped in `*os.PathError`) when data is durable but the namenode hasn't closed the lease yet; tests treat this as benign (`ignoreErrReplicating`) and callers may retry `Close`.

## Testing conventions

- Root-package tests are integration/E2E tests against a real cluster; new filesystem behavior should be tested there (following the existing `getClient`/fixture helpers in `client_test.go`), not with mocks.
- Kerberos tests skip themselves when `/etc/krb5.conf` or the per-user ccache is unavailable.
- CLI tests are bats scripts in `cmd/hdfs/test/`; `helper.bash` defines the assertion helpers and points at the locally built `./hdfs`. Several compare behavior against `hadoop fs` via `HADOOP_FS`.
- Cluster-independent tests live in `hadoopconf/` and `internal/{rpc,sasl,transfer}`.
