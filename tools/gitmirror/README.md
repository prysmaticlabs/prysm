# Gitmirror Tool

This tool provides a Go server which listens for Github webhook events for **new releases** on Github repositories and mirrors specified directories to a separate, mirror repository. This is a useful internal tool for us to mirror certain directories in the Prysm monorepo into mirror repositories with more permissive licensing purposes, such as our API schemas and certain helper packages. 

## Usage

### Server

With Bazel
```bash
bazel run //tools/gitmirror
```

With Go
```bash
go build -o /tmp/gitmirror ./tools/gitmirror && /tmp/gitmirror
```

#### Parameters

The following are the required flags and secrets for the server:

**Flags**

| flag   | Description                                 | Default Value
| ------ | ------------------------------------------- | ------------- |
| --config | Path to a .yaml file configuring the gitmirror | ""
| --host |  Host for web server | "127.0.0.1"
| --port | Port for the web server | 3000

**Required Environment Variables**

| flag   | Description                                 
| ------ | ------------------------------------------- 
| GITHUB_WEBHOOK_SECRET | Secret used when creating a Github webhook
| GITHUB_PUSH_ACCESS_TOKEN |  Personal access token to push to the specified mirror repositories
| GITHUB_USER | Username for pushing to specified mirror repositories
| GITHUB_EMAIL | Username for pushing to specified mirror repositories

#### Configuration

You can configure the server by using a yaml configuration file. The schema of the configuration file is as follows:

```yaml
cloneBasePath: path where repositories will be cloned locally
repositories:
  - remoteUrl: url of the github remote we are listening to webhooks for
    remoteName: name of the github remote repository
    mirrorUrl: url of the github remote for the mirror we are pushing to
    mirrorName: name of the github remote repository
    mirrorDirectories: list of directories we want to mirror. If empty, entire repo will be mirrored
```

and running the server by specifying the path to the configuration file as follows:

```
bazel run //tools/gitmirror -- --config=/path/to/config.yaml
```

**Example Config File**

```yaml
cloneBasePath: tmp
repositories:
  - remoteUrl: git@github.com:prysmaticlabs/prysm.git
    remoteName: prysm
    mirrorUrl: git@github.com:prysmaticlabs/ethereumapis.git
    mirrorName: ethereumapis
    mirrorDirectories:
      - proto/eth
      - proto/p2p
```
